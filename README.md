invariants
==========

Compares latest data from [pools-events](https://github.com/glifio/pools-events) API
and checks that it matches on-chain data from a Lotus node.

# Usage

`go test`.

# Required environment variables

* `EVENTS_API` - URL for pools-events API endpoint
* `CHAIN_ID` - set to 314 for mainnet
* `LOTUS_PRIVATE_ADDR` - eg. http://127.0.0.1:1234/rpc/v1
* `LOTUS_PRIVATE_TOKEN` - Lotus JWT

# License

Proprietary

