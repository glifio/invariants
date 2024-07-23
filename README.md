invariants
==========

Compares latest data from [pools-events](https://github.com/glifio/pools-events) API
and checks that it matches on-chain data from a Lotus node.

# Usage

Copy `mainnet.env.sample` to `mainnet.env` and update entries.

```
$ go run ./cmd/...
Checks values from REST API against Lotus node values

Usage:
  invariants [command]

Available Commands:
  agent-balances    Compare the balances from the API and the node for an agent
  agent-econ        Compare the econ values from the API and the node for an agent
  completion        Generate the autocompletion script for the specified shell
  help              Help about any command
  ifil-total-supply Compare the iFIL Total Supply from the API and the node
  metrics           Compare the metrics from the API and the node at height
  miner-liquidation Compare liquidation values computed using various methods

Flags:
      --archive         use archive Lotus node (default true)
      --config string   config file (default is ./mainnet.env) (default "mainnet")
  -h, --help            help for invariants

Use "invariants [command] --help" for more information about a command.
```

# License

Proprietary
