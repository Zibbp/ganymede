dev_server:
	rm ./tmp/server
	air -c ./.server.air.toml

dev_worker:
	rm ./tmp/worker
	air -c ./.worker.air.toml

ent_generate:
	go run -mod=mod entgo.io/ent/cmd/ent generate --feature sql/upsert ./ent/schema

go_update_packages:
	go get -u ./... && go mod tidy
