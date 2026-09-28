# 021 — CLI projects and typed transport

P2 on change/021-cli-projects-typed-transport, from fresh origin/dev 6a3d5bb.
P1/spec020 is merged after native review pass08 was clean. Scope is P2 in
agent/cli-rebuild-plan.md; read the compact checkpoint, AGENTS.md, target CLI
architecture, and current backend project handlers/domain/config types. The user
authorized this entire factory sequence. Implement this specification only;
caller owns review, commits/pushes, merge and continuation.

Allowed writes cli/, numbered spec, plan status. Do not change backend, DB,
frontend, cli-proto, .mch retained resources, root helpers, dependencies or other
docs. The factory owns commits/pushes. Log cli/implementation-log.md. No missing
skeleton/PRD assumptions. No live backend is designated; use aligned fakes.

## Purpose

Deliver complete ordinary project management and establish typed, cancellable
HTTP transport for subsequent feature passes. Preserve retained ordinary editor,
navigation and config behavior from P1. Do not implement the agent workflow or
unrelated epic/change/testcase/document/config-management/health screens here.
Existing unported feature methods may remain until their owning pass; do not
add new compatibility aliases, envelope searches or alternate routes for projects.

## Acceptance criteria

- P2-01 One HTTP operation per typed transport method. Project POST operations are
  /api/v1/project/list, details, create, update, delete, config. Current inputs:
  create(name), update(id,name), details/delete/config(id), list without invented
  filtering/pagination. Current Project fields: id integer, name, config slug,
  last_ref, created_at, updated_at, change_count. Current create status201 with
  created ID only; update/delete204 empty. Read project configuration via its
  own typed response for all six arrays. Re-read actual current DTOs before
  implementation; no backend imports or client-invented request fields.
- P2-02 Transport accepts caller context, finite HTTP timeout, injected HTTP client,
  and handles request/encode/network/status/decode/close paths with preserved
  status and underlying error causes. Decode exact typed wire shapes, reject
  malformed/wrong-shape/missing-required data as contract errors, and handle204
  without decoding. No recursive map or alternate field-name guessing for
  migrated operations. Keep wire types private where possible and shared
  cross-feature values in dto; format display strings at presentation boundary.
  Unit adapter tests assert exact routes/methods/payloads, status, cancellation,
  errors, malformed bodies and no hidden extra requests. Adapter-only counters
  never count as terminal integration.
- P2-03 Reachable discoverable project list/details/create/edit/delete/config actions
  expose all supported request fields and returned detail fields with loading,
  empty/error/success states. Real saves/deletes, no placeholder success. Own
  project forms, validation, sequencing/results in internal/projects; root app
  owns shell/focus/routing/selection and constructs injected capabilities.
  Keep feature imports within actual enforced architecture.
- P2-04 Creating records the ID and explicitly refreshes details/list when needed.
  Update/delete successes are preserved if subsequent reads fail; show saved
  or deleted plus refresh-failed and allow read retry, never automatic write
  retry. Validation blocks invalid local inputs without losing drafts. Unit
  tests cover success, failure, partial success and exactly-once mutation calls;
  program scenarios drive keyboard actions and assert visible outcomes/effects.
- P2-05 Project selection loads that project's configuration, refreshes on switching,
  scopes caches to the model/project, and surfaces missing configuration without
  global/default fallback. Replace old global option endpoints used for these
  catalogs with selected project config; downstream option consumers use the
  resolved catalogs without requiring other feature migrations. Preserve P1's
  atomic asynchronous local selection persistence and honest save-failure state.
- P2-06 Requests/results carry project/entity and operation identity; late list/detail/
  config/save results cannot overwrite newer selection or current operation.
  Propagate program cancellation to reads, cancel or ignore obsolete work, and
  sequence dependent writes/reads outside synchronous Update/View. Unit tests
  use controlled delayed responses; full-program tests prove switching while
  requests are pending and shutdown with active HTTP work.
- P2-07 Extend reused fake backend/program harness and scenario manifest to current
  project wire contracts. Full-program integration covers all six project routes,
  all editable fields, selection/config behavior and partial-success errors;
  real PTY test verifies discoverable navigation/editor behavior. Keep ordinary
  API operations independent of agents, Flow files and workflow workspaces.
  Preserve valuable previous tests, migrate obsolete project contract assertions
  with ledger entries, and add named unit tests for each acceptance bullet.

## Reference details and migration boundaries

Backend project contract sources are backend/internal/project/api.go and
backend/internal/domain/{project,dtos}.go. Config response has slug plus
project_docs, epic_docs, change_docs, change_phases, change_colors, change_types
as arrays of strings. Respect catalog order and backend values. Do not invent
catalog-object IDs, derive missing defaults, or silently manufacture colors when
configuration is incomplete. No project config assignment write exists in the
current project request DTOs; expose the returned configuration and its slug,
not a client-invented field or endpoint. Configuration management is P7.

Current project DTO is presentation-heavy with string ID/timestamps; migrate wire
and shared business values to accurate typed values and perform display/selector
conversion at the feature/presentation boundary. It is legitimate to adapt other
consumers and existing test fakes as necessary to compile, but do not migrate
unrelated business operations here. Keep a clear ledger of remaining old client
methods and the pass that will replace them. No removed /get or /options route
may remain in a migrated project/catalog path.

Retain P1's exact document bytes, dirty-draft retry, recognized form command vs
literal data, editor cleanup, queued atomic config-save permissions and orderly
quit behavior. Preserve created IDs/partial success, even while unrelated change
wire contracts remain deferred to P4. Review regression tests are part of the
retained behavior suite, not disposable implementation-specific tests.

Every criterion needs meaningful named unit tests and explicit terminal scenario
mapping where applicable. Reuse existing tests/harnesses; update their assertions
only for the recorded current-backend migration. Extend the manifest if adding
new complete-program or PTY test names; a source test existing outside the
selected manifest alone does not establish terminal coverage.

## Baseline

P1 final unit2220/2727 (81.4081%), terminal1427/2727 (52.3286%). All unit race,
program, PTY, architecture, tooling and deps-audit pass. Check still fails on14
untouched format files and sampled package-comment lint; see checkpoint for paths.
Repair touched project/client/app files and related DTO formatting/comments in
this pass; retain an explicit owner for unrelated baseline findings. Do not stop
implementation or intermediate merge solely for a numerical coverage shortfall.

## Verification

Run make -C cli check, coverage, deps-audit, integration-test and
integration-coverage on final code and after review fixes. Keep strict all-package
unit >95% and terminal >90% gates with separate profiles; these are final-result
targets, so honest intermediate numerical gaps alone do not block dev merge.
Repair introduced regressions and touched baseline lint/format issues; keep any
remaining unrelated baseline failures visible with owners. Update contract/reuse
ledger, compact checkpoint, plan status and implementation log with actual
revision, command exits, coverage counts/package gaps, scenarios and next P3.

Run on final implementation and review fixes:

```sh
make -C cli check
make -C cli coverage
make -C cli deps-audit
make -C cli integration-test
make -C cli integration-coverage
```

No APIHydra or live backend is required for these CLI terminal scenarios. Keep
all production packages in both denominators, including cmd and packages with no
tests. Preserve raw failures, measured counts and package gaps; do not call a
failed/incomplete campaign passing. Keep coverage targets strict and independent.
Use cli/implementation-log.md per AGENTS.md. Compact the checkpoint to final
contracts/results and next P3; retain historical evidence in Git/logs rather than
accumulating every review transcript. Update plan status with actual merged P1
SHA6a3d5bb and P2 implementation evidence, never a predicted future squash hash.

## Notes

Do not commit/push or invoke another factory from the implementation child. The
caller uses the implementation helper, native review loop and dev merge helper.
No stage/production promotion. Ordinary project API use must require neither
Flow resources nor an agent executable. Framework/dependency migrations and
backend fixes are outside this specification; report a genuine contract blocker
rather than guessing or altering a backend operation.
