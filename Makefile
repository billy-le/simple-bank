network:
	docker network create bank-network

postgres:
	docker run --name postgres16.1 --network bank-network -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:16.1-alpine3.19

createdb:
	docker exec -it postgres16.1 createdb --username=root --owner=root simple_bank

dropdb:
	docker exec -it postgres16.1 dropdb simple_bank

migrateup:
	migrate -path db/migrations -database "postgres://root:secret@localhost:5432/simple_bank?sslmode=disable" --verbose up

migratedown:
	migrate -path db/migrations -database "postgres://root:secret@localhost:5432/simple_bank?sslmode=disable" --verbose down

sqlc:
	sqlc generate

test:
	go test -v -cover ./...

server:
	go run main.go

mockdb:
	mockgen -package mockdb -destination db/mock/store.go github.com/billy-le/simple-bank/db/sqlc Store

.PHONY: postgres createdb dropdb migrateup migratedown sqlc test server mockdb