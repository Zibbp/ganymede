prereq_command:
	set -o allexport; source .env.dev; set +o allexport;

dev_server: prereq_command
	go run cmd/server/main.go

dev_worker: prereq_command
	go run cmd/worker/main.go

ent_generate:
	go run -mod=mod entgo.io/ent/cmd/ent generate --feature sql/upsert ./ent/schema