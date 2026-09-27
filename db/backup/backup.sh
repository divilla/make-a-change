#!/bin/bash
set -euo pipefail

backup_file="changes_$(date +%Y-%m-%d_%H_%M_%S).sql.gz"
backup_tmp=$(mktemp ".${backup_file}.XXXXXX")
trap 'rm -f -- "$backup_tmp"' EXIT

pg_dump --clean --if-exists -U postgres -d changes -h localhost | gzip > "$backup_tmp"
mv -- "$backup_tmp" "$backup_file"
