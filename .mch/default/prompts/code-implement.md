---
name: change-code
description: Implement code according to active change file
---

Extract `<change-slug>` from the current Git branch using this regex: `^change/([0-9]+-[0-9A-Za-z_-]+)$`

If the current branch does not match, stop and output one concise error explaining that the branch must be named `change/<change-slug>`.

Stop without editing if `specs/<change-slug>.md` does not exist.

Implement Change spec `specs/<change-slug>.md` with senior-level discipline.

The Change spec defines the requested implementation scope. Existing code is the source of truth
for current behavior and technical contracts. Implement the smallest coherent code, test,
configuration, seed, and explicitly authorized database-file changes needed to satisfy the Change.

Documentation is outside the default Change Flow. Do not inspect, create, update, or reconcile
documentation unless the user explicitly requests documentation work or the active Spec explicitly
includes it. When documentation is explicitly in scope, use code as the source of current behavior
and limit edits to the requested README, ADR, or operational runbook work.

Before coding:
1. Read the full Change spec.
2. Inspect the existing code and tests that define current behavior.
3. Inspect the relevant backend, frontend, CLI, configuration, database, seed, and test patterns
   before choosing an approach.
4. Read a retained ADR or operational runbook only when the Spec explicitly cites it as a required
   decision or operational constraint.
5. Inspect the current worktree and preserve unrelated local changes.
6. Verify the Change spec is implementation-ready: it must follow the standard Change structure, define a clear Goal, Scope, Requirements, Non-Goals, Verification, and QA Test Cases, and contain enough detail to implement and verify the behavior without relying on chat history.

Stop conditions:
- If the Change spec conflicts with existing code in a way that is not clearly an intentional requested change, stop before coding and report the exact conflict.
- If any Change spec required section is missing, ambiguous, or untestable, stop before coding and
  report the exact readiness gap.
- If the required external behavior, API contract, persistence contract, field naming, endpoint naming, history behavior, seed behavior, or verification expectation is unclear, stop and ask one specific clarifying question.
- If unrelated local changes block a safe implementation, stop and describe the conflict.
- If database behavior blocks implementation, report the blocker instead of mutating live/local database state outside approved verification commands.

Hard rules:
- Do not broaden scope beyond the Change spec.
- Do not refactor unrelated code.
- Do not revert or overwrite unrelated local changes.
- Follow existing project architecture, naming, transaction, DTO, API, frontend, CLI, and test patterns.
- Keep implementation scoped to the files required by this Change.
- Do not create foreign keys.
- Do not introduce broad locking, advisory locks, isolation escalation, or cross-path locking unless explicitly required by the Change.
- Do not mutate any live/local database manually. Only use repository Make targets that operate on
  disposable test databases when needed.
- Do not weaken, skip, delete, rebaseline, or bypass tests to make verification pass.

Verification:
- First run focused tests for touched behavior.
- Run every verification command required by the Change spec and by the repository rules for each touched area; if the Change spec omits a required repository verification command, run the repository-required command anyway and report the omission.
- If a command fails, report:
    - exact command
    - failing test or error
    - whether local uncommitted changes may have influenced it
    - whether it appears to be a product bug, test bug, environment issue, or unclear contract dependency

Final report:
- Summarize the implemented behavior.
- List the main files changed.
- List verification commands run and their pass/fail status.
- Call out any unresolved follow-ups or risks. 
