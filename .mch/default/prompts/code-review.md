---
name: change-review
description: Review a Change branch or PR against origin/stage using the active Change spec as scope and code as the source of truth like a very senior engineer. Use when asked to review a Change, review the current changes branch, review a Change PR, find blocking issues, or produce review findings without implementing fixes.
---

Review the current branch against `origin/stage` like a very senior engineer.

Build fresh context from the repository. Do not rely on conversation memory.

Before review:
- Read `AGENTS.md` first when present. Repository instructions override this skill if they are stricter.
- Identify the current Git branch:
  ```bash
  git branch --show-current
  ```
- The branch must match, extracted text is `<change-slug>`:
  ```text
  ^change/([0-9]+-[0-9A-Za-z_-]+)$
  ```
- If it does not match, stop with this exact error:
  ```text
  The branch must be named change/<change-slug>.
  ```
- Derive `<change-slug>` from the branch and require:
  ```text
  specs/<change-slug>.md
  ```
- If the file does not exist, stop with this exact error:
  ```text
  Missing specs/<change-slug>.md.
  ```

This is a review skill. Do not edit, format, stage, commit, reset, restore, migrate, seed, or otherwise mutate tracked files, database files, fixtures, snapshots, generated artifacts, or application data while reviewing.

Documentation is not a default review input. Inspect documentation only when the Change explicitly
includes documentation work, the diff changes it, or the Spec explicitly cites a retained ADR or
operational runbook as a required constraint.

Respect `AGENTS.md`: treat `specs/<change-slug>.md` as the intended Change instructions and code as
the source of current behavior. Do not mutate database state, edit files, or post a PR comment.

Review process:
- Read the active Change spec under `specs/`.
- Read a retained ADR or operational runbook only when the Spec explicitly cites it.
- Inspect the full diff against `origin/stage`.
- Identify changed public contracts, API behavior, data model changes, migrations, tests,
  explicitly scoped documentation, and workflows.
- Run or inspect the listed verification commands when feasible.
- Verify every Requirement against the diff and tests, and report omitted instructions unless they
  are explicitly identified as deferred.

Return findings only, in this exact format:

Review Findings:

- [P<severity>] <short imperative title> — <absolute file path>:<line or line-range>
  <one concise paragraph explaining the concrete impact, when it occurs, and the specific fix direction.>

Severity rules:
- P0: data loss, security breach, production outage, or unrecoverable corruption.
- P1: broken core workflow, incorrect persisted state, API contract break, serious regression.
- P2: real bug, missing required coverage, race/stale state issue, edge case with user-visible impact.
- P3: minor correctness issue or low-risk maintainability problem that can still cause confusion.

Output rules:
- Start with exactly `Review Findings:`
- Use one Markdown bullet per finding.
- Add one blank line after each finding
- Use absolute file paths and exact line or line range.
- Put the explanation on the next line, indented by two spaces.
- Do not include praise, summaries, style nits, preferences, or broad refactor suggestions.
- Do not suggest changes outside the active Change scope unless they block correctness.
- If no blocking issues exist, return exactly:

Review Findings:

No blocking issues found.
