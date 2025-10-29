#!/usr/bin/env sh
set -eu

if [ -z "${PG_ADDRESS:-}" ] && [ -z "${CASSANDRA_ADDRESS:-}" ]; then
  echo "PG_ADDRESS or CASSANDRA_ADDRESS must be provided" >&2
  exit 1
fi

REPO_URL="${GOCHAT_MIGRATIONS_REPO:-https://github.com/FlameInTheDark/gochat.git}"
BRANCH="${GOCHAT_MIGRATIONS_BRANCH:-main}"

apk add --no-cache ca-certificates git build-base >/dev/null

retry_command() {
  max_attempts="$1"
  shift
  delay_seconds="$1"
  shift

  attempt=1
  while [ "$attempt" -le "$max_attempts" ]; do
    if "$@"; then
      return 0
    fi

    if [ "$attempt" -lt "$max_attempts" ]; then
      echo "Command failed (attempt ${attempt}/${max_attempts}), retrying in ${delay_seconds}s..."
      sleep "$delay_seconds"
    fi

    attempt=$((attempt + 1))
  done

  echo "Command failed after ${max_attempts} attempts" >&2
  return 1
}

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
  retry_command "${PG_RETRY_ATTEMPTS:-30}" "${PG_RETRY_DELAY_SECONDS:-5}" \
    "${MIGRATE_BIN}" -database "${PG_ADDRESS}" -path "${MIGRATIONS_ROOT}/postgres" up
else
  echo "Skipping PostgreSQL migrations because PG_ADDRESS is empty"
fi

if [ -n "${CASSANDRA_ADDRESS:-}" ]; then
  if [ ! -d "${MIGRATIONS_ROOT}/cassandra" ]; then
    echo "Scylla migrations not found at ${MIGRATIONS_ROOT}/cassandra" >&2
    exit 1
  fi
  echo "Running Scylla migrations against ${CASSANDRA_ADDRESS}"
  retry_command "${CASSANDRA_RETRY_ATTEMPTS:-30}" "${CASSANDRA_RETRY_DELAY_SECONDS:-5}" \
    "${MIGRATE_BIN}" -database "${CASSANDRA_ADDRESS}" -path "${MIGRATIONS_ROOT}/cassandra" up
else
  echo "Skipping Scylla migrations because CASSANDRA_ADDRESS is empty"
fi

