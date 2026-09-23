#!/usr/bin/env bash
set -euo pipefail

# Publish the current Change PR and persist its URL in the repository.

fail() {
    printf 'error: %s\n' "$1" >&2
    exit 1
}

if (($# != 0)); then
    printf '%s\n' 'usage: pr-publish.sh' >&2
    exit 2
fi

if ! repo_root=$(git rev-parse --show-toplevel 2>/dev/null); then
    fail "cannot extract repo root"
fi

if [[ -z "$repo_root" ]]; then
    fail "cannot extract repo root"
fi

cd "$repo_root"

if ! branch_name=$(git branch --show-current 2>/dev/null); then
    fail "cannot read current branch"
fi

branch_pattern='^change/([0-9]+-[0-9A-Za-z_-]+)$'
if [[ ! "$branch_name" =~ $branch_pattern ]]; then
    fail "current branch is not a change/<change-slug> branch: $branch_name"
fi
change_slug="${BASH_REMATCH[1]}"

pr_author="divilla"
if ! gh_login=$(gh api user --jq .login); then
    fail "gh api user --jq .login failed"
fi
if [[ "$gh_login" != "$pr_author" ]]; then
    fail "gh is authenticated as $gh_login, expected $pr_author"
fi

if ! git merge-base --is-ancestor origin/stage HEAD; then
    fail "rebase needed: origin/stage is not an ancestor of HEAD"
fi

pr_file="agent/prs/$change_slug.md"
if [[ ! -f "$pr_file" ]]; then
    fail "cannot read PR body file $pr_file"
fi

first_line=""
if ! IFS= read -r first_line < "$pr_file" && [[ -z "$first_line" ]]; then
    fail "PR body file is empty: $pr_file"
fi
first_line="${first_line%$'\r'}"
if [[ ! "$first_line" =~ ^#[[:space:]]+(.+)$ ]]; then
    fail "first line of $pr_file must be '# <Title>'"
fi
pr_title="${BASH_REMATCH[1]}"
while [[ "$pr_title" == *[[:space:]] ]]; do
    pr_title="${pr_title%?}"
done
if [[ -z "$pr_title" ]]; then
    fail "first line of $pr_file must be '# <Title>'"
fi

git add -A
git commit -m "Write PR for $change_slug by agent"
git push origin "change/$change_slug"

mkdir -p agent/prurls
if ! pr_url=$(
    gh pr create \
        --base stage \
        --head "$pr_author:change/$change_slug" \
        --title "$pr_title" \
        --body-file "$pr_file"
); then
    fail "gh pr create failed"
fi
printf '%s\n' "$pr_url" > "agent/prurls/$change_slug"

git add -A
git commit -m "Write PR URL for $change_slug by agent"
git push origin "change/$change_slug"
