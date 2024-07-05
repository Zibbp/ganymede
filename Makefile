dev_server:
	rm -f ./tmp/server
	air -c ./.server.air.toml

dev_worker:
	rm -f ./tmp/worker
	air -c ./.worker.air.toml

ent_generate:
	go run -mod=mod entgo.io/ent/cmd/ent generate --feature sql/upsert ./ent/schema

go_update_packages:
	go get -u ./... && go mod tidy

river-webui:
	curl -L https://github.com/riverqueue/riverui/releases/latest/download/riverui_linux_amd64.gz | gzip -d > /tmp/riverui
	chmod +x /tmp/riverui
	@export $(shell grep -v '^#' .env | xargs) && \
	VITE_RIVER_API_BASE_URL=http://localhost:8080/api DATABASE_URL=postgres://$$DB_USER:$$DB_PASS@dev.tycho:$$DB_PORT/$$DB_NAME /tmp/riverui