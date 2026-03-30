.PHONY: dev run test tidy infra-up infra-down infra-logs infra-init-postgis schema-sync

dev:
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
