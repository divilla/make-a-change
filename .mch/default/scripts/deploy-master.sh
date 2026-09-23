#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 0 ]]; then
  printf '%s\n' 'usage: deploy-master.sh' >&2
  exit 2
fi

repo="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "${repo}" ]]; then
  printf '%s\n' 'cannot extract repo root' >&2
  exit 1
fi

branch="$(git -C "${repo}" branch --show-current)"
if [[ "${branch}" != "stage" ]]; then
  printf '%s\n' 'Please checkout stage branch.' >&2
  exit 1
fi

if [[ -n "$(git -C "${repo}" status --short)" ]]; then
  printf '%s\n' 'uncommitted changes' >&2
  exit 1
fi

remote_head_commit() {
  local branch_name="${1}"
  local output commit ref extra

  output="$(git -C "${repo}" ls-remote --heads origin "${branch_name}")"
  read -r commit ref extra <<< "${output}"
  if [[ ! "${commit}" =~ ^[0-9a-f]{40}$ || "${ref}" != "refs/heads/${branch_name}" || -n "${extra:-}" ]]; then
    printf 'cannot verify origin/%s\n' "${branch_name}" >&2
    return 1
  fi
  printf '%s\n' "${commit}"
}

local_branch_commit() {
  local branch_name="${1}"
  git -C "${repo}" show-ref --verify --quiet "refs/heads/${branch_name}" || return 1
  git -C "${repo}" rev-parse "refs/heads/${branch_name}"
}

ensure_commit_is_ancestor() {
  local ancestor="${1}"
  local descendant="${2}"
  local message="${3}"

  if ! git -C "${repo}" merge-base --is-ancestor "${ancestor}" "${descendant}"; then
    printf '%s\n' "${message}" >&2
    return 1
  fi
}

ensure_stage_is_current() {
  local expected_commit="${1}"
  local current_commit
  current_commit="$(remote_head_commit stage)"
  if [[ "${current_commit}" != "${expected_commit}" ]]; then
    printf 'origin/stage moved from %s to %s\n' "${expected_commit}" "${current_commit}" >&2
    return 1
  fi
}

git -C "${repo}" fetch origin
git -C "${repo}" checkout stage
git -C "${repo}" pull --ff-only origin stage
stage_commit="$(git -C "${repo}" rev-parse HEAD)"
origin_stage_commit="$(remote_head_commit stage)"
if [[ "${stage_commit}" != "${origin_stage_commit}" ]]; then
  printf 'stage HEAD %s does not match origin/stage %s\n' \
    "${stage_commit}" "${origin_stage_commit}" >&2
  exit 1
fi

origin_master_commit="$(remote_head_commit master)"
if master_commit="$(local_branch_commit master)"; then
  ensure_commit_is_ancestor \
    "${master_commit}" \
    "${origin_master_commit}" \
    'Local master contains commits that are not on origin/master. Refusing to promote until master is reconciled.'
fi

ensure_commit_is_ancestor \
  "${origin_master_commit}" \
  "${stage_commit}" \
  "cannot fast-forward master to stage: master is not an ancestor of ${stage_commit}"
ensure_stage_is_current "${stage_commit}"

git -C "${repo}" push \
  --atomic \
  "--force-with-lease=refs/heads/master:${origin_master_commit}" \
  "--force-with-lease=refs/heads/stage:${stage_commit}" \
  origin \
  "${stage_commit}:refs/heads/master" \
  "${stage_commit}:refs/heads/stage"

ensure_stage_is_current "${stage_commit}"
new_origin_master_commit="$(remote_head_commit master)"
if [[ "${new_origin_master_commit}" != "${stage_commit}" ]]; then
  printf 'origin/master %s does not match stage commit %s\n' \
    "${new_origin_master_commit}" "${stage_commit}" >&2
  exit 1
fi

git -C "${repo}" fetch origin refs/heads/master:refs/heads/master
new_master_commit="$(local_branch_commit master || true)"
if [[ "${new_master_commit}" != "${stage_commit}" ]]; then
  printf 'local master does not match promoted commit %s\n' "${stage_commit}" >&2
  exit 1
fi

current_branch="$(git -C "${repo}" branch --show-current)"
if [[ "${current_branch}" != "stage" ]]; then
  printf 'current branch is %s after promotion; expected stage\n' "${current_branch}" >&2
  exit 1
fi
printf 'Promoted master to %s; current branch is stage.\n' "${stage_commit}"
