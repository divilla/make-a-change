# PIV Loop — Finalize

The implementation has been approved. Push changes and create a PR.

---

## Step 1: Push Changes

```bash
git push -u origin HEAD 2>&1 || echo "WARNING: Push failed — verify remote authentication and branch state before creating the PR."
```

## Step 2: Generate Summary

```bash
git log --oneline --no-merges $(git merge-base HEAD $BASE_BRANCH)..HEAD
git diff --stat $(git merge-base HEAD $BASE_BRANCH)..HEAD
```

Read `$ARTIFACTS_DIR/plan.md` and `$ARTIFACTS_DIR/progress.txt` for context.

## Step 3: Create PR (if not already created)

Resolve the origin repo first — in a fork clone, gh otherwise targets the upstream parent:

```bash
ORIGIN_REPO=$(git remote get-url origin | sed -E 's#^.*[:/]([^/]+/[^/]+)$#\1#; s#\.git$##')
gh pr view HEAD --repo "$ORIGIN_REPO" --json url 2>/dev/null || echo "NO_PR"
```

If no PR exists:

```bash
cat .github/pull_request_template.md 2>/dev/null || echo "NO_TEMPLATE"
```

Create with `gh pr create --repo "$ORIGIN_REPO" --draft --base $BASE_BRANCH`
(re-run the `ORIGIN_REPO=...` line in the same shell — it does not persist across shells):
- Title from the plan's feature name
- Body summarizing the implementation
- Use a HEREDOC for the body

## Step 4: Output Summary

```
===============================================================
PIV LOOP — COMPLETE
===============================================================

Feature: {from plan}
Plan: {plan file path}
Branch: {branch name}
PR: {url}

-- Tasks Completed -----------------------------------------------
{list from progress tracking}

-- Commits -------------------------------------------------------
{git log output}

-- Files Changed -------------------------------------------------
{git diff --stat output}

-- Validation ----------------------------------------------------
All checks passed.
===============================================================
```
