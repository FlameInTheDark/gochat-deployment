#!/usr/bin/env sh
set -eu

if [ -z "${PG_ADDRESS:-}" ] && [ -z "${CASSANDRA_ADDRESS:-}" ]; then
  echo "PG_ADDRESS or CASSANDRA_ADDRESS must be provided" >&2
  exit 1
fi

REPO_URL="${GOCHAT_MIGRATIONS_REPO:-https://github.com/FlameInTheDark/gochat.git}"
BRANCH="${GOCHAT_MIGRATIONS_BRANCH:-main}"

apk add --no-cache ca-certificates git build-base >/dev/null

WORKDIR="$(mktemp -d)"
trap 'rm -rf "${WORKDIR}"' EXIT INT TERM

echo "Cloning ${REPO_URL} (${BRANCH}) for migrations..."
git clone --depth 1 --branch "${BRANCH}" "${REPO_URL}" "${WORKDIR}/gochat"

# Install migrate tool with postgres and cassandra support
go install -tags "postgres cassandra" github.com/golang-migrate/migrate/v4/cmd/migrate@latest

MIGRATE_BIN="$(go env GOPATH)/bin/migrate"
MIGRATIONS_ROOT="${WORKDIR}/gochat/db"

if [ -n "${PG_ADDRESS:-}" ]; then
  if [ ! -d "${MIGRATIONS_ROOT}/postgres" ]; then
    echo "PostgreSQL migrations not found at ${MIGRATIONS_ROOT}/postgres" >&2
    exit 1
  fi
  echo "Running PostgreSQL migrations against ${PG_ADDRESS}"
  ${MIGRATE_BIN} -database "${PG_ADDRESS}" -path "${MIGRATIONS_ROOT}/postgres" up
else
  echo "Skipping PostgreSQL migrations because PG_ADDRESS is empty"
fi

if [ -n "${CASSANDRA_ADDRESS:-}" ]; then
  if [ ! -d "${MIGRATIONS_ROOT}/cassandra" ]; then
    echo "Scylla migrations not found at ${MIGRATIONS_ROOT}/cassandra" >&2
    exit 1
  fi
  echo "Running Scylla migrations against ${CASSANDRA_ADDRESS}"
  ${MIGRATE_BIN} -database "${CASSANDRA_ADDRESS}" -path "${MIGRATIONS_ROOT}/cassandra" up
else
  echo "Skipping Scylla migrations because CASSANDRA_ADDRESS is empty"
fi

