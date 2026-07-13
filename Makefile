# glifio/invariants build helpers.

.PHONY: build abigen forge-build forge-clean docker-build docker-push

# Default: build the CLI.
build:
	go build -o invariants ./cmd/invariants

# Build the container image, tagged with the current git SHA.
docker-build:
	git rev-parse HEAD
	docker build --no-cache --tag inv --tag "gcr.io/glif-370419/inv:$$(git rev-parse HEAD)" .

# Push the current SHA-tagged image to GCR.
docker-push:
	git rev-parse HEAD
	docker push "gcr.io/glif-370419/inv:$$(git rev-parse HEAD)"

# Compile the InvariantsQuery contract via Foundry.
forge-build:
	cd contracts && forge build

forge-clean:
	cd contracts && forge clean

# Regenerate Go bindings for InvariantsQuery from the Foundry artifact.
# Requires: forge (foundry), abigen (go-ethereum). Install:
#   curl -L https://foundry.paradigm.xyz | bash && foundryup
#   go install github.com/ethereum/go-ethereum/cmd/abigen@latest
abigen: forge-build
	@mkdir -p abigen
	jq -r '.abi' contracts/out/InvariantsQuery.sol/InvariantsQuery.json > /tmp/InvariantsQuery.abi
	abigen --abi /tmp/InvariantsQuery.abi --pkg abigen --type InvariantsQuery \
	  --out abigen/InvariantsQuery.go
	@rm -f /tmp/InvariantsQuery.abi
	@echo "regenerated abigen/InvariantsQuery.go"
