# PIV Loop — Address Validation Feedback

The human has reviewed the implementation and provided feedback.

**Human's feedback**: $LOOP_PREV.fix-feedback-review.output.text

---

## Step 1: Read Context

Read `$ARTIFACTS_DIR/plan.md` and CLAUDE.md for conventions.

## Step 2: Process Feedback

**If there is no user feedback yet** (first iteration, $LOOP_PREV.fix-feedback-review.output.text is empty):
- Present the code review results and ask the user to test the implementation
- Present the result for the following approval gate.


**If the user provided specific feedback:**
1. Read the relevant files
2. Understand each issue
3. Make the fixes
4. Type-check after each change

## Step 3: Full Validation

```bash
bun run validate 2>&1 || (bun run type-check && bun run lint && bun run test && bun run format:check)
```

## Step 4: Commit Fixes

Stage **only** the files you actually edited while addressing feedback — never `git add -A`. List them by name:

```bash
git add path/to/file1 path/to/file2 ...
git status --porcelain  # verify nothing scratch/review/PR-body is staged
git commit -m "$(cat <<'EOF'
fix: address review feedback

Changes:
- {fix 1}
- {fix 2}
EOF
)"
```

**Never stage**: `.pr-body.md`, `pr-body.md`, `*.scratch.md`, `*.tmp.md`, `review/`, `*-report.md` at the repo root, anything under `$ARTIFACTS_DIR`, or repo-local `.archon/artifacts/`, `.archon/logs/`, `.archon/state/` (local-only Archon telemetry — never in git).

## Step 5: Report

```
## Feedback Addressed

Changes made:
- {fix 1}
- {fix 2}

Validation: {PASS / FAIL with details}

Review again, then use the approval gate to finalize or request changes.
```

Only the following approval gate decides whether to proceed. Treat its previous feedback as requests to address, never as implicit approval.
