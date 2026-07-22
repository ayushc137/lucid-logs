#!/bin/bash
# Seed the live lucid-logs DB using a temp copy + in-process API approach.
# libSQL takes an exclusive file lock, so backend and seeder can't share a file.
# Strategy: seed a copy of the live DB with its own API instance, then swap back.
set -e
cd "$(dirname "$0")"
export PATH=/usr/local/go/bin:$PATH

LIVE_DB=/tmp/lucid-live.db
SEED_DB=/tmp/lucid-seed.db
SEED_PORT=8081
JWT_SECRET=$(grep '^JWT_SECRET=' ~/repos/lucid-logs/.env | cut -d= -f2)

echo "==> Copying live DB to $SEED_DB"
cp "$LIVE_DB" "$SEED_DB"
rm -f "$SEED_DB-wal" "$SEED_DB-shm"

echo "==> Building seed-api + seeder"
go build -o /tmp/lucid-seed-api ./cmd/api
go build -o /tmp/lucid-seed ./cmd/seed

echo "==> Pre-setting WAL mode on seed DB (no other process may hold it)"
lsof -ti:8080 | xargs -r kill; sleep 1
python3 - "$SEED_DB" <<'EOF'
import sqlite3, sys
db = sqlite3.connect(sys.argv[1], timeout=10)
print("journal_mode:", db.execute("PRAGMA journal_mode=WAL").fetchone())
db.close()
EOF

echo "==> Starting temp API on :$SEED_PORT against seed DB"
DATABASE_PATH=$SEED_DB JWT_SECRET=$JWT_SECRET HTTP_PORT=$SEED_PORT /tmp/lucid-seed-api > /tmp/lucid-seed-api.log 2>&1 &
API_PID=$!
trap "kill $API_PID 2>/dev/null || true" EXIT

for i in $(seq 1 20); do
  if curl -s -m 2 -o /dev/null http://127.0.0.1:$SEED_PORT/api/v1/health; then break; fi
  sleep 1
done

echo "==> Running seeder (API on $SEED_PORT, DB $SEED_DB)"
# config.Load reads HTTP_PORT + DATABASE_PATH env
DATABASE_PATH=$SEED_DB JWT_SECRET=$JWT_SECRET HTTP_PORT=$SEED_PORT /tmp/lucid-seed "$@"

echo "==> Stopping temp API"
kill $API_PID 2>/dev/null || true
wait $API_PID 2>/dev/null || true
trap - EXIT

# Checkpoint WAL into main file before swap
echo "==> Checkpointing WAL"
python3 - "$SEED_DB" <<'EOF'
import sqlite3, sys
db = sqlite3.connect(sys.argv[1], timeout=10)
print(db.execute("PRAGMA wal_checkpoint(TRUNCATE)").fetchone())
db.close()
EOF

echo "==> Stopping live backend on :8080"
lsof -ti:8080 | xargs -r kill
sleep 1

echo "==> Swapping seeded DB into place"
cp "$SEED_DB" "$LIVE_DB"
rm -f "$LIVE_DB-wal" "$LIVE_DB-shm"

echo "==> Restarting live backend"
cd ~/repos/lucid-logs/apps/go_backend
DATABASE_PATH=$LIVE_DB JWT_SECRET=$JWT_SECRET HTTP_PORT=8080 nohup /tmp/lucid-api-turso > /tmp/lucid-backend.log 2>&1 &
sleep 3
curl -s -m 5 -o /dev/null -w "backend health: %{http_code}\n" -X POST http://localhost:8080/api/v1/auth/login -H "Content-Type: application/json" -d '{"username":"admin@example.com","password":"adminadmin"}'
echo "==> Done. Seeded DB is live."
