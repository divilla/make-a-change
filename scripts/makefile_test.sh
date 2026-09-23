#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
test_root=$(mktemp -d)
trap 'rm -rf -- "$test_root"' EXIT
repo="$test_root/repo with spaces"
fake_bin="$test_root/bin"
log="$test_root/tool.log"
mkdir -p "$repo/backend/cmd/server" "$repo/frontend" "$repo/db" "$fake_bin"
cp "$script_dir/../Makefile" "$repo/Makefile"
touch "$repo/backend/cmd/server/main.go" "$repo/db/init.sql" "$repo/db/seed.sql"
# All current targets must still run when files share their names.
touch "$repo/init" "$repo/run" "$repo/db/seed-demo.sql"
export MAKEFILE_TEST_REPO="$repo" MAKEFILE_TEST_LOG="$log"
export PATH="$fake_bin:$PATH"

cat >"$fake_bin/go" <<'TOOL'
#!/usr/bin/env bash
set -euo pipefail
printf 'go:%s\n' "$*" >>"$MAKEFILE_TEST_LOG"
if [[ $1 == build ]]; then
    [[ $PWD == "$MAKEFILE_TEST_REPO/backend" ]]
    [[ $* == 'build -o aipm-server cmd/server/main.go' ]]
    [[ ${MAKEFILE_TEST_FAIL_BUILD:-0} == 0 ]] || exit 17
    cat >aipm-server <<'SERVER'
#!/usr/bin/env bash
set -euo pipefail
[[ $PWD == "$MAKEFILE_TEST_REPO/backend" ]]
printf '%s\n' 'backend:started' >>"$MAKEFILE_TEST_LOG"
SERVER
    chmod +x aipm-server
else
    [[ $1 == install && $# -eq 2 ]]
fi
TOOL
cat >"$fake_bin/lsof" <<'TOOL'
#!/usr/bin/env bash
set -euo pipefail
printf 'lsof:%s\n' "$*" >>"$MAKEFILE_TEST_LOG"
# Return no PIDs so the Makefile cannot terminate any actual processes.
TOOL
cat >"$fake_bin/docker-compose" <<'TOOL'
#!/usr/bin/env bash
set -euo pipefail
[[ $PWD == "$MAKEFILE_TEST_REPO" ]]
printf 'docker-compose:%s\n' "$*" >>"$MAKEFILE_TEST_LOG"
TOOL
cat >"$fake_bin/pnpm" <<'TOOL'
#!/usr/bin/env bash
set -euo pipefail
[[ $PWD == "$MAKEFILE_TEST_REPO/frontend" ]]
printf 'pnpm:%s\n' "$*" >>"$MAKEFILE_TEST_LOG"
TOOL
cat >"$fake_bin/psql" <<'TOOL'
#!/usr/bin/env bash
set -euo pipefail
[[ $# -eq 5 && $2 == -v && $3 == ON_ERROR_STOP=1 && $4 == -f ]]
[[ -f $5 ]]
printf 'psql:%s\n' "$*" >>"$MAKEFILE_TEST_LOG"
if [[ ${MAKEFILE_TEST_FAIL_SCHEMA:-0} == 1 && $5 == db/init.sql ]]; then
    echo 'simulated schema failure' >&2
    exit 19
fi
TOOL
chmod +x "$fake_bin/"*

make -s -C "$repo" init
[[ $(<"$log") == $'go:install golang.org/x/lint/golint@latest\ngo:install honnef.co/go/tools/cmd/staticcheck@latest' ]]

: >"$log"
make -s -C "$repo" run >"$test_root/run.out"
[[ $(wc -l <"$log") -eq 6 ]]
grep -Fxq 'lsof:-ti tcp:8080' "$log"
grep -Fxq 'lsof:-ti tcp:8000' "$log"
grep -Fxq 'docker-compose:up -d' "$log"
grep -Fxq 'go:build -o aipm-server cmd/server/main.go' "$log"
grep -Fxq 'backend:started' "$log"
grep -Fxq 'pnpm:dev' "$log"
grep -Fq 'Backend:  http://localhost:8080' "$test_root/run.out"
grep -Fq 'Frontend: http://localhost:8000' "$test_root/run.out"

: >"$log"
if MAKEFILE_TEST_FAIL_BUILD=1 make -s -C "$repo" run >"$test_root/build.out" 2>&1; then
    echo 'expected build failure' >&2
    exit 1
fi
! grep -Eq '^(backend:|pnpm:)' "$log"

: >"$log"
make -s -C "$repo" db
expected='postgres://postgres:postgres@localhost:5432/changes -v ON_ERROR_STOP=1 -f'
[[ $(<"$log") == "psql:$expected db/init.sql"$'\n'"psql:$expected db/seed.sql"$'\n'"psql:$expected db/seed-demo.sql" ]]

: >"$log"
make -s -C "$repo" db DATABASE_URL=postgres://test/custom
[[ $(wc -l <"$log") -eq 3 ]]
[[ $(grep -c '^psql:postgres://test/custom ' "$log") -eq 3 ]]

: >"$log"
if MAKEFILE_TEST_FAIL_SCHEMA=1 make -s -C "$repo" db >"$test_root/db.out" 2>&1; then
    echo 'expected schema failure' >&2
    exit 1
fi
[[ $(wc -l <"$log") -eq 1 ]]
grep -Fq 'simulated schema failure' "$test_root/db.out"

printf '%s\n' 'Makefile tests passed'
