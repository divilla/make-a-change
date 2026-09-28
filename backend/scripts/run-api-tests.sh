#!/usr/bin/env bash
# Temporary legacy HTTP runner. APIHydra replaces this during the planned P0 pass.
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."

case "$API_TEST_DB_URL" in
    postgres://*/changes_test|postgres://*/changes_test\?*|postgresql://*/changes_test|postgresql://*/changes_test\?*) ;;
    *) echo 'API_TEST_DB_URL must target changes_test' >&2; exit 1 ;;
esac
# A libpq URI query can override the pathname database. Disallow that ambiguity.
if [[ $API_TEST_DB_URL == *\?* ]]; then
    query=${API_TEST_DB_URL#*\?}
    if [[ $query != sslmode=disable && $query != sslmode=require ]]; then
        echo 'API_TEST_DB_URL query must be sslmode=disable or sslmode=require' >&2
        exit 1
    fi
fi

run_dir=$(mktemp -d "${TMPDIR:-/tmp}/mch-api-test.XXXXXXXX")
server_pid=
cleanup() {
    if [[ -n $server_pid ]]; then
        kill "$server_pid" 2>/dev/null || true
        wait "$server_pid" 2>/dev/null || true
    fi
    rm -rf -- "$run_dir"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# Compile before resetting the disposable database; never overwrite a user binary.
"$GO" build -o "$run_dir/mch-server" ./cmd/server
if curl --silent --output /dev/null --connect-timeout 1 --max-time 2 "$API_TEST_BASE_URL/api/v1/health"; then
    echo "A service already responds at $API_TEST_BASE_URL; choose a dedicated test port" >&2
    exit 1
fi
for fixture in ../db/init.sql ../db/seed.sql; do
    if ! psql "$API_TEST_DB_URL" -v ON_ERROR_STOP=1 -f "$fixture" >>"$run_dir/db.log" 2>&1; then
        cat "$run_dir/db.log" >&2
        exit 1
    fi
done
"$run_dir/mch-server" -db "$API_TEST_DB_URL" -port "$API_TEST_PORT" >"$run_dir/server.log" 2>&1 &
server_pid=$!
for ((attempt = 0; attempt < 50; attempt++)); do
    if ! kill -0 "$server_pid" 2>/dev/null; then
        cat "$run_dir/server.log" >&2
        exit 1
    fi
    if curl --fail --silent --output /dev/null --connect-timeout 1 --max-time 2 "$API_TEST_BASE_URL/api/v1/health"; then
        "$GO" test -count=1 ./api-tests/...
        exit 0
    fi
    sleep 0.1
done
cat "$run_dir/server.log" >&2
echo 'Timed out waiting for the test backend' >&2
exit 1
