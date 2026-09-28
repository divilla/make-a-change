# 023 — CLI change management

P3/spec022 merged as actual dev `80c67871aaa29ed92a967aa70b449b29bd88c406`
after clean native review pass04. Branch change/023-cli-change-management was
created from freshly fetched dev. The user authorized the full sequential factory.
P4 in approved CLI plan. Read AGENTS, architecture, latest compact checkpoint,
contract/reuse ledger and current backend change handlers/domain/service. Scope
is CLI changes and required adjacent read/document adaptation to preserve existing
editor tests when the obsolete embedded change payload disappears. Allowed cli/,
numbered spec and plan status only. No backend/DB/frontend/cli-proto/root-helper/
.mch/dependency edits. Log cli/implementation-log.md. Caller owns factory Git.

## Acceptance

- P4-01 Typed cancellable single-operation POST transport for all eleven current
  /api/v1/change routes: list(project_id), details(id), create(project_id,title,
  brief,optional ref_uuid), update-epic(id,nullable epic_id), update-after-change
  (id,nullable after_change_id), update-phase(id,change_phase), update-open(id,
  explicit open), update-types(id,change_types), update-title(id,title),
  update-pr-url(id,pr_url), delete(id). List/details200 exact typed body,
  create201 ID-only, writes/deletes204 empty. Use established finite timeout,
  injection, typed error/status/cause and malformed-data rejection. Each method
  performs one HTTP operation, no hidden reads or write retries. No /get,
  /assign-flow, def field, recursive envelope search or alternate field guesses
  for migrated change paths. Adapter-only tests remain unit-only.
- P4-02 Accurate returned fields: id/project_id integers; ref_uuid string;
  nullable ref int32, slug, epic_id and epic_name; phase/types/title/open;
  done_tc/total_tc/completed int64; updated_at timestamp; details also nullable
  after_change_id, pr_url, created_at. Display all fields including nullable
  state and server identity/completion values without deriving old identity or
  recalculating completion. Changes do not contain document bodies, agent_edit,
  or embedded testcases. Presentation state is separate from wire DTOs.
- P4-03 Reachable keyboard list/details/create/delete and all seven updates,
  help/actions, loading/empty/error/success states, preserved filtering/navigation.
  Explicit title, editable brief and optional UUID are all accessible on create;
  allow backend-generated UUID by omission. Validate title/brief and UUID without
  requiring arbitrary Markdown structure; reuse helpful parsing/editor mechanics
  where consistent. Backend requires configured backlog phase and brief document
  type for create; explain absent catalogs and failures, no invented defaults.
  Phase/type choices use selected-project catalog order; explicit empty type array
  clears types. Epic/prerequisite can clear to null; open can explicitly be false.
  PR URL accepts backend HTTP(S) contract and cannot be cleared with blank input;
  do not invent a clear operation unsupported by backend. Test each field/action.
- P4-04 Changes owns feature state, validation, forms, sequencing and results;
  shell only composes focus/navigation, injects capabilities and routes messages.
  Keep feature-to-feature and adapter-to-feature boundaries enforced. Long lists
  and multiline details must fit scrolling viewports; retain linear-time ANSI-aware
  truncation and access to every field. Loaded command-shaped values remain literal
  data on unchanged saves; retained mutation outcomes are scoped only to the
  associated failed read retry and cleared after recovery/unrelated navigation.
  Empty find returns restart reads canceled on entering search. Capture
  project/entity/op/revision for async results and ignore/cancel obsolete reads.
  No stale detail response overwrites new selection, draft or status. Do not select
  invisible cached list rows during reload. Program shutdown cancels owned work.
- P4-05 Record every committed write independently of subsequent reads. Preserve
  created ID, saved field/value and deletion outcome when refresh or a later write
  fails; show exact completed step and remaining failure. Read-only retry never
  repeats create/update/delete. Prevent duplicate submits while busy. Preserve
  dirty raw editor bytes across save failures, unchanged-editor no-op, literal
  slash data and retry, exact original bytes/tabs/long documents, cleanup/redraw,
  cancellation, selected-project config-save serialization/draining and all P2/P3
  partial-success/stale-result regressions. Test multi-call partial failures and
  cancellation without falsely claiming a rolled-back write.
- P4-06 Preserve existing document/editor behavior while eliminating fake change
  document fields: use current/insert document operations and configured change
  document types through owned document state/capabilities. Rename def to brief
  in active CLI DTOs/commands/screens/parsers; no compatibility alias or hidden old
  route. Ordinary configured document access remains separate from agent workflow.
  This prerequisite supports existing change editor paths only; generic owner
  navigation, history/details and full document management remain P6. Use current
  backend doc DTOs: current(ref_id,ref_table) returns array of docs for all types;
  insertion carries ref_id/ref_table/doc_type/body/agent_edit and returns201 ID.
  Do not resend a successful insertion after subsequent type/detail read failure.
  Separate testcase reads as needed for detail display rather than expect embedded
  testcases; remaining testcase mutation migration/feature ownership is P5.
  Record this dependency-driven supporting scope explicitly in plan/ledger.
- P4-07 Reuse complete-program fake server/drivers and PTY harness. Exercise all
  eleven operations via keyboard with exact payload/effect and visible result
  assertions, including nullable clears, false open and empty type set, malformed
  response, partial success, project switch/late response and exit. Migrate fixtures
  to actual backend wire types; preserve each valuable old assertion or document
  its obsolete-contract replacement. Direct client/model calls are not integration.
  Map each acceptance bullet and every field to named meaningful unit tests;
  add complete-program names to manifest and inventory assertion, plus real PTY
  navigation/editor/rendering evidence. No live backend or real agent needed.

## Verification and handoff

Run make -C cli check, coverage, deps-audit, integration-test and
integration-coverage on final production and after review fixes. Strict >95% unit
and >90% terminal apply to final rebuild. Separate all-production profiles include
cmd and testless packages; no dishonest exclusions, merged counters or skipped
required scenarios. Intermediate numerical gaps alone do not block dev merge;
repair introduced failures and touched baseline issues, report remaining owners.
Update ledger/compact checkpoint/implementation log/plan with actual commands,
exits, counts, package gaps, hashes and next P5. No predicted merge hash; no full
rebuild-complete claim. Caller runs code helper, native review loop and dev merge.
No child commits/pushes or nested factory. No stage/production promotion.


## Baseline and required commands

P3 final unit2723/3201 (85.0672%), terminal2028/3201 (63.3552%). All required
unit/race/program/PTY/architecture/tooling/deps scenarios pass; four untouched
format files and package-comment lint remain in the checkpoint. Repair touched
baseline files and all introduced failures; numerical shortfalls alone may continue
through intermediate review/dev merge. Preserve P1–P3 review regressions.

```sh
make -C cli check
make -C cli coverage
make -C cli deps-audit
make -C cli integration-test
make -C cli integration-coverage
```

Generic change-code skeleton/PRD and root implementation-log defaults are
explicitly overridden for this factory by AGENTS.md. Read the skill and follow
its implementation/acceptance checks using the authorized CLI references and
cli/implementation-log.md block format. Implement only this specification;
caller handles commits, pushes, native review and dev merge. Do not launch a
nested factory. Ask about genuine unresolved product/contract decisions instead
of inventing behavior. No live backend is designated; use current-contract fakes.


## Resume after quota interruption

The first helper invocation implemented this spec and completed verification,
then hit its Codex CLI usage limit before final documentation/log/publication.
The user restored quota. An explicitly incomplete snapshot preserves that work
for the helper's clean-tree retry. Inspect the current checkpoint and source;
continue the existing implementation rather than restart it. Finish acceptance
inspection, missing implementation-log block and accurate status, verifying any
new edits as required. Use original spec commit79d1305 as the full diff baseline
for implementation statistics. P4 has not yet had native review or dev merge.
