#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)

"$script_dir/codex-code-spec.pl" agent/specs/032-cli-brief-spec-flow.md
"$script_dir/codex-review-loop.pl" agent/specs/032-cli-brief-spec-flow.md
