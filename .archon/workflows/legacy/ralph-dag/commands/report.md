# Completion Report

The Ralph implementation loop has finished. Generate a completion report.

## Context

**Loop output (last iteration):**

$implement.output

**Setup context:**

$validate-prd.output

---

## Instructions

### 1. Read Final State

Extract the `PRD_DIR=...` from the setup context above.
Read the CURRENT files from disk:

```bash
cat {prd-dir}/prd.json
cat {prd-dir}/progress.txt
```

### 2. Gather Git Info

```bash
git log --oneline --no-merges $(git merge-base HEAD $BASE_BRANCH)..HEAD
git diff --stat $(git merge-base HEAD $BASE_BRANCH)..HEAD
```

### 3. Check PR Status

```bash
gh pr view HEAD --json url,number,state 2>/dev/null || echo "No PR found"
```

### 4. Generate Report

Output this format:

```
═══════════════════════════════════════════════════════
RALPH DAG — COMPLETION REPORT
═══════════════════════════════════════════════════════

Feature: {feature name from prd.json}
PRD: {prd-dir}
Branch: {branch name}
PR: {url or "not created"}

── Stories ─────────────────────────────────────────

| ID | Title | Status |
|----|-------|--------|
{for each story from prd.json}

Total: {N}/{M} stories passing

── Commits ─────────────────────────────────────────

{git log output}

── Files Changed ─────────────────────────────────

{git diff --stat output}

── Patterns Discovered ─────────────────────────────

{from ## Codebase Patterns in progress.txt, or "None"}

═══════════════════════════════════════════════════════
```

Keep it factual. No commentary — just the data.
