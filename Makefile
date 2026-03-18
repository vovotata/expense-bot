.PHONY: build build-watcher run run-watcher test lint migrate docker-up docker-down sqlc generate

build:
	go build -o bin/bot ./cmd/bot
build-watcher:
	go build -o bin/email-watcher ./cmd/email-watcher
run:
	go run ./cmd/bot
run-watcher:
	go run ./cmd/email-watcher
test:
	go test -v -race -cover ./...
lint:
	golangci-lint run ./...
migrate:
	goose -dir migrations postgres "$(DATABASE_URL)" up
sqlc:
	sqlc generate
docker-up:
	docker compose up -d --build
docker-down:
	docker compose down
generate: sqlc
	go generate ./...
