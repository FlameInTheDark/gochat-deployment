#!/usr/bin/env sh
set -eu

if [ -z "${PG_ADDRESS:-}" ] && [ -z "${CASSANDRA_ADDRESS:-}" ]; then
  echo "PG_ADDRESS or CASSANDRA_ADDRESS must be provided" >&2
  exit 1
fi

# Install migrate tool with postgres and cassandra support
apk add --no-cache ca-certificates git build-base >/dev/null

go install -tags "postgres cassandra" github.com/golang-migrate/migrate/v4/cmd/migrate@latest

MIGRATE_BIN="$(go env GOPATH)/bin/migrate"

if [ -n "${PG_ADDRESS:-}" ]; then
  ${MIGRATE_BIN} -database "${PG_ADDRESS}" -path /migrations/postgres up
else
  echo "Skipping PostgreSQL migrations because PG_ADDRESS is empty"
fi

if [ -n "${CASSANDRA_ADDRESS:-}" ]; then
  ${MIGRATE_BIN} -database "${CASSANDRA_ADDRESS}" -path /migrations/cassandra up
else
  echo "Skipping Scylla migrations because CASSANDRA_ADDRESS is empty"
fi
