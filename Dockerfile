FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY . .

RUN go mod download

RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache bash postgresql-client

COPY --from=builder /app/server .
COPY --from=builder /go/bin/migrate ./migrate
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/config.yaml .
COPY entrypoint.sh ./entrypoint.sh

RUN chmod +x ./entrypoint.sh

EXPOSE 8080

CMD ["./entrypoint.sh"]
