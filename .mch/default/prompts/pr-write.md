---
name: change-pr
description: Draft a GitHub pull request body for a repository Change workflow without creating the PR. Use when the user asks to write, generate, rewrite, or prepare a PR body for a Change branch.
---

# Change PR

Draft a reviewer-focused GitHub PR body for the current Change. Do not create, update, push, merge, or comment on a pull request.

## Hard Limits

- Read repository instructions first when `AGENTS.md` is present. Follow repository instructions when they are stricter than this skill.
- Do not edit code, docs, tests, database files, fixtures, snapshots, generated artifacts, or application data.
- Only create or update the PR artifact at `/stg-tmp-dir/output.md`.
- Do not stage, commit, reset, restore, format, migrate, seed, or mutate any database.
- Do not post PR comments.

## Identify The Change

- If the user names a Change spec or Change name, use it.
- Otherwise derive `<change-slug>` from the current branch:

  ```bash
  git branch --show-current
  ```

- The branch must match:

  ```text
  ^change/([0-9]+-[0-9A-Za-z_-]+)$
  ```

- If the branch does not match, stop with exactly:

  ```text
  The branch must be named change/<change-slug>.
  ```

- Require `specs/<change-slug>.md`. If it does not exist, stop with exactly:

  ```text
  Missing specs/<change-slug>.md.
  ```

## Build Context

Read the Change spec as the strict instructions for the intended Change. Then inspect the actual
branch contents:

- `git status --short`
- full tracked diff against the PR base branch
- staged and unstaged diffs
- untracked files that would be relevant to the PR

Use the configured workflow base branch when available. In this Project Manager repository, default
to `origin/stage` when available.

Documentation is outside the default Change Flow. Do not inspect it by default or add documentation
work to the PR summary unless it is present in the diff. Read a retained ADR or operational runbook
only when the Spec explicitly cites it or the diff changes it.

## Contract Check

Compare the Change spec with the branch diff before writing the PR draft.

Stop instead of drafting if:

- the diff includes material behavior outside the Change scope
- the PR title cannot be determined from the Change spec
- verification claims in the Change spec conflict with available evidence
- database or generated-file changes appear and repository instructions prohibit acting on them

Report the exact conflict with file paths and the specific Change requirement or diff item involved.

When a Spec instruction is absent from the diff, continue drafting and identify it explicitly as
deferred. Never describe absent behavior as implemented.

## Draft Requirements

Write the complete PR body to `/stg-tmp-dir/output.md`, intentionally overwriting that file.

The first line must be the title, formatted exactly:

```markdown
# <Title>
```

Follow it with exactly one blank line. The title must match the Change title exactly. If the Change spec has no explicit title, derive the title from the first `# ...` heading in the Change spec. If no title can be found, stop and report that the Change title is missing.

Use this structure unless the repository gives a stricter PR template:

```markdown
# <Title>

## Summary
- ...

## Behavior
- ...

## Verification
- ...

## References
- `specs/<change-slug>.md`
- ...
```

Omit `## Behavior` only when the branch has no externally observable behavior or contract change. Include additional sections only when they help reviewers, such as `## Data Model`, `## API`, `## Frontend`, `## CLI`, `## Docs`, or `## Risks`.
Add `## Deferred Spec Instructions` whenever the code does not implement one or more Spec
instructions.

## Writing Standards

- Be specific and concise. Write for reviewers who need to understand what changed, why it changed,
  how it was verified, and what was intentionally deferred.
- Reflect the branch diff, not intentions. Mention backend, frontend, CLI, docs, database, tests, seed/demo data, or generated artifacts only if they are present in the diff.
- Prioritize externally observable behavior, public contracts, API payloads, data model changes, migrations, seed/demo changes, and verification evidence.
- Do not include implementation diary, generic praise, filler, broad claims, or unexplained internal details.
- Do not claim verification passed unless the exact command was run in this branch and the result is known.
- If verification was not run, say `Not run` and give the reason if known.
- If verification failed, include the exact command and the relevant failure summary. Do not soften or reinterpret failures as passing.
- Include a `References` section listing `specs/<change-slug>.md`. Add an ADR or runbook only when
  it was explicitly part of the Change.

## Final Response

After successfully writing `/stg-tmp-dir/output.md`, output exactly:

Done.
