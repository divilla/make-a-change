#!/usr/bin/env bash
set -euo pipefail

fail() {
    printf 'error: %s\n' "$1" >&2
    exit 1
}

if (($# != 2)); then
    printf '%s\n' 'usage: commit.sh <change-slug> <message>' >&2
    exit 2
fi

change_slug="$1"
message="$2"
change_slug_pattern='^[0-9]+-[0-9A-Za-z_-]+$'

if [[ ! "$change_slug" =~ $change_slug_pattern ]]; then
    fail "invalid change slug: $change_slug"
fi

if [[ -z "$message" ]]; then
    fail "commit message is required"
fi

if ! repo_root=$(git rev-parse --show-toplevel 2>/dev/null); then
    fail "cannot extract repo root"
fi

if [[ -z "$repo_root" ]]; then
    fail "cannot extract repo root"
fi

if ! branch_name=$(git -C "$repo_root" branch --show-current 2>/dev/null); then
    fail "cannot read current branch"
fi

branch_pattern='^change/([0-9]+-[0-9A-Za-z_-]+)$'
if [[ ! "$branch_name" =~ $branch_pattern ]]; then
    fail "current branch does not match $branch_pattern: $branch_name"
fi

branch_slug="${BASH_REMATCH[1]}"
if [[ "$branch_slug" != "$change_slug" ]]; then
    fail "branch slug $branch_slug does not match change slug $change_slug"
fi

git -C "$repo_root" add -A
git -C "$repo_root" commit -m "$message"
git -C "$repo_root" push
