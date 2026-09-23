Do not use any skills for this request.

# Spec Review

Review the provisional Spec before implementation. Verify that it converts the Definition and relevant
branch work into a complete, internally consistent, implementation-ready guide.

Perform exactly one bounded review pass, return the required result, and stop.

## Preconditions

Read `AGENTS.md` first when present.

Derive `<change-slug>` from the current branch using:

```text
^change/([0-9]+-[0-9A-Za-z_-]+)$
```

If the branch does not match, stop with exactly:

```text
The branch must be named change/<change-slug>.
```

Require:

- `agent/defs/<change-slug>.md`
- `specs/<change-slug>.md`

Stop with one concise, path-specific error when either file is missing.

This is a read-only review. Do not edit, format, stage, commit, reset, restore, migrate, seed,
post PR comments, or mutate tracked files, database files, generated artifacts, or application
data.

Documentation is outside the default Change Flow. Do not inspect documentation unless the Definition or
Spec explicitly includes documentation work or cites a retained ADR or operational runbook.

## Sources and Authority

Build fresh context from the current repository on every invocation. Do not rely on conversation
memory, prior review output, or findings from an earlier pass.

The Definition and initial code originate the Change. The Spec expresses the desired future state and
strictly instructs what must change and how the Change must be conducted. Inspect:

- the complete current Definition
- the complete current provisional Spec
- existing code as the source of truth for current behavior
- committed, staged, unstaged, and relevant untracked branch work
- relevant tests and repository-supported verification commands
- only explicitly scoped documentation or cited ADR/runbook constraints

If a PR exists, inspect it as a summary and evidence source. Code remains the source of current
behavior, and the Spec remains the intended Change instructions.

Do not require the branch to have already implemented provisional Requirements. Report an
implementation mismatch only when existing work contradicts the Spec and is not clearly
intentional future work.

## Review Criteria

Verify that the Spec:

- uses Goal, Scope, Requirements, Non-Goals, Design Notes, Verification, QA Test Cases,
  Review Focus, and Follow-Ups in the required order
- accounts for every meaningful Definition item
- accounts for relevant implementation already applied before the Spec was written
- describes intended final behavior without progress markers, completion statuses, or a change
  diary
- contains testable, non-contradictory Requirements and clear scope boundaries
- distinguishes intentionally excluded or unrelated work through Non-Goals or Follow-Ups
- lists realistic verification commands without unsupported success claims
- provides QA scenarios for applicable happy paths, failures, no-op behavior, persistence, and
  boundaries
- contains enough information to implement and review the Change without chat history

## Convergence and Stopping Rules

The review must converge when invoked repeatedly against corrected files:

- Evaluate only the current Definition, Spec, code, branch state, and available PR evidence.
- Never repeat a finding that the current files have resolved.
- Never reintroduce a resolved finding with different wording or split it into narrower variants.
- Consolidate findings with the same root cause into one actionable finding.
- Report every currently discoverable blocking root cause in this pass. Do not save findings for a
  later pass.
- Do not manufacture findings to continue the review loop.
- Do not require optional elaboration, preferred wording, future-proofing, or implementation detail
  that can follow an established project convention without changing observable behavior.
- Treat unchanged inputs deterministically: the same repository state must produce the same result.
- Do not invoke another review, resume another session, or ask the user to rerun the review.
- Stop reviewing immediately after producing the required output.

An unresolved material ambiguity is a finding, not a conversational question. Once no blocking
finding remains, return the exact clean result below. That result is the terminal success signal for
any external review loop.

## Finding Rules

Use severities:

- P0: data loss, security breach, production outage, or unrecoverable corruption
- P1: contradiction that makes implementation unsafe, broken core workflow, serious contract
  conflict, or destructive ambiguity
- P2: missing material Definition or branch behavior, unverifiable Requirement, missing required
  coverage, or user-visible ambiguity
- P3: minor correctness ambiguity that can still cause implementation or review confusion

Report only concrete findings that block safe implementation or reliable review. Exclude style
preferences, wording nits, optional improvements, and speculative concerns.

Reference only the current Definition or Spec in finding locations. Use the absolute file path and the
current line or line range that must change.

Number findings sequentially starting at `1`.

## Output Format

Return findings only in this exact format:

```markdown
Review Findings:

1. [P<severity>] <short imperative title> — <absolute file path>:<line-or-range>
   <one concise paragraph explaining the impact and specific fix direction.>
```

Do not include praise, summaries, questions, style preferences, or broad refactor suggestions.

If there are no blocking findings, return exactly:

```markdown
Review Findings:

No blocking issues found.
```
