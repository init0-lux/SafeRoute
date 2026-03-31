.PHONY: dev run test tidy infra-up infra-down infra-logs infra-init-postgis schema-sync
.PHONY: soroban-setup soroban-build soroban-deploy

	air -c .air.toml

run:
	go run ./cmd/api

test:
	go test ./...

tidy:
	go mod tidy

infra-up:
	docker compose up -d

infra-down:
	docker compose down

infra-logs:
	docker compose logs -f postgres redis

infra-init-postgis:
	docker compose exec -T postgres sh -lc 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -c "CREATE EXTENSION IF NOT EXISTS postgis;"'

schema-sync:
	go run ./cmd/schema-sync

# configure testnet and generate and fund relayer keypair
soroban-setup:
	docker run --rm -v $(PWD)/../contracts:/workspace -w /workspace stellar/stellar-cli:latest \
		sh -c "stellar network add --global testnet --rpc-url https://soroban-testnet.stellar.org:443 --network-passphrase 'Test SDF Network ; September 2015' && \
		stellar keys generate --global backend-relayer --network testnet --fund"

# compile the rust smart contract to WASM
soroban-build:
	docker run --rm -v $(PWD)/../contracts:/workspace -w /workspace rust:1.80 \
		sh -c "rustup target add wasm32-unknown-unknown && cargo build --target wasm32-unknown-unknown --release"

# deploy the built WASM contract to the testnet
soroban-deploy:
	docker run --rm -v $(PWD)/../contracts:/workspace -w /workspace stellar/stellar-cli:latest \
		stellar contract deploy --wasm target/wasm32-unknown-unknown/release/saferoute_contract.wasm --source backend-relayer --network testnet
