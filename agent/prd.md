# Make a Change Product Requirements Document

## Scope and baseline

First release; draft pending the open decisions below. Confirmed requirements
in this document also serve as acceptance criteria; they are stated once.
Unresolved requirements must not be guessed.

Make a Change helps individuals and small teams organize development work,
clarify briefs, write specs and PR drafts, and track implementation, human
review, QA, and production. Participants are humans, artifact-writing agents,
external coding agents, and QA; these roles do not define account permissions.
The product should be simple and easy to use; measurable criteria remain open.

- `mch` supports interactive terminal use and scriptable commands for agents
  and automation. Web development begins after the CLI release is complete.
- Development runs locally with a database, backend, and clients, initially
  the CLI. The design must accommodate future hosting, accounts, and
  authentication, all deferred from this release.
- Codex is the only artifact-writing runner. Other providers and configurable
  provider selection are deferred.
- Another app handles coding launch, execution, self-review, fixes, and
  resumption. Make a Change supplies specs and records progress; it does not
  launch, supervise, resume, retry, or stop coding runs.
- Code-change detection, PR rewrite scheduling, and opening PRs on repository
  hosts are outside the confirmed scope.

Retain existing functionality, including project, epic, change, and test-case
operations and metadata, from commit
`1a7f718ee7321c0ef08867205d7a3877144d4034`, except where this PRD explicitly
changes, removes, or defers it. The baseline is defined by the [CLI](../cli/),
[backend](../backend/), [database](../db/init.sql),
[CLI integration tests](../cli/integration/), and
[backend API tests](../backend/api-tests/). This PRD takes precedence.

## Changes and identifiers

- Hierarchy: Project → optional Epic → Change. Changes represent issues,
  tasks, or features and belong directly to a project or an epic within it.
- Each project corresponds to one repository; `mch` runs inside it. Broader
  project-to-repository relationships are deferred.
- New changes immediately start in `backlog`, with database column `open`
  defaulting to `true`.
- Creation generates the slug once from the submitted title. Later title
  edits preserve it; users can explicitly edit it. Slugs may repeat within
  a project because `ref` distinguishes branches and directories.
- Slug generation follows [the database normalization](../db/init.sql):
  lowercase and trim the title, replace runs outside `a-z` and `0-9` with a
  hyphen, trim surrounding hyphens, and use `change` if empty.
- The database assigns `ref`, unique within the project, when a change first
  leaves `backlog`. It is permanent, including when returning to `backlog`.
- All change views return
  `to_char(c.ref, 'FM000000') || '-' || c.slug as ref_slug`.
  Branches and directories use this returned value: `ref` of `6` and slug
  `some-change` produce `000006-some-change`.

## Phase and open state

Stored and displayed phase names preserve this exact casing and spelling:

| Phase | Meaning |
| --- | --- |
| `backlog` | Initial phase. |
| `todo` | Chosen by the user for execution. |
| `in-progress` | External coding, self-review, and fixing findings. |
| `in-review` | Ready for or undergoing human review. |
| `in-testing` | Thorough QA testing after human review. |
| `in-prod` | In production. |

Users and agents can set `phase` through the CLI at any time, in any direction.
The only condition is that leaving `backlog` requires a valid written spec
attached; without it, the change cannot leave `backlog`. There are no other
phase rules. Phase changes record tracking state without controlling coding.

The user can independently set `open` at any time, in any phase: `false`
closes the change; changing `false` to `true` reopens it.

## Brief and spec flow

Creating a brief or applying any user update to it automatically runs steps
1–6 below, in order, using Codex, even when a spec already exists. Any user
update to a spec runs steps 5–6. Agent API saves continue the current flow
without restarting it as a user edit.

1. Validate the brief: first line `# Title`, then one empty line, then the
   brief body. This format is required before proceeding.
2. Check that the brief can become a spec without guesswork. For ambiguities
   or missing information, ask the user clarifying questions, obtain valid
   answers that resolve them, and update the brief through the backend API.
3. Rewrite the brief for wording and clarity; update it through the backend API.
4. Write the spec from the clarified brief; save it through the backend API.
5. Audit all possible spec ambiguities. Ask the user clarifying questions
   wherever needed and obtain valid answers; resolve every ambiguity without
   guesswork and update the spec through the backend API.
6. Rewrite the spec for wording and clarity; update it through the backend API.

Wording improvements do not resolve missing requirements or ambiguities.
There is no separate human approval between clarification and spec writing.
The exported spec is the handoff to the external coding app.

## Documents and repository files

Maintained product documentation defines current application behavior.
Individual change specs describe incremental changes that can supersede
earlier functionality; historical specs collectively are not the current
product contract.

Users edit documents through `mch`; the agent saves brief and spec flow updates
through the backend API. Repository files are exported copies. Direct edits
to those copies are not imported or synchronized back into the database.

Specs can be created and edited before `ref` assignment. Creating or editing
PR, fixes, and findings documents requires an assigned `ref`; access remains
available when a numbered change returns to `backlog`.

Whenever a change leaves `backlog`, create `change/<ref_slug>` if absent and
check it out. Write the attached spec to repository-root
`agent/<ref_slug>/spec.md`, creating the directory if needed, and commit the
spec to that branch. Every subsequent spec edit immediately rewrites its
repository copy at any time, in every phase, including `backlog`.

All document paths are relative to the repository root:

| Document | Path | Purpose |
| --- | --- | --- |
| Spec | `agent/<ref_slug>/spec.md` | Required work. |
| Fixes | `agent/<ref_slug>/fixes.md` | User feedback during `in-review`. |
| Findings | `agent/<ref_slug>/findings.md` | QA problems during `in-testing`. |
| PR | `agent/<ref_slug>/pr.md` | Markdown draft describing actual code changes. |

For example: branch `change/000006-some-change`, spec
`agent/000006-some-change/spec.md`.

Fixes and findings are optional documents attached to the change; either,
both, or neither may exist. Saving either first checks out `change/<ref_slug>`
and exports the saved document to its path above. Filenames are fixed;
additional documents may be defined, but their names and behavior remain open.

Editing the slug after export renames the branch and entire document directory
using the updated `ref_slug`, preserving all contents, filenames, and `ref`.

## PR drafting

Only explicit invocation of the `write pr` CLI command triggers PR writing or
rewriting. Humans and agents can invoke it with an assigned `ref`. Codex writes
a Markdown draft describing actual code changes to `agent/<ref_slug>/pr.md`.
The caller decides when to request it. Brief and spec flows, feedback saves,
and phase changes do not trigger PR writing.

## Open decisions

1. **Repository operations:** behavior on checkout, export, commit, or rename
   failure, and when a rename target branch or directory already exists.
2. **Manual slugs:** permitted characters and handling of invalid values.
3. **Clarification lifecycle:** controls beyond existing document-agent
   functionality for managing the brief and spec flow.
4. **Feedback history:** fixes/findings behavior across repeated review and QA
   cycles, and required history.
5. **Release quality:** measurable targets beyond existing behavior and checks,
   including usability.

The existing APIHydra architecture document is not a resolved Make a Change
architecture contract; reconciling it is outside this PRD-only edit.
