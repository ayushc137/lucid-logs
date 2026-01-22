---
description: Database operations (start, stop, reset, migrate)
---

# Database Operations

Common SurrealDB database commands.

## Start Database

// turbo
```bash
docker compose -f deploy/docker-compose.yml up -d
```

## Stop Database

```bash
docker compose -f deploy/docker-compose.yml down
```

## Reset Database (DESTRUCTIVE!)

```bash
docker compose -f deploy/docker-compose.yml down -v
docker compose -f deploy/docker-compose.yml up -d
```

## Check Database Status

// turbo
```bash
docker compose -f deploy/docker-compose.yml ps
```

## Database Connection Info

- Host: localhost
- Port: 8000
- Namespace: daily_journal
- Database: core
- Credentials: See `.env` file

## Query the Database

Connect using SurrealDB CLI:
```bash
surreal sql --endpoint http://localhost:8000 --namespace daily_journal --database core --username root --password root
```

## View Schema

Reference file: `db/schema.surql`
