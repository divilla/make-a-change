#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
test_root=$(mktemp -d)
trap 'rm -rf -- "$test_root"' EXIT
repo="$test_root/repo with spaces"
mkdir -p "$repo/scripts" "$repo/agent/specs"
cp "$script_dir/codex-flow.sh" "$repo/scripts/"
printf '%s\n' '# Test specification' > "$repo/agent/specs/032-cli-brief-spec-flow.md"
export FLOW_TEST_CALLS="$test_root/calls"
for helper in codex-code-spec codex-review-loop; do
	cat > "$repo/scripts/$helper.pl" <<'HELPER'
#!/usr/bin/env bash
set -euo pipefail
[[ $# == 1 ]]
[[ $1 == agent/specs/032-cli-brief-spec-flow.md ]]
[[ -f $1 ]]
printf '%s\n' "$(basename -- "$0")" >> "$FLOW_TEST_CALLS"
case "$(basename -- "$0")" in
	codex-code-spec.pl) exit "${FLOW_TEST_CODE_EXIT:-0}" ;;
	codex-review-loop.pl) exit "${FLOW_TEST_REVIEW_EXIT:-0}" ;;
esac
HELPER
	chmod +x "$repo/scripts/$helper.pl"
done
(
	cd -- "$repo"
	bash scripts/codex-flow.sh
)
printf '%s\n' codex-code-spec.pl codex-review-loop.pl > "$test_root/expected"
cmp "$test_root/expected" "$FLOW_TEST_CALLS"

# Implementation failure must retain its status and skip review.
: > "$FLOW_TEST_CALLS"
status=0
(
	cd -- "$repo"
	FLOW_TEST_CODE_EXIT=7 FLOW_TEST_REVIEW_EXIT=0 bash scripts/codex-flow.sh
) || status=$?
[[ $status == 7 ]]
printf '%s\n' codex-code-spec.pl > "$test_root/expected"
cmp "$test_root/expected" "$FLOW_TEST_CALLS"

# Review failure after successful implementation must also propagate.
: > "$FLOW_TEST_CALLS"
status=0
(
	cd -- "$repo"
	FLOW_TEST_CODE_EXIT=0 FLOW_TEST_REVIEW_EXIT=9 bash scripts/codex-flow.sh
) || status=$?
[[ $status == 9 ]]
printf '%s\n' codex-code-spec.pl codex-review-loop.pl > "$test_root/expected"
cmp "$test_root/expected" "$FLOW_TEST_CALLS"
printf '%s\n' 'codex-flow root invocation and failure propagation passed'
