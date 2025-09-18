#!/usr/bin/env bash
set -euo pipefail

KEYSPACE_NAME=${SCYLLA_KEYSPACE_NAME:-gochat}
REPLICATION_CLASS=${SCYLLA_REPLICATION_CLASS:-SimpleStrategy}
REPLICATION_FACTOR=${SCYLLA_REPLICATION_FACTOR:-1}

cqlsh_host=${SCYLLA_CONTACT_POINT:-localhost}

until cqlsh "$cqlsh_host" -e "DESCRIBE KEYSPACES" >/dev/null 2>&1; do
  echo "Waiting for Scylla to become available..."
  sleep 5
done

echo "Ensuring keyspace $KEYSPACE_NAME exists"
cqlsh "$cqlsh_host" -e "CREATE KEYSPACE IF NOT EXISTS $KEYSPACE_NAME WITH replication = {'class': '$REPLICATION_CLASS', 'replication_factor': $REPLICATION_FACTOR};"
