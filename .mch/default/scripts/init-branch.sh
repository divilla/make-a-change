#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  printf '%s\n' 'usage: init-branch.sh <change-slug>' >&2
  exit 2
fi

change_slug="${1}"
change_slug_pattern='^[0-9]+-[0-9A-Za-z_-]+$'
if [[ ! "${change_slug}" =~ $change_slug_pattern ]]; then
  printf 'invalid change slug: %s\n' "${change_slug}" >&2
  exit 1
fi
change_ref="${change_slug%%-*}"

repo="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "${repo}" ]]; then
  printf '%s\n' 'cannot extract repo root' >&2
  exit 1
fi

if [[ -n "$(git -C "${repo}" status --porcelain)" ]]; then
  printf '%s\n' 'working tree contains uncommitted changes' >&2
  exit 1
fi

branch="change/${change_slug}"
if git -C "${repo}" show-ref --verify --quiet "refs/heads/${branch}"; then
  git -C "${repo}" checkout "${branch}"
  exit 0
fi

git -C "${repo}" fetch --prune origin
if git -C "${repo}" show-ref --verify --quiet "refs/remotes/origin/${branch}"; then
  git -C "${repo}" checkout --track -b "${branch}" "origin/${branch}"
  exit 0
fi

mapfile -t local_ref_branches < <(
  git -C "${repo}" for-each-ref \
    --format='%(refname:short)' \
    "refs/heads/change/${change_ref}-*"
)
if [[ ${#local_ref_branches[@]} -gt 0 ]]; then
  printf 'branch for change ref %s already exists locally: %s\n' \
    "${change_ref}" "${local_ref_branches[0]}" >&2
  exit 1
fi

mapfile -t remote_ref_branches < <(
  git -C "${repo}" for-each-ref \
    --format='%(refname:short)' \
    "refs/remotes/origin/change/${change_ref}-*"
)
if [[ ${#remote_ref_branches[@]} -gt 0 ]]; then
  printf 'branch for change ref %s already exists remotely: %s\n' \
    "${change_ref}" "${remote_ref_branches[0]}" >&2
  exit 1
fi

git -C "${repo}" checkout stage
git -C "${repo}" merge --ff-only origin/stage
git -C "${repo}" checkout -b "${branch}"
