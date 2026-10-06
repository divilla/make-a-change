#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)

spec=${1:?Usage: codex-flow.sh SPECIFICATION}
"$script_dir/codex-code-spec.pl" "$spec"
"$script_dir/codex-review-loop.pl" "$spec"
