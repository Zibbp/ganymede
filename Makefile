dev_setup:
	go install github.com/joho/godotenv/cmd/godotenv@latest
	go install github.com/air-verse/air@latest
	cd frontend && npm install --force

build_server:
	go build -ldflags='-X github.com/zibbp/ganymede/internal/utils.Commit=${GIT_SHA} -X github.com/zibbp/ganymede/internal/utils.Tag=${GIT_TAG} -X github.com/zibbp/ganymede/internal/utils.BuildTime=$(shell date -u "+%Y-%m-%d_%H:%M:%S")' -o ganymede-api cmd/server/main.go

build_worker:
	go build -ldflags='-X github.com/zibbp/ganymede/internal/utils.Commit=${GIT_SHA} -X github.com/zibbp/ganymede/internal/utils.Tag=${GIT_TAG} -X github.com/zibbp/ganymede/internal/utils.BuildTime=$(shell date -u "+%Y-%m-%d_%H:%M:%S")' -o ganymede-worker cmd/worker/main.go

build_dev_server:
	go build -ldflags='-X github.com/zibbp/ganymede/internal/utils.Commit=${GIT_SHA} -X github.com/zibbp/ganymede/internal/utils.Tag=${GIT_TAG} -X github.com/zibbp/ganymede/internal/utils.BuildTime=$(shell date -u "+%Y-%m-%d_%H:%M:%S")' -o ./tmp/server ./cmd/server/main.go

build_dev_worker:
	go build -ldflags='-X github.com/zibbp/ganymede/internal/utils.Commit=${GIT_SHA} -X github.com/zibbp/ganymede/internal/utils.Tag=${GIT_TAG} -X github.com/zibbp/ganymede/internal/utils.BuildTime=$(shell date -u "+%Y-%m-%d_%H:%M:%S")' -o ./tmp/worker ./cmd/worker/main.go

dev_server:
	rm -f ./tmp/server
	air -c ./.server.air.toml

dev_worker:
	rm -f ./tmp/worker
	air -c ./.worker.air.toml

dev_web:
	cd frontend && npm run dev

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
	go test -count=1 -timeout 30m -v ./...

web_update:
	cd frontend && npx npm-check-updates -i

river-tui:
	go install github.com/almottier/rivertui@latest
	@export $(shell grep -v '^#' .env | xargs) && \
	RIVER_DATABASE_URL=postgres://$$DB_USER:$$DB_PASS@$$DB_HOST:$$DB_PORT/$$DB_NAME rivertui --refresh 0.5s