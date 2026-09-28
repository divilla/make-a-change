# 009 — Backend testcase alignment

## Authorized execution

Implement this specification now, on `change/009-backend-testcase-alignment`
created from freshly fetched origin/dev after P3 merges. Read AGENTS.md,
docs/backend-architecture.md, db/init.sql, db/seed.sql, this specification,
backend/agents/backend-contracts.md and backend/agents/backend-refactor-checkpoint.md.
Reconcile against merged P3, not a draft working tree. The user approved the
sequential backend factory through the final refactor; the explicit API/domain
migrations below are authorized. Do not stop for reconfirmation. The user scope
overrides change-code defaults: ignore PRD, skeleton, CLI, frontend and unrelated
architecture references.

All implementation/tests/fixtures/artifacts/logs stay under backend/. This
committed specification and top status of agent/backend-refactor-plan.md are
authorized exceptions. No root scripts, root AGENTS.md, db/, docs/, dependency
changes or Go-managed transactions. Append the prescribed change-code log to
backend/implementation-log.md, not root, excluding the log from counts. The
factory owns commits/pushes; do not commit/push in the implementation subprocess.
Do not begin R1 or promote to stage/production. Do not spawn concurrent checkout
writers.

Use change-code and apih-build skills with backend/apih-tests and the installed
apih manual at c3947513e2a4b948ce732dec60e090a5e83be744 (local
/home/vito/go/src/apihydra). Retain retries:-1 for mutations/error scenarios to
avoid replay. >95% unit and >=90% real APIHydra production statement coverage
are FINAL refactor goals, not intermediate blockers. Measure both honestly,
retain failed gates, and finish bounded work despite numerical shortfalls.
Repair introduced failures; report known unrelated baseline formatting/lint
debt. No artificial tests, exclusions, mixed unit/API profiles or weakened
assertions. Do not implement the later owned-database outage campaign here.

## Current authority and explicit migration

public.testcase has six columns only: id bigint, change_id bigint FK to change,
scenario text, done bool default false, created/modified timestamptz default
now(). There is no version, testcase history, soft delete, insert/update/delete
procedure or recalculation function. Change and epic views calculate counts
from current testcase rows; there are no parent counter columns or parent
modified triggers to emulate.

Retain all five POST /api/v1/test-case routes: list, create, update, update-done,
delete. Keep /update-change absent. Create returns201 with only a domain ID DTO
(use domain.TestCaseIDRequest consistently with P2's entity ID DTO pattern).
Updates and delete return204 with truly empty bodies and error-only service/repo
methods. Independent list provides current testcase state; change/epic reads
provide counters. Do not add a details/history/move endpoint merely for symmetry.

Keep int IDs across requests/results to match P2/P3; test representable large
IDs rather than gratuitously converting every API ID. Counters remain the wide
P2/P3 types. Preserve existing bool done behavior: omitted or null done means
false, explicit false is valid, wrong JSON types fail400. Do not introduce a
required *bool migration in this pass.

All cross-layer business input/output types live in internal/domain. Pass the
same domain request through API -> service -> repository where possible. API
binds/validates using existing Echo/validate and central errors; service validates
direct callers independently and normalizes scenario; repo does SQL/binding/
scanning and central DB errors. Service must no longer import change or receive
its Renderer. Small pgx Query/QueryRow/Exec interfaces are justified by meaningful
boundary unit tests; no generic SQL/transaction framework.

## Acceptance criteria

P4-01 Current independent reads: domain.TestCase contains exactly id, change_id,
scenario, done, created, modified, with no version. List accepts
TestCaseListRequest end-to-end; positive change_id required. Check live parent,
then SELECT explicit six columns FROM public.testcase WHERE change_id=$1 ORDER
BY id. Existing parent with no cases returns non-nil [] and200; missing parent
returns centralized testcase404 as before. Rows close on all iteration paths;
query/scan/iteration failures preserve causes. Document the existing two-read
existence/list race: concurrent deletion may produce empty[] after a positive
existence check; do not introduce a transaction to claim an atomic snapshot.

P4-02 Create: trim scenario and reject blank/nonpositive parent in service;
INSERT INTO public.testcase(change_id,scenario) VALUES($1,$2) RETURNING id,
letting DB supply false and timestamps. Scan only ID, return201 exact {"id":N}.
No post-insert read/list/render, transaction or parent preflight is needed for
this single FK-backed operation. SQLSTATE23503 maps through central Database to
ErrTestCaseNotFound while preserving PgError cause and errors.Is/As. Unknown
errors retain their cause and generic500 HTTP envelope. Keep existing message
`test case not found` for missing parent instead of introducing module sentinels.

P4-03 Explicit update/delete: update accepts TestCaseUpdateRequest, trims and
rejects blank scenario/nonpositive ID; one UPDATE public.testcase SET
scenario=$2,modified=now() WHERE id=$1. update-done accepts
TestCaseUpdateDoneRequest, positive ID and the retained bool policy; one UPDATE
sets done=$2,modified=now() WHERE id=$1. delete accepts TestCaseIDRequest and
positive ID; one DELETE FROM public.testcase WHERE id=$1. RowsAffected0=>404;
otherwise error-only success and204. No reload, child/parent mutation, implicit
history or rendering. Same-value scenario/done updates succeed204 and execute
the timestamp update without a preliminary read. Preserve created/unrelated
fields; no parent modified or manual count update. Direct statements provide
statement-atomic affected-row and FK semantics, not a multi-call transaction.

P4-04 Error/API/direct-service contracts: retain module error definitions and
functions in internal/error (ErrTestCaseInvalidInput, ErrTestCaseNotFound,
InvalidPayload, HTTP, Database). All five handlers preserve P1 operation-specific
bind messages and400/404/generic500 envelope. Direct service calls enforce the
same positive IDs/nonblank normalized scenarios. All request/business values
cross layers as domain types; no unpacked list ID at the service/repo boundary.
Unit tests prove malformed/wrong-type payloads, zero/negative/large valid IDs,
both done values and omitted/null false, repository errors and exact create/
mutation bodies. /update-change remains404. No side reads hidden in API composition.

P4-05 Remove unsupported coupling: inspect actual merged P3 references, then
remove testcase transaction wrappers, finishMutation, old procedure/getTestCase
reload helpers, changeColumns/getChange/scanChange, service.renderMutation and
change renderer injection at startup. Remove TestCaseMutationResponse and
TestCase.Version. Remove P3's explicitly temporary legacy domain.Change,
Renderer.RenderChange and RenderMutation once testcase is the last consumer;
retain P3's explicit current-document/artifact rendering and parser/sanitizer
collaborators. No production code may query public.test_case, testcase history,
removed testcase procedures or perform Begin/Commit/Rollback after alignment.
Remove obsolete-only tests with reasons recorded; retain each supported error,
state, counter and sanitization guarantee through its appropriate current API.
Do not preserve dead legacy DTOs/adapters merely to keep old tests compiling.

P4-06 Real testcase workflow and view counts: extend APIHydra with one dependent
flow creating its own project, epic and change through P2/P3's ID-only201,
associating epic through204, then independent list[] and counts0/0. Create two
cases, capture IDs, list validates default false/trimmed scenarios/ID order.
Toggle first true => separate change/epic reads1/2 and completed50; second true
=>2/2/100. Edit scenario, independently list verifies text and retained done;
repeat identical scenario/done updates204; second false=>1/2/50. Delete first
=>0/1/0 and remaining item; delete last=>0/0/0 and empty[]. Add invalid/blank400,
missing parent/case404 and repeated-delete404 for all applicable operations.
Assertions use explicit statuses and stable bodies; exact keys/empty204 remain
unit-proven because installed apih subset/empty-body comparison cannot prove them.
No mutation side data is expected or counted as a read.

P4-07 Parent integrity and append-only docs: a change with live testcase children
must return P3's centralized409 on deletion and remain readable with unchanged
children/current documents. Remove children through their own operations, then
change delete204; change get/documents/set-document and testcase list/create
for the removed parent return404. Keep P3's append-only document rows after
parent deletion, including old noncurrent rows and latest current row. Reuse
its meaningful isolated SQL postcondition support to verify retained history,
parent absence and child absence; SQL checks do not contribute Go code coverage.
If direct SQL evidence currently lives in a useful legacy test, retain/adapt it
until an equivalent isolated assertion actually runs and passes. Never drop
useful SQL assertions merely because APIHydra does not directly execute SQL.
No automatic child/document cascade or fabricated testcase version history.

P4-08 Meaningful coverage, legacy tests and handoff: every criterion and its
individual behavior needs named unit evidence and genuine integration coverage
where feasible. Repository tests prove exact parameters/scan widths/order,
rows closure, unknown/constraint/query/scan/iterator/affected-row errors and
no forbidden extra calls; service fakes prove direct validation and single
repository delegation. Adapt P1 error tests rather than weaken cause assertions.
Port retained legacy GoHTTP scenarios to APIHydra/current contracts before
removing obsolete expectations; do not leave knowingly broken migrated tests
or remove independent SQL checks without working replacement. Preserve complete
route denominator (expected34 registered operations after P3; recalculate),
count all five testcase successful operations separately from negative-only reach.
Run required checks/coverage on final code, refresh compact checkpoint/contract
ledger/API report/top plan status, then hand off to R1 deadcode audit. No R1 work
or promotion in this subprocess.

## Tests to migrate and evidence required

- testcase/repo_test TestCreateTestCase/UpdateTestCaseDone/Scenario and error
  tables: replace removed fn_test_case/sp_test_case and postmutation read
  assertions with real current-table statements. Preserve valid parameter,
  both bool values, FK-to404 and errors.Is/As guarantees. Remove transaction-
  embedding mocks once unused; fake pool should fail unexpected extra calls.
- TestRepositoryTranslationKeepsExternalCauses: retain on current INSERT/scan/
  Exec paths. TestMutationChangeMissingCause and TestScanChangeCurrentSchema
  cease to apply when testcase no longer reads a change; replace with current
  six-column testcase scan/errors, leave P3 current change scan tests intact.
- TestServiceRendersMutationChangeSpecHTML disappears because related rendered
  mutation data is architecturally forbidden; retained document sanitization
  is proven in P3's explicit reads, not deleted wholesale.
- TestRepositoryHistoryProcedures is based on nonexistent test_case_history,
  version/procedures and parent cascade. Remove those unsupported expectations;
  preserve its valid FK/lifecycle/counter intent in the real APIHydra flow and
  isolated SQL postconditions. Do not add new integration writes to an arbitrary
  environment-selected external DB.
- Legacy TestTestCaseCRUDRecalculatesChangeAndEpicCompleteness is ported with
  independent reads and no versions/composite responses; views compute counts,
  so no recalculation guarantee is claimed. Keep delete-last zero counters,
  toggle correctness, invalid/missing IDs and removed-move-route assertions.
  Version-increment/preserved-version assertions are removed for absent storage.
- Audit old GoHTTP project/change helpers that decode testcase mutation or
  assume change-delete cascade. Reconcile with P3's migrated harness without
  erasing useful state/SQL coverage. Record each removed obsolete assertion's
  database/architecture reason and where retained guarantees now run.

## Fixture and runner guidance

The owned runner loads unchanged init/seed then backend/apih-tests/fixtures.sql.
Keep P2/P3 fixture IDs/scenarios intact; default project1002 supports P3 creation,
custom1001 lacks backlog/brief and must not be used for a success-create fixture.
Prefer API-created IDs captured within the same serial dependent file. Use
unique capture names because captures are write-once per invocation, body-only
interpolation, static paths, retries:-1. Isolate epic/change counters so fixtures
from other suites do not contaminate assertions. For retained-doc SQL evidence,
reuse P3's stable reserved deletion fixture/postcondition pattern instead of
trying to find an already-deleted dynamically captured parent afterward.

Do not add a production fixture/fault/shutdown endpoint, broaden into outage
or startup/config refactoring, or modify APIHydra itself. Any necessary runner
postcondition extension must remain small, preserve isolated DB ownership/
command failure cleanup and have tooling regressions; it must not merge SQL,
unit or legacy GoHTTP counters into the API statement profile.

Format touched Go files explicitly and inspect diff. Run make -C backend check,
coverage, deps-audit and api-test. If existing formatting still short-circuits
check, run make -k -C backend check to expose all components. Run tooling-test
when tooling changes. Logs /tmp/mch-p4*. Record fresh exact covered/total counts,
package gaps, exits and failed/skipped/blocked scenarios in the compact checkpoint.
Existing shared HTTP client formatting/comment/body-close debt may remain for
R5 if untouched; repair new/touched-file failures. Native review --base origin/dev
and supervisor merge-to-dev follow factory implementation; do not claim whole
backend readiness or final numerical targets passed when they have not.

## Confirmed P3 handoff additions

P3 merged at origin/dev b626dd528362c355ebc33d5c8134069a88c23377.
Its testcase adapter is now solely RenderMutation/RenderChange in
internal/change/change_view.go plus legacy domain.Change. Replace the positive
transitional testcase-consumer assertions in TestChangeArchitectureAndP4Boundary
(internal/change/schema_test.go) with absence assertions when removing those
adapters; replace TestP4RenderingAdapterRetainsLegacyContract's obsolete purpose
without weakening retained explicit-read rendering tests. P3's real current
Render method remains.

Run `make -C backend legacy-api-test` for the entire retained Go HTTP and SQL
campaign after migrating testcase tests. This uses the owned disposable runner
and preserves meaningful timestamp, generated UUID and append-only document
history evidence from P3. Repair migrated retained tests rather than skip them;
report any actual residual failure. Legacy counters stay separate and never
enter the APIHydra coverage result. Reuse/adapt
TestChangeDeletionRetainsAppendOnlyDocuments for child deletion through the new
API instead of direct SQL where practical, preserving actual SQL history
assertions. Inspect all seven previously passing project/epic/change tests and
keep their useful guarantees.

Existing fixed P3 projects2001–2002, epic2101, changes2201–2207 and testcase
IDs230001–280000 (50,000 count fixture rows) must survive unrelated P4 tests.
P3 currently executes212 requests in apih-tests/p3/steps.yaml; complete baseline
campaign312. Current final counts before P4 are unit933/1021 and API766/1021;
these are baselines to improve, not thresholds to falsify or intermediate gates.
