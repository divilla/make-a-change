---
name: change-fix
description: Implement and commit local fixes for review findings on a Change branch using the active Change spec as scope and code as the source of truth. Use when the user asks to fix, address, implement findings, implement review findings, or implement review comments for a Change without pushing or rerunning the full PR workflow. Do not use when the user explicitly declines implementation or only asks to ignore, explain, discuss, review, document, or summarize findings.
---

Implement only review-fix work for the active Change.

Documentation is outside the default Change Flow. Do not inspect, create, update, or reconcile
documentation unless the user explicitly requests documentation work or the active Spec explicitly
includes it.

## Preconditions

If the user explicitly says not to implement findings, or asks only to ignore, explain, discuss, review, document, or summarize findings, stop and answer without making changes.

Extract `<change-slug>` from the current Git branch using `^change/([0-9]+-[0-9A-Za-z_-]+)$`.

If the branch does not match, stop with: `Branch must be change/<change-slug>.`

If `specs/<change-slug>.md` does not exist, stop with one concise error.

## Gather Findings

Use exactly one findings source:

- When the user asks to fix PR comments or review findings from the PR, read PR comments for the current branch PR with `gh pr view change/<change-slug> --json url,comments`. If `gh` fails, report it as a blocker. Treat the latest comment as the comment with the newest `createdAt` timestamp. Stop if there is no PR, no comments, or the latest comment body is exactly `No blocking issues found.`
- For pasted review findings in the conversation, use those findings.
- For a local review artifact explicitly named by the user, read that artifact.

If no findings are available, stop and ask for the review findings.

When PR comments include IDs, treat a comment as already resolved if its ID is already recorded in the Change spec `Follow-Ups` section as fixed or invalid. Stop if all actionable PR comments are already resolved.

Only fix actionable review-finding comments: comments with finding-style content, explicit requested fixes, or comments the user identifies. Ignore status comments, discussion, and non-actionable notes.

## Before Editing

1. Read `AGENTS.md` and obey it.
2. Read `specs/<change-slug>.md`; treat it as the requested Change scope.
3. Inspect the relevant code and tests as the source of truth for current behavior. Read a retained
   ADR or operational runbook only when the Spec or finding explicitly cites it.
4. Inspect and remember starting `git status`, unstaged diff, and staged diff.
5. Stop if pre-existing staged changes are not explicitly part of the supplied findings.
6. Stop if existing unstaged changes overlap with files needed for the fix and cannot be safely preserved.
7. Map each finding to expected behavior, affected files, and needed tests.

## Fix Rules

- Fix every supplied actionable finding, or mark it invalid with evidence, or block with a concrete reason.
- Do not broaden scope beyond the Change spec and review findings.
- If a finding is unclear, ask one specific question for that finding; continue with independent clear findings only when safe.
- Preserve unrelated local changes.
- Do not refactor unrelated code.
- Add or update focused tests for changed behavior unless the fix is documentation-only or the finding requires no executable change.
- Obey the full `AGENTS.md` database boundary.
- For PR-comment fixes, record each resolved PR comment ID in the Change spec `Follow-Ups` section with outcome fixed or invalid. Do not record blocked comments as resolved.
- Do not push, merge, or post PR comments unless explicitly asked.

## Verify And Commit

1. Inspect changes introduced by this fix; remove accidental unrelated edits or leave pre-existing unrelated edits unstaged.
2. Run focused tests for changed behavior.
3. Run applicable commands listed in `specs/<change-slug>.md` under `Verification`.
4. Run required verification commands for touched areas.
5. If any required verification fails, is skipped, or is infeasible, do not stage or commit; report the blocker.
6. If any later edit, format, generated update, or staging correction changes review-fix content, rerun affected focused tests before committing.
7. Stage only changes introduced for the supplied review findings.
8. If a touched file also contains unrelated local edits, stage only review-fix hunks; if that cannot be verified safely without stashing, resetting, or discarding user changes, stop before committing.
9. Inspect staged diff and confirm it is limited to review-fix changes.
10. Inspect remaining unstaged diff and confirm it contains no review-fix changes that should be committed.
11. Commit only when all findings are resolved, no finding is blocked, required verification passed, staged review-fix changes exist, and staged diff is limited to review-fix changes.
12. Use commit subject: `Fix review findings for <change-slug>`.

## Final Report

Report:

- Findings fixed, invalid, or blocked.
- Verification commands and results.
- Final worktree state, including unrelated local changes.
- Commit hash, if created.
- Final status exactly one of:
  - `committed`: commit was created.
  - `fixed-uncommitted`: findings are resolved but no commit was created because there were no file changes, verification failed/skipped/was infeasible, or staging/commit safety blocked the commit; include the reason.
  - `blocked`: any finding remains blocked.
