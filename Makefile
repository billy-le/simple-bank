DB_URL=postgres://root:secret@localhost:5432/simple_bank?sslmode=disable

network:
	docker network create bank-network

postgres:
	docker run --name postgres16.1 --network bank-network -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:16.1-alpine3.19

createdb:
	docker exec -it postgres16.1 createdb --username=root --owner=root simple_bank

dropdb:
	docker exec -it postgres16.1 dropdb simple_bank

migrateup:
	migrate -path db/migrations -database "$(DB_URL)" --verbose up

migratedown:
	migrate -path db/migrations -database "$(DB_URL)" --verbose down

sqlc:
	sqlc generate

test:
	go test -v -cover ./...

server:
	go run main.go

mockdb:
	mockgen -package mockdb -destination db/mock/store.go github.com/billy-le/simple-bank/db/sqlc Store

dbdocs:
	dbdocs build docs/db.dbml

db_schema:
	dbml2sql --postgres -o docs/schema.sql docs/db.dbml

proto:
	rm -f pb/*.go
	protoc --proto_path=proto --go_out=pb --go_opt=paths=source_relative \
    --go-grpc_out=pb --go-grpc_opt=paths=source_relative \
    proto/*.proto

evans:
	evans --host localhost --port 9090 -r repl

.PHONY: postgres createdb dropdb migrateup migratedown sqlc test server mockdb dbdocs db_schema proto evans