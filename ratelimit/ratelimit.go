// Package ratelimit installs a client-side token-bucket cap on outbound
// chain RPCs so the tool stays under the provider's requests-per-minute
// budget (chain.love: 1000/min) instead of blasting until it gets 429s.
//
// Two RPC client stacks are in play and they use different transports:
//
//   - go-ethereum's ethclient (all eth_calls) builds zero-value
//     http.Clients, which fall back to http.DefaultTransport. Install
//     wraps http.DefaultTransport, so every eth client in the process is
//     covered without touching go-pools.
//   - filecoin go-jsonrpc (Lotus RPCs) has its own private http.Client,
//     so wrapping the default transport does NOT cover it. Those
//     connections must be built with jsonrpc.WithHTTPClient(HTTPClient())
//     — see singleton/lotus.go.
//
// Both paths share ONE limiter, matching the provider's single budget.
// Requests to hosts other than the configured RPC hosts (events API,
// Discord webhooks, ...) pass through untouched.
package ratelimit

import (
	"net/http"
	"net/url"
	"sync"

	"golang.org/x/time/rate"
)

var (
	mu      sync.Mutex
	limiter *rate.Limiter
	hosts   map[string]struct{}
	client  *http.Client
)

// transport applies the shared limiter to requests bound for the
// configured RPC hosts and passes everything else straight through.
type transport struct {
	base http.RoundTripper
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	mu.Lock()
	lim := limiter
	_, limited := hosts[req.URL.Host]
	mu.Unlock()
	if lim != nil && limited {
		if err := lim.Wait(req.Context()); err != nil {
			return nil, err
		}
	}
	return t.base.RoundTrip(req)
}

// Install activates the limiter at perMinute requests/min against the
// hosts of the given RPC URLs, and wraps http.DefaultTransport so all
// default-transport HTTP clients (ethclient et al.) are covered.
// Safe to call once at process startup, before clients dial.
func Install(perMinute int, rpcURLs ...string) {
	if perMinute <= 0 {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if limiter != nil {
		return
	}
	hosts = make(map[string]struct{})
	for _, raw := range rpcURLs {
		if u, err := url.Parse(raw); err == nil && u.Host != "" {
			hosts[u.Host] = struct{}{}
		}
	}
	// Steady rate perMinute/60 with a small burst: worst-case in any
	// sliding 60s window is perMinute + burst, so keep the sum under
	// the provider's cap (e.g. 900 + 30 < 1000).
	limiter = rate.NewLimiter(rate.Limit(float64(perMinute)/60.0), 30)
	if _, ok := http.DefaultTransport.(*transport); !ok {
		http.DefaultTransport = &transport{base: http.DefaultTransport}
	}
	client = &http.Client{Transport: http.DefaultTransport}
}

// HTTPClient returns a client that shares the limiter, for RPC
// libraries that don't use http.DefaultTransport (go-jsonrpc). Before
// Install (or if Install was skipped) it returns nil, which
// jsonrpc.WithHTTPClient callers must treat as "use the default".
func HTTPClient() *http.Client {
	mu.Lock()
	defer mu.Unlock()
	return client
}
