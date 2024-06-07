package singleton

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/filecoin-project/go-jsonrpc"
	lotusapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/build"
)

var lotusAPIOnce sync.Once
var lotusArchiveAPIOnce sync.Once

type ChainOptions struct {
	DialAddr string
	Token    string
}

type LotusNode struct {
	Api    lotusapi.FullNodeStruct
	Closer jsonrpc.ClientCloser
}

var lotusClient *LotusNode

func ConnectLotus(opts ChainOptions) error {
	var connectionErr error

	if lotusClient != nil {
		log.Fatal("Lotus client already initialized")
	}

	lotusAPIOnce.Do(func() {
		// log.Printf("new lotus client: %s\n", opts.DialAddr)
		lotusClient = &LotusNode{}
		head := http.Header{}

		if opts.Token != "" {
			head.Set("Authorization", "Bearer "+opts.Token)
		}

		closer, err := jsonrpc.NewMergeClient(
			context.Background(),
			opts.DialAddr,
			"Filecoin",
			lotusapi.GetInternalStructs(&lotusClient.Api),
			head,
		)

		if err != nil {
			connectionErr = err
		}

		chainId, err := lotusClient.Api.EthChainId(context.Background())
		if err != nil {
			// default to mainnet
			chainId = 314
		}
		// log.Printf("connected to chain id: %v\n", chainId)

		if chainId != 314 {
			err = build.UseNetworkBundle("calibrationnet")
			log.Printf("use network bundle: %v\n", "calibrationnet")
			if err != nil {
				log.Fatalf("use network bundle error: %v\n", err)
			}
		}
		lotusClient.Closer = closer
	})

	return connectionErr
}

func ConnectArchiveLotus(opts ChainOptions) error {
	var connectionErr error

	if lotusClient != nil {
		log.Fatal("Lotus client already initialized")
	}

	lotusArchiveAPIOnce.Do(func() {
		// log.Printf("new lotus archive client: %s\n", opts.DialAddr)
		lotusClient = &LotusNode{}
		head := http.Header{}

		if opts.Token != "" {
			head.Set("Authorization", "Bearer "+opts.Token)
		}

		closer, err := jsonrpc.NewMergeClient(
			context.Background(),
			opts.DialAddr,
			"Filecoin",
			lotusapi.GetInternalStructs(&lotusClient.Api),
			head,
		)

		if err != nil {
			connectionErr = err
		}

		chainId, err := lotusClient.Api.EthChainId(context.Background())
		if err != nil {
			// default to mainnet
			chainId = 314
		}
		// log.Printf("connected to chain id: %v\n", chainId)

		if chainId != 314 {
			err = build.UseNetworkBundle("calibrationnet")
			log.Printf("use network bundle: %v\n", "calibrationnet")
			if err != nil {
				log.Fatalf("use network bundle error: %v\n", err)
			}
		}
		lotusClient.Closer = closer
	})

	return connectionErr
}

func Lotus() *LotusNode {
	return lotusClient
}

func (node *LotusNode) Close() {
	if node.Closer != nil {
		node.Closer()
	}
}
