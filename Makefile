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

migrateup1:
	migrate -path db/migrations -database "$(DB_URL)" --verbose up 1

migratedown:
	migrate -path db/migrations -database "$(DB_URL)" --verbose down

migratedown1:
	migrate -path db/migrations -database "$(DB_URL)" --verbose down 1

new_migration:
	migrate create -ext sql -dir db/migrations -seq $(name)

sqlc:
	sqlc generate

test:
	go test -v -cover -short ./...

server:
	go run main.go

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/billy-le/simple-bank/db/sqlc Store
	mockgen -package mockwk -destination worker/mock/distributor.go github.com/billy-le/simple-bank/worker TaskDistributor

dbdocs:
	dbdocs build docs/db.dbml

db_schema:
	dbml2sql --postgres -o docs/schema.sql docs/db.dbml

proto:
	rm -f pb/*.go
	rm -f docs/swagger/*.swagger.json
	protoc --proto_path=proto --go_out=pb --go_opt=paths=source_relative \
    --go-grpc_out=pb --go-grpc_opt=paths=source_relative \
	--grpc-gateway_out=pb --grpc-gateway_opt=paths=source_relative \
	--openapiv2_out=docs/swagger --openapiv2_opt=allow_merge=true,merge_file_name=simple_bank \
    proto/*.proto
	statik -src=./docs/swagger -dest=./docs

evans:
	evans --host localhost --port 9090 -r repl

redis:
	docker run --name redis -p 6379:6379 -d redis:7-alpine

.PHONY: postgres createdb dropdb migrateup migratedown sqlc test server mock dbdocs db_schema proto evans migrateup1 migratedown1 new_migration