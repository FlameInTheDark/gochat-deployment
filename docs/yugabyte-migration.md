# Citus To YugabyteDB Migration

This runbook is for applying the YugabyteDB deployment changes to an existing GoChat environment without data loss. Do not delete Citus pods, PVCs, Compose volumes, or dumps until export, import, verification, and smoke tests all pass.

References:

- YugabyteDB Docker quick start: https://docs.yugabyte.com/stable/quick-start/docker/
- YugabyteDB Kubernetes Helm chart: https://docs.yugabyte.com/stable/deploy/kubernetes/single-zone/oss/helm-chart/
- YugabyteDB Voyager offline migration: https://docs.yugabyte.com/stable/yugabyte-voyager/migrate/migrate-steps/

## Design Decision

GoChat targets very high write and read volume. Use YugabyteDB as a distributed YSQL database with `COLOCATION=false`.

Colocated databases are useful for many small tables because colocated tables share one tablet. GoChat guild, channel, membership, role, invite, settings, and auth tables are expected to grow and should be able to split and rebalance independently. The deployment therefore creates the target database with:

```sql
CREATE DATABASE gochat WITH COLOCATION = false;
```

Guild-level query locality should be handled through schema/index design and query predicates rather than database-level colocation. Keep guild-scoped tables indexed by `guild_id`, and prefer future schema work that makes `guild_id` a leading key where query patterns need it.

## Local Compose

Render or export the deployment workspace, then start the active stack:

```bash
docker compose up -d yugabyte yugabyte-init scylla migrations
```

YugabyteDB local endpoints:

- YSQL: `yugabyte:5433`, host port `${YUGABYTE_YSQL_PORT:-5433}`
- UI: `http://localhost:${YUGABYTE_UI_PORT:-15433}`
- data volume: `yugabyte-data`

Legacy Citus is available only through the explicit profile:

```bash
docker compose --profile legacy-citus up -d citus-master citus-init
```

Do not run `docker compose down -v` during migration. It removes volumes.

## Kubernetes Preparation

Install YugabyteDB separately with the official YugabyteDB Helm chart. The GoChat chart defaults assume:

- YugabyteDB namespace: `gochat-yb`
- YugabyteDB release: `yb`
- YSQL service: `yb-tservers.gochat-yb.svc.cluster.local`
- YSQL port: `5433`

Before installing YugabyteDB, prepare every Kubernetes node that can run `yb-master` or `yb-tserver`:

```bash
# If the VM disk was expanded in the hypervisor and the node uses the default
# Ubuntu LVM layout, grow the partition, PV, LV, and ext4 filesystem online.
growpart /dev/sda 3
pvresize /dev/sda3
lvextend -r -l +100%FREE /dev/ubuntu-vg/ubuntu-lv

# YugabyteDB's Kubernetes wrapper expects a file-based core pattern. A pipe
# pattern such as Ubuntu Apport causes repeated k8s_parent.py core-copy errors.
printf 'kernel.core_pattern=core.%%e.%%p.%%t\n' >/etc/sysctl.d/99-yugabyte-core.conf
sysctl -w kernel.core_pattern='core.%e.%p.%t'
```

Example development-sized install for the current small cluster:

```bash
kubectl create namespace gochat-yb
helm repo add yugabytedb https://charts.yugabyte.com
helm repo update
helm upgrade --install yb yugabytedb/yugabyte \
  --version 2025.2.2 \
  --namespace gochat-yb \
  --set replicas.master=3,replicas.tserver=3,enableLoadBalancer=False \
  --set storage.ephemeral=false \
  --set storage.master.size=100Gi,storage.tserver.size=100Gi \
  --set livenessProbe.timeoutSeconds=10
```

For production, size CPU, memory, storage class, zones, replication factor, and node placement explicitly before the maintenance window. Prefer dedicated fast disks and a storage class sized for the expected WAL/SST growth; do not treat a small `local-path` test volume as a production storage profile.

## Offline Cutover

1. Stop write-capable GoChat services: API, auth, attachments, search, ws, webhook, and any mutation-capable jobs.
2. Confirm Citus is still running and healthy.
3. Take a source backup and store it outside the repo:

```bash
pg_dump "postgres://postgres:<password>@<citus-host>:5432/gochat?sslmode=disable" \
  --format=custom \
  --file gochat-citus-before-yugabyte.dump
```

4. Assess with Voyager:

```bash
yb-voyager assess-migration \
  --source-db-type postgresql \
  --source-db-host <citus-host> \
  --source-db-port 5432 \
  --source-db-user postgres \
  --source-db-password '<password>' \
  --source-db-name gochat \
  --export-dir ./voyager-export-gochat
```

5. Apply GoChat Helm values that point to YugabyteDB YSQL, but keep `citus.enabled=true`.
6. Let the `gochat-yugabyte-init` hook create/check the target database.
7. Run GoChat migrations with `MIGRATION_SCOPE=all` or `MIGRATION_SCOPE=yugabyte` plus Scylla migration as appropriate.
8. Export Citus data:

```bash
yb-voyager export data \
  --source-db-type postgresql \
  --source-db-host <citus-host> \
  --source-db-port 5432 \
  --source-db-user postgres \
  --source-db-password '<password>' \
  --source-db-name gochat \
  --export-dir ./voyager-export-gochat
```

9. Import into YugabyteDB:

```bash
yb-voyager import data \
  --target-db-host <yb-tserver-service> \
  --target-db-port 5433 \
  --target-db-user yugabyte \
  --target-db-password '<password>' \
  --target-db-name gochat \
  --export-dir ./voyager-export-gochat
```

10. Check Voyager status:

```bash
yb-voyager get data-migration-report --export-dir ./voyager-export-gochat
yb-voyager import data status --export-dir ./voyager-export-gochat
```

11. Verify with app tooling:

```bash
gctools yugabyte verify \
  --source-dsn "postgres://postgres:<password>@<citus-host>:5432/gochat?sslmode=disable" \
  --target-dsn "postgres://yugabyte:<password>@<yb-host>:5433/gochat?sslmode=disable"
```

Expected validation:

- every migrated table reports matching row counts,
- deterministic checksums match where supported,
- `schema_migrations` version matches the application release,
- invite PL/pgSQL functions exist and run,
- auth, API, search, attachments, websocket, and webhook smoke tests pass.

## Rollback

Rollback remains Citus-first until verification is complete:

1. Keep Citus StatefulSets/PVCs or Compose `citus-data` mounted and untouched.
2. Keep the pre-cutover `pg_dump` outside tracked files.
3. If Yugabyte verification fails, keep write-capable GoChat services stopped, point app configs back to Citus, and restart the previously running release.
4. Investigate using the Voyager export directory and dump. Do not overwrite the source.

Only after successful import, verification, smoke tests, and operator sign-off should you render with `--disable-legacy-citus` or set `citus.enabled=false`.
