# Build stage
FROM golang:1.21.6-alpine3.19 AS builder
WORKDIR /app
COPY . .
RUN go build -o main main.go

# Run stage
FROM alpine:3.19
ENV DB_DRIVER=postgres
ENV DB_SOURCE=postgres://root:secret@localhost:5432/simple_bank?sslmode=disable
ENV SERVER_ADDRESS=0.0.0.0:8080
ENV TOKEN_SYMMETRIC_KEY=12345678901234567890123456789012
ENV ACCESS_TOKEN_DURATION=15m
WORKDIR /app
COPY --from=builder /app/main .
COPY start.sh .
COPY app.env .
COPY wait-for.sh .
COPY db/migrations ./db/migrations

EXPOSE 8080
CMD [ "/app/main" ]
ENTRYPOINT [ "/app/start.sh" ]