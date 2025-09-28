#!/usr/bin/env sh
set -eu

echo "== env =="
env | sort

echo "== ls -l /migrations =="
ls -l /migrations || true

DATABASE_URL="mysql://${MYSQL_USER}:${MYSQL_PASSWORD}@tcp(nutfesmap-database:3306)/${MYSQL_DATABASE}?multiStatements=true"
echo "DATABASE_URL=${DATABASE_URL}"

echo "== migrate up (verbose) =="
set +e
migrate -verbose -path /migrations -database "${DATABASE_URL}" up
code=$?
set -e

if [ "${code}" = "0" ] || [ "${code}" = "2" ]; then
  echo "migrations applied (exit ${code})"
  exit 0
else
  echo "migration failed (exit ${code})"
  exit "${code}"
fi
