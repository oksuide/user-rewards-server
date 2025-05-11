#!/bin/sh

echo "Waiting for PostgreSQL..."

until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER"; do
  sleep 1
done

echo "PostgreSQL is available. Applying migrations..."


./migrate -path ./migrations -database "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable" up

echo "Starting server..."
./server
