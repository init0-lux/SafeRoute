.PHONY: dev run test tidy infra-up infra-down infra-logs

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
