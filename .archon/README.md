# Repository workflow overrides

These files override Archon's supplied workflows **in this repository**. Run
`archon` from the repository root. Workflow filenames and `name:` values retain
their upstream names; project definitions take precedence over global and bundled
definitions. No additional configuration or installation step is required.

## Where to edit

- `workflows/sdlc/ship/archon-ship.yaml`: the shipping entry point.
- `workflows/sdlc/`: the complete SDLC pack, including `triage`, `investigate`,
  `plan`, `deliver`, `implement`, `pr`, `review`, and `validate`, plus `upkeep`,
  which shares the delivery pipeline.
- Each workflow's `commands/`, `scripts/`, and `fixtures/`: local prompts,
  executable helpers, and dry-run cases. `workflows/sdlc/.shared/` contains the
  shared helper modules; keep the pack together when copying it.
- `workflows/legacy/`: overrides for all four workflows that reported deprecated
  loop syntax. Their prompts are in adjacent `commands/` folders, including
  Ralph's `archon-ralph-generate.md` dependency.

The SDLC pack was copied from `/home/vito/dev/Archon/.archon/workflows/sdlc`
at source revision `7b3eb317d2246c306264db7915c48840d3a8070f` on 2026-09-25.
Legacy definitions and Ralph's generator command came from the same checkout.
These are independent local snapshots: upgrading Archon will not update them.
Compare upstream changes deliberately before refreshing the copies.

## Temporary specification handoff

Pass an absolute path to `archon-ship`:

```bash
archon workflow run archon-ship \
  --input spec_path=/tmp/mch/spec/123-slug.md \
  --branch change/123-slug --detach
```

The first step runs `cp` into `$ARTIFACTS_DIR/spec.md`, then `rm` on the source
only if copying succeeds. A failed copy stops the workflow and retains the source.
Triage receives the artifact path. `spec_path` takes precedence over `target`;
without it, the existing target/trigger-message behavior remains available.

## Application-code examples

`commands/` overrides the 11 original command prompts that contain
JavaScript/TypeScript application-code examples. Their original examples are
preserved, with adjacent Go alternatives for evidence, fixes, patterns, types,
tests, and documentation. The packaged Ralph generator has the same additions.

Each prompt chooses examples based on the files being built or reviewed: Go
examples for Go, JavaScript/TypeScript examples for those languages, and a choice
per file in mixed changes. Generated reports should include only the applicable
alternative. Go test examples use `testing`, table-driven cases, and subtests;
Go comments follow Go doc conventions. Template placeholders must be filled with
actual repository code. Conditional Go validation examples run from the owning
module and defer to its documented checks. Archon's runtime scripts and product
documentation are not translated.

## Loop migrations

| Workflow | Local completion mechanism |
| --- | --- |
| `archon-adversarial-dev` | `until_bash` reads valid `state.json` and checks for terminal `complete` or `failed` status. Both still reach the report, as upstream intended. |
| `archon-ralph-dag` | Required boolean `done` in structured output; true only after stories, validation, push, and PR creation succeed. |
| `archon-test-loop-dag` | `until_bash` checks the counter file for an integer of at least three. |
| `archon-piv-loop` | Implementation uses structured `done`; exploration, plan refinement, and feedback use `loop_group` with explicit approval gates. |

PIV remains interactive. Select the gate's approval decision to proceed, or its
revision decision with feedback to repeat the phase. Revision text reaches the
next iteration through `$LOOP_PREV.<gate>.output.text`. Exploration writes
`exploration.md` into the run's artifact directory so plan creation receives the
approved summary even though the group's terminal output is now a gate decision.
Existing iteration limits and the three human review boundaries are retained.

The upstream `deprecated:` notices for the legacy workflows themselves are
retained. They are distinct from the removed deprecated loop syntax.

## Validation

Requires Archon and Bun on `PATH`:

```bash
archon workflow list
bun test ./.archon/tests
archon workflow test .archon/workflows/legacy
archon workflow test .archon/workflows/sdlc
```

The fixtures never launch AI runs. SDLC fixtures marked `exec-code: true` execute
helpers in temporary Git worktrees and therefore require write access to Git's
worktree metadata. Six legacy fixtures check completion and incomplete-work
paths. Focused tests additionally check local dependencies, deterministic
completion conditions, explicit gate decisions, feedback wiring, discovery, and
PIV's first pause. Dry-run fixtures auto-approve gates and assume `until_bash`
completion; the focused tests check those boundaries separately. No live AI run
or end-to-end interactive resume has been performed.

## Forge backend

`archon-ship`, `archon-upkeep`, `archon-deliver`, `archon-pr`, and `archon-review`
each declare a `forge` input with default `gh`. Every composed call and affected
script receives the selection through an explicit binding. Scripts read
`INPUTS_FORGE`; they no longer depend on the ambient `ARCHON_SDLC_FORGE` variable.
This removes the undeclared-environment warnings and keeps standalone and nested
execution consistent.

The default uses GitHub CLI. To opt into an installed forge plugin, pass
`--input forge=forge` when launching one of these workflows. The global
`~/.archon/.env` entry added during diagnosis remains in place but no longer
controls these local overrides. Tests verify all bindings and that every affected
script uses the declared input even when the old environment variable is set.
