#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
if [[ $# -gt 1 ]]; then
    echo 'usage: scripts/merge-to-stage.sh [all|<commit-sha>]' >&2
    exit 1
fi

run_step() {
    local line=$1 error
    shift
    # Capture stderr while leaving successful command output visible.
    if error=$("$@" 2>&1 1>&3); then
        [[ -z "$error" ]] || printf '%s\n' "$error" >&2
    else
        printf 'error: lines %s-%s of file %s: %s\n' \
            "$line" "$line" "${BASH_SOURCE[0]}" "$error" >&2
        exit 1
    fi
}

run_step "$LINENO" "$script_dir/merge-to-dev.pl" 3>&1
run_step "$LINENO" "$script_dir/promote-to-stage.pl" "$@" 3>&1
