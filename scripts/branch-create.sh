#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "usage: scripts/branch-create.sh <spec-path>" >&2
}

fail() {
	echo "branch-create: $*" >&2
	exit 1
}

ensure_clean_worktree() {
	local changes
	changes=$(git status --porcelain=v1 --untracked-files=all --ignore-submodules=none)
	if [[ -n "$changes" ]]; then
		echo 'error: Uncommitted work detected. Commit or stash your changes, then retry.' >&2
		exit 1
	fi
}

if [[ $# -ne 1 ]]; then
	usage
	exit 2
fi

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(git -C "$script_dir" rev-parse --show-toplevel 2>/dev/null) || \
	fail "scripts directory is not inside a git repository"
cd "$repo_root"

spec_path=$1
[[ -f "$spec_path" ]] || fail "specification file not found: $spec_path"
if [[ "$spec_path" =~ ^agent/specs/([^/]+)\.md$ ]]; then
	spec_slug=${BASH_REMATCH[1]}
else
	fail "specification path must match agent/specs/<spec-slug>.md"
fi

branch="change/$spec_slug"
git check-ref-format --branch "$branch" >/dev/null 2>&1 || \
	fail "invalid change branch derived from specification: $branch"

ensure_clean_worktree
bash "$script_dir/git-auth.sh" --check
git fetch --prune origin 1>&2
ensure_clean_worktree

perl -I "$script_dir/lib" -Mmch::Release=run_cli,ensure_release_branches \
	-e 'run_cli(sub { ensure_release_branches() })' 1>&2

if git show-ref --verify --quiet "refs/heads/$branch"; then
	fail "branch already exists: $branch"
else
	status=$?
	[[ $status -eq 1 ]] || fail "cannot inspect branch $branch"
fi

if git show-ref --verify --quiet "refs/remotes/origin/$branch"; then
	fail "branch already exists: origin/$branch"
else
	status=$?
	[[ $status -eq 1 ]] || fail "cannot inspect origin/$branch"
fi

git checkout --no-track -b "$branch" origin/dev 1>&2
ensure_clean_worktree
printf '%s\n' "$branch"
