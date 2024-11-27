dev_setup:
	go install github.com/joho/godotenv/cmd/godotenv@latest
	go install github.com/air-verse/air@latest

build_server:
	go build -ldflags='-X github.com/zibbp/ganymede/internal/utils.Commit=$(shell git rev-parse HEAD) -X github.com/zibbp/ganymede/internal/utils.BuildTime=$(shell date -u "+%Y-%m-%d_%H:%M:%S")' -o ganymede-api cmd/server/main.go

build_worker:
	go build -ldflags='-X github.com/zibbp/ganymede/internal/utils.Commit=$(shell git rev-parse HEAD) -X github.com/zibbp/ganymede/internal/utils.BuildTime=$(shell date -u "+%Y-%m-%d_%H:%M:%S")' -o ganymede-worker cmd/worker/main.go

build_dev_server:
	go build -ldflags='-X github.com/zibbp/ganymede/internal/utils.Commit=$(shell git rev-parse HEAD) -X github.com/zibbp/ganymede/internal/utils.BuildTime=$(shell date -u "+%Y-%m-%d_%H:%M:%S")' -o ./tmp/server ./cmd/server/main.go

build_dev_worker:
	go build -ldflags='-X github.com/zibbp/ganymede/internal/utils.Commit=$(shell git rev-parse HEAD) -X github.com/zibbp/ganymede/internal/utils.BuildTime=$(shell date -u "+%Y-%m-%d_%H:%M:%S")' -o ./tmp/worker ./cmd/worker/main.go

dev_server:
	rm -f ./tmp/server
	air -c ./.server.air.toml

dev_worker:
	rm -f ./tmp/worker
	air -c ./.worker.air.toml

ent_generate:
	go run -mod=mod entgo.io/ent/cmd/ent generate --feature sql/upsert ./ent/schema

ent_new_schema:
	@read -p "Enter schema name:" schema; \
	go run -mod=mod entgo.io/ent/cmd/ent new $$schema

go_update_packages:
	go get -u ./... && go mod tidy

lint:
	golangci-lint run

test:
	go test -v ./...

river-webui:
	curl -L https://github.com/riverqueue/riverui/releases/latest/download/riverui_linux_amd64.gz | gzip -d > /tmp/riverui
	chmod +x /tmp/riverui
	@export $(shell grep -v '^#' .env | xargs) && \
	VITE_RIVER_API_BASE_URL=http://localhost:8080/api DATABASE_URL=postgres://$$DB_USER:$$DB_PASS@$$DB_HOST:$$DB_PORT/$$DB_NAME /tmp/riverui