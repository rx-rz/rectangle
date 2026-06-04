#!/bin/sh

set -eu

cd "$(dirname "$0")"

if [ $# -lt 1 ]; then
  echo "Usage: sh generate.sh <migration_name> [sql|go]" >&2
  exit 1
fi

name="$1"
type="${2:-sql}"

case "$type" in
  sql|go) ;;
  *)
    echo "Migration type must be either 'sql' or 'go'." >&2
    exit 1
    ;;
esac

set -a
. .env
set +a

goose -dir migrations postgres "$DB_URL" create "$name" "$type"
