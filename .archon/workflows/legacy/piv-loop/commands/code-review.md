# PIV Loop — Automated Code Review

The implementation phase is complete. Review ALL changes against the plan.

**Implementation output**: $implement.output

---

## Step 1: Read the Plan

Read `$ARTIFACTS_DIR/plan.md` to understand the intended implementation.

## Step 2: Review All Changes

```bash
git log --oneline --no-merges $(git merge-base HEAD $BASE_BRANCH)..HEAD
git diff $BASE_BRANCH..HEAD --stat
git diff $BASE_BRANCH..HEAD
```

## Step 3: Check Against Plan

For EACH task: was it implemented correctly? Do success criteria hold?
For EACH file: check quality, security, patterns, CLAUDE.md compliance.

## Step 4: Run Validation

```bash
bun run validate 2>&1 || (bun run type-check && bun run lint && bun run test && bun run format:check)
```

## Step 5: Fix Obvious Issues

Fix type errors, lint warnings, missing imports, formatting. Stage only the files you fixed — never `git add -A`. Skip the commit if there were no fixes:
```bash
git add path/to/file1 path/to/file2 ...  # list real fixes only
git status --porcelain  # verify nothing scratch/review/PR-body is staged
git diff --cached --quiet || git commit -m "fix: address code review findings"
```

**Never stage**: `.pr-body.md`, `pr-body.md`, `*.scratch.md`, `*.tmp.md`, `review/`, `*-report.md` at the repo root, anything under `$ARTIFACTS_DIR`, or repo-local `.archon/artifacts/`, `.archon/logs/`, `.archon/state/` (local-only Archon telemetry — never in git).

## Step 6: Present Review

```
## Code Review Complete

### Implementation Status
| Task | Status | Notes |
|------|--------|-------|
| {task} | DONE / PARTIAL / MISSING | {notes} |

### Validation Results
- Type-check: PASS / FAIL
- Lint: PASS / FAIL
- Tests: PASS / FAIL
- Format: PASS / FAIL

### Code Quality Findings
{Issues found, or "No issues found."}

### Recommendation
{READY FOR REVIEW / NEEDS FIXES}
```
