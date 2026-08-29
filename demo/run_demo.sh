#!/usr/bin/env bash
# Good path: Goose automates SQL + complex Go code in ONE ordered pipeline.
#   STEP 1  Postgres source:  create app_user table (SQL) + seed from /users API (Go)
#   STEP 2  SQLite target:    create tables -> copy PG->SQLite (Go) -> fetch /posts (Go) -> rename column (SQL)
set -euo pipefail
cd "$(dirname "$0")/.."   # repo root

echo "================================================================"
echo " STEP 1 — Postgres (source): create app_user + seed from /users API"
echo "================================================================"
go run ./demo/runner pg up

echo
echo "================================================================"
echo " STEP 2 — SQLite (target): the actual PG->SQLite migration demo"
echo "================================================================"
go run ./demo/runner sqlite up

echo
echo "================================================================"
echo " STEP 3 — Result"
echo "================================================================"
echo "--- Postgres app_user (seeded from jsonplaceholder /users) ---"
go run ./demo/runner pg show
echo
echo "--- SQLite app_user (migrated from Postgres) ---"
go run ./demo/runner sqlite show
