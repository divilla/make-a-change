#!/bin/bash
set -euo pipefail

archive=${1:?Usage: restore.sh archive.sql.gz}
restore_sql=$(mktemp "${TMPDIR:-/tmp}/changes-restore.XXXXXX")
trap 'rm -f -- "$restore_sql"' EXIT

# Finish decompression before sending any SQL to the database.
gunzip < "$archive" > "$restore_sql"
psql --echo-errors -h localhost -U postgres -X -d changes \
    --single-transaction -v ON_ERROR_STOP=1 -f "$restore_sql"
