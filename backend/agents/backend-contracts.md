# Backend route, schema and error ledger — validation-cause repair

Authority: read-only `../../docs/backend-architecture.md`, `../../db/init.sql`
and `../../db/seed.sql`. The current inventory is **45 registered method/path
pairs**: 13 change, eight doc, five config, 12 project/epic, five testcase and two
health operations. APIHydra exercises all 45 with successful status and response
assertions. The numbered P/R sections preserve historical refactor evidence;
the current schema alignment below supersedes their older API/SQL contracts.
No authentication middleware or session/token implementation is added.

Current HTTP tests use only `backend/apih-tests/`, with seven independent groups
running concurrently in APIHydra mode1. Each group executes numbered init/main/post
files serially. Setup and cleanup use API endpoints and captured response IDs only.
Plain `apih` uses its configured running server; `make -C backend api-test` runs an
owned instrumented server against the user-designated development database,
without SQL setup or database lifecycle management. The 90% statement gate remains
strict and independent of unit coverage. See [the suite guide](../apih-tests/coverage.md)
and [the checkpoint](backend-refactor-checkpoint.md) for fresh measurements.

## Schema alignment — 2026-10-02

Project reads use `vw_project_list` plus base-table `config_slug`, `last_ref` and
`active`. All projects remain listed, ordered active DESC, updated_at DESC, id DESC.
Epic and change active/inactive lists use the corresponding separate views.
Epic details combine both views in one UNION ALL statement and join the base
`active` field; change details use the unfiltered details view. Inactive details,
updates and child creation remain supported. Epic lists order by name,id,
matching the current view's name ordering within each requested project.

Project/epic DELETE first attempts physical deletion. Only the central mapped
SQLSTATE23503 dependency error triggers a separate `active=false,updated_at=now()`
UPDATE. Both physical deletion and successful deactivation return empty204.
Missing rows remain404; other delete errors and failed deactivation remain errors.
The two statements are independent, with no Go transaction or atomic-workflow claim.
The fallback does not modify children or their activity. Unit tests cover wrapped
FK semantics, absent rows, unchanged contexts/requests and both failure stages;
APIHydra verifies actual FK fallbacks and subsequent details/list reads.

`/change/update-active` replaces `/change/update-open`; it requires an explicit
`active` boolean. Project JSON exposes `config_slug`. `/doc/list-active` replaces
`/doc/current` and reads `vw_doc_active`, scanning `doc_id` as the domain ID.
All document reads include nullable `deleted_at`; list/details retain soft-deleted
history. `/doc/comment-list` filters only doc_type=comment, including deleted
comments. Comment creation uses its distinct function without config-type checks
or active selection; normal `/doc/insert` rejects comment even if configured.
Comment updates accept an explicit empty string, reject missing/null body, and
use the function that edits only comments; missing/non-comment IDs return404.
`agent_edit` is required for inserts and is not editable through comment-update,
matching the current function signature. Document deletion delegates to
`fn_doc_delete`, retaining history and removing the active selection atomically.

Document function calls explicitly cast parameters to their checked-in SQL types,
selecting the authoritative signatures even when the live database retains older
overloads. `fn_doc_insert` now always returns its inserted ID: the repository scans
that ID directly, and parent validation remains in the service rather than relying
on the obsolete function-null parent contract. Config deletion is a direct DELETE
with FK409 and affected-row404; the removed `fn_config_delete` is no longer used.

Tests in each affected module's `schema_alignment_test.go`, existing repository,
service/API matrices, `TestAPIConstructorRouteInventory` and the ledger audit
cover all changed contracts. New public operations have APIHydra happy/negative
cases and independent reads. The error-suite refresh adds isolated field errors,
wrong-method checks for every operation, removed-route checks, parent errors and
real PostgreSQL-rejected NUL-text errors with masked500/state-retention assertions.
List query/scan failures, outages and missing-document deletion remain meaningful
unit-only cases; no outages or guessed mutation IDs are used for HTTP coverage. No DB, CLI or frontend file is edited by this task.

## 011 validation-cause repair

Spec011 authorizes one new ordinary-error helper, `internal/error.Validation(cause,
semantic)`. It uses the existing central multiple-`%w` convention: both errors
remain inspectable, nil cause returns the semantic unchanged, nil semantic
returns the cause unchanged, and both nil return nil. No duplicate-wrapper
optimization, transport dependency in services, or registry is introduced.

All 30 tag-validation handlers (project5, epic5, testcase5, change15) now retain
actual `validate.Errors` and the module invalid-input sentinel through
`HTTP(Validation(...))`. The JSON field and failed rule remain available through
`errors.As`; existing `errors.Is` identities and generic400 envelopes remain.
Change gains sentinel identity without losing its prior validator cause.
Bind errors retain their operation-specific messages and original wrapping.

Change.UpdatePRUrl preserves the real `*url.Error` and underlying EscapeError on
parse failure. It still trims before validation, rejects invalid ID/nonblank
input before parsing/persistence, and returns the original sentinel for parsed
but business-invalid schemes/hosts. HTTP(S) remains case-insensitive; accepted
userinfo/query/fragment forms and repository call counts are unchanged. URL
rules stay in service, wrapping stays in internal/error. No routes, DTOs, SQL,
dependencies, transactions, database or deployment behavior changed.

The new handler matrices and direct service/helper tests prove cause identity;
APIHydra retains all 408 requests/34 operations, including its existing `%`
malformed-URL request and safe400 body. Full legacy SQL/HTTP tests also pass.
See the [current checkpoint](backend-refactor-checkpoint.md) for fresh evidence.
This is an error-contract repair; R2 remains no actionable duplicate cleanup.

## R1 removal audit and retained contracts

Audited clean specification HEAD `12ae35ac5d1cbbc77e9460e1fae1e0469bb199b8`
over merged P4 `b9f630f0466d93381cf8e8935563672c92c0f6b1` before editing.
Local dev and origin/dev both identify that P4 commit; HEAD adds only spec010.
Backend production/test symbol and import inspection, route registration,
`go list -json ./cmd/... ./internal/... ./pkg/...`, test dependencies and server
dependencies establish the following bounded removals:

| Candidate | Fresh evidence and disposition |
| --- | --- |
| Private `API.e` in change, testcase, project, epic and health | Each has exactly one constructor assignment and no reads; API literals occur only in their constructors. Removed only field and assignment. Echo constructor arguments, `API.g`, service injection and every registration remain unchanged, including direct `e.GET("/api/health", a.check)`. |
| Config `config.Data()["db-ws"]` print | Already commented out; removed this obsolete comment only. Global config, defaults, environment and startup flags remain unchanged. |
| `pkg/db.Pool` and `TestPoolCreationAndPanicCause` | No production or external-test import, absent from server dependencies, and only two calls in its own test. No init/registration, ignored Go files/build variants, generator/linkname/plugin/unsafe use or documented backend consumer. Backend reflection is limited to test row scans and domain.TestCase field inspection. Removed the obsolete wrapper and its sole wrapper-specific test; no coverage exclusion added. |
| Prior migration removals | Options, legacy domain.Change, RenderChange/RenderMutation and testcase transaction/history/reload paths are already absent in P4. They are not R1 deletions. |
| Retained seams and error paths | `API.g` registers routes; Renderer/parser/sanitizer, repository pool interfaces and server application lifecycle callbacks all have live callers/tests. Defensive errors, SQL operations and startup/config behavior stay. No additional removal candidates selected. |

`cmd/server.start` still calls `pgxpool.New` directly, returns wrapped errors,
closes pools on listener/CORS failures, and owns listener/server/pool shutdown.
`TestStartFailures` now checks the retained pgconn.ParseConfigError cause and
unwrap chain; `TestLifecycle`, `TestStartCancelRealServer`,
`TestHTTPServerPreservesReadTimeout`, `TestConfigurationPanicCausesAndPrecedence`
and centralized error tests remain. Removing the unused panic wrapper does not
change server failure handling. No shared contract or dependency is introduced.

`TestAPIConstructorRouteInventory` adds an exact runtime assertion of all 34
method/path pairs; the existing source/ledger tooling audit is unchanged.
Existing module handler tests retain service/context/error/JSON assertions,
including ID-only201, empty204, explicit project config and document operations.
The full unchanged APIHydra campaign passes all 408 requests. See the
[R1 checkpoint](backend-refactor-checkpoint.md) for criterion mapping, fresh
statement counts, baseline lint debt and the factory/R2 handoff.

| Method | Path | Current behavior / database source | Target and deferred pass | APIHydra |
| --- | --- | --- | --- | --- |
| GET | /api/v1/health | 200 health JSON after pool ping; 503 degraded JSON on ping error | Retain; exact body unit tested | pass |
| GET | /api/health | Same health alias and database ping | Retain | pass |
| POST | /api/v1/project/list | 200 all active/inactive projects; vw_project_list; active DESC,updated_at DESC,id DESC | P2 aligned | pass (project/02-main.yaml) |
| POST | /api/v1/project/details | 200 database-backed active/inactive details, config_slug and active | P2 aligned | pass (project/02-main.yaml) |
| POST | /api/v1/project/config | 200 selected config slug and all six ordered arrays; unavailable join 404 | P2 aligned, no fallback | pass (project/02-main.yaml) |
| POST | /api/v1/project/create | 201 {id}; one INSERT returning ID | P2 aligned | pass (project/02-main.yaml) |
| POST | /api/v1/project/update | 204 empty; one name/updated_at UPDATE, including same name | P2 aligned | pass (project/02-main.yaml) |
| POST | /api/v1/project/delete | 204 empty; DELETE, FK-only deactivation fallback; missing404; fallback errors propagated | P2 aligned | pass (project/02-main.yaml) |
| POST | /api/v1/epic/list | 200 active epics from vw_epic_list; name,id order | P2 aligned | pass (epic/02-main.yaml) |
| POST | /api/v1/epic/list-inactive | 200 inactive epics from vw_epic_inactive_list; name,id order | current | pass (epic/02-main.yaml) |
| POST | /api/v1/epic/details | 200 active/inactive details from both epic views and stored active | P2 aligned | pass (epic/02-main.yaml) |
| POST | /api/v1/epic/create | 201 {id}; one INSERT returning ID | P2 aligned | pass (epic/02-main.yaml) |
| POST | /api/v1/epic/update | 204 empty; one name/updated_at UPDATE, including same name | P2 aligned | pass (epic/02-main.yaml) |
| POST | /api/v1/epic/delete | 204 empty; DELETE, FK-only deactivation fallback; missing404; fallback errors propagated | P2 aligned | pass (epic/02-main.yaml) |
| POST | /api/v1/change/list | 200 current vw_change_list columns; service int64 completion; updated_at DESC,id; [] for absent project | P3 aligned | pass (change/02-main.yaml) |
| POST | /api/v1/change/list-inactive | 200 inactive vw_change_inactive_list entries; same completion/ordering | current | pass (change/02-main.yaml) |
| POST | /api/v1/change/details | 200 current vw_change_details; flat fields, nullable after_change_id and references, no inline docs/version/testcases; 404 missing | P3 aligned | pass (change/02-main.yaml) |
| POST | /api/v1/change/create | 201 exact {id}; selected config must support backlog/brief; UUIDv7 default or preserved caller UUID; fn_change_insert only | P3 aligned | pass (change/02-main.yaml) |
| POST | /api/v1/change/update-epic | 204; targeted parent/epic project preflight, sp_change_epic_update; nil detaches; no config | P3 aligned | pass (change/02-main.yaml) |
| POST | /api/v1/change/update-phase | 204; selected-config phase validation then sp_change_phase_update | P3 aligned | pass (change/02-main.yaml) |
| POST | /api/v1/change/update-active | 204; explicit bool; direct UPDATE active/updated_at; affected-row 404 | P3 aligned | pass (change/02-main.yaml) |
| POST | /api/v1/change/update-types | 204; ordered trim/dedup/validation using selected config; direct UPDATE types/updated_at, including clears | P3 aligned | pass (change/02-main.yaml) |
| POST | /api/v1/change/update-title | 204; existence preflight then sp_change_title_update; DB whitespace normalization; no config | P3 aligned | pass (change/02-main.yaml) |
| POST | /api/v1/change/update-slug | 204; accepts only the nonempty lowercase `[a-z0-9_-]` suffix in `slug`; direct UPDATE; list/details expose nullable `ref_slug` as the ref padded to three digits for values 0–99, otherwise unchanged, followed by `-` and the suffix | detail editor | pass (change/02-main.yaml) |
| POST | /api/v1/change/update-pr-url | 204; nonblank http(s) URL; direct UPDATE pr_url/updated_at; affected-row 404; no config | P3 aligned | pass (change/02-main.yaml) |
| POST | /api/v1/change/delete | 204; direct DELETE; actual testcase FK 409; missing 404; docs retained; no config | P3 aligned | pass (change/02-main.yaml) |
| POST | /api/v1/test-case/list | 200 ordered six-column public.testcase array after live-parent check; [] for no cases; missing parent404 | P4 aligned | pass (testcase/02-main.yaml) |
| POST | /api/v1/test-case/create | 201 exact {id}; one INSERT returning ID; default false/timestamps; missing parent FK404 | P4 aligned | pass (testcase/02-main.yaml) |
| POST | /api/v1/test-case/update | 204 empty; one scenario/updated_at UPDATE; affected rows0 means404; same-value writes execute | P4 aligned | pass (testcase/02-main.yaml) |
| POST | /api/v1/test-case/update-done | 204 empty; one done/updated_at UPDATE; omitted/null done=false; affected rows0 means404 | P4 aligned | pass (testcase/02-main.yaml) |
| POST | /api/v1/test-case/delete | 204 empty; one testcase DELETE; affected rows0 means404; no cascade or reload | P4 aligned | pass (testcase/02-main.yaml) |


## P3 change/document contracts and concurrency limits

Current reads explicitly scan the two change views. done_tc/total_tc/completed
are int64; completion is service-derived integer 100*done/total (zero for zero
total), matching P2. No details envelope, inline docs, version or testcase list
remains. Configuration-independent reads do not consult project configuration.

All mutations use domain requests and return only error, except creation's
existing ChangeIDRequest ID DTO. Config-dependent operations call project.Service.Config
using the actual parent ID: its stored slug determines all types/phases/docs.
Absent project or config on create uses the established unavailable-config 404;
unsupported function defaults are invalid-reference 400. UUID uniqueness is 409;
function parent-disappearance errors preserve PG causes through central APIs.
Title/open/pr-url/epic/delete need no configuration. Type filtering retains its
ordered intersection and clearing semantics; phases and document kinds reject
unknown values. No global lookup, fallback or additional dependencies exist.

Documents map public.doc ref_table='change', ref_id, doc_type, body, agent_edit,
current and created into domain.ChangeDocument (plus service-rendered html).
Specialized brief/spec/pr writes map to the same domain.ChangeDocumentSetRequest
used by set-document. Repeated identical writes append; no revision/version
compatibility is invented. Raw bodies are preserved while rendered HTML is
sanitized. Bulk artifacts pick the latest ID per current spec/pr kind if malformed
data contains duplicate current rows, preserving requested order and omitting
absent parents. Documents returns all current rows deterministically by doc_type,id.

Procedure preflights return 404 for already-missing parents. The preflight and
CALL are separate statements: **concurrent parent deletion may yield a successful
no-op**. Same-project epic validation can race with **concurrent changes to the
epic's project** because the database deliberately has no composite FK. A
concurrently deleted epic is still classified through its ordinary FK failure.
Document reads use preflight plus a live-parent join: deletion between those
statements can yield [] rather than 404, but cannot expose retained orphan docs.
No Go transaction or anonymous SQL workaround pretends to make these atomic.
Direct UPDATE/DELETE affected-row semantics hold at each statement boundary.

Deletion executes only DELETE public.change. Real testcase FK conflicts map to
409 `change has testcases and cannot be deleted`, preserving the PG cause.
No testcase cascade, recalculation or history procedure runs. public.doc is
append-only with polymorphic, non-FK references: historical AND current doc rows
survive parent deletion. Reads/set after an already absent parent return 404;
bulk artifacts omit that parent. No new purge or document blocker is invented.

P4 removed the final consumers of legacy domain.Change and both
RenderChange/RenderMutation adapters. The explicit Renderer.Render and its
parser/sanitizer remain for current document/artifact reads. Testcase has no
change dependency, renderer injection, transaction or post-mutation reads.

## P4 testcase contracts and concurrency limits

TestCase exposes exactly id, change_id, scenario, done, created and modified.
All requests cross API/service/repository as domain types, including list.
IDs remain int; tests exercise 1<<40 on this 64-bit toolchain. Echo binds and
validate checks request format; services independently reject nonpositive IDs
and trim/reject blank scenarios. Bool done retains false for omitted/null,
accepts explicit false/true, and rejects other JSON types at the API.

List checks live public.change then selects explicit six columns from
public.testcase ordered by id, with closed rows and preserved query/scan/iterator
causes. These two reads are not an atomic snapshot: concurrent parent deletion
can yield [] after a positive existence check. No Go transaction is implied.
Create inserts only change_id/scenario and scans only ID; FK SQLSTATE23503 maps
through Database to existing ErrTestCaseNotFound and preserves errors.Is/As.
Unknown SQL errors remain generic500 externally and inspectable internally.

Scenario/done updates always execute one UPDATE setting modified=now(), even
for identical values. Delete executes one DELETE. Zero affected rows means404;
success is error-only at both lower layers and an empty204 at the API. Each
statement has PostgreSQL statement atomicity. No preliminary read, parent
modified/count write, history row, cascade, reload or response rendering occurs.
Current change/epic views supply counters via independent reads. created and
unrelated fields remain unchanged; timestamps and parent invariance have real
HTTP evidence. /update-change remains absent (404).

Change deletion with live cases still returns409 and preserves parent, children
and current documents. Explicit testcase deletion permits change deletion204.
Already removed parents produce404 for get/documents/set-document and testcase
list/create. Append-only document rows retain all historic/current IDs, bodies
and flags, checked by the owned legacy SQL campaign, never merged into APIHydra
coverage. There was no runner SQL-postcondition hook at the merged P3 baseline;
the meaningful TestChangeDeletionRetainsAppendOnlyDocuments is retained and
adapted to create/delete its blocking testcase through the API.

### Obsolete assertion migration

- Procedure/function, transaction-embedding mocks, postmutation entity/list
  reads, testcase version/history and parent cascade assertions describe no
  current database capability. Replaced by TestRepositoryCurrentSixColumnList,
  TestRepositoryCreateOnlyID and TestRepositorySingleStatementMutations.
- TestScanChangeCurrentSchema and TestMutationChangeMissingCause in testcase
  disappear because testcase no longer reads changes. Six-column testcase scan
  errors and TestRepositoryTranslationKeepsExternalCauses preserve supported
  cause checks; P3's current change scans remain intact.
- TestServiceRendersMutationChangeSpecHTML and the temporary legacy renderer
  test disappear because mutation side data is forbidden. Explicit P3
  TestServiceSanitizesExplicitReads and TestRendererExplicitSource retain raw
  content/sanitization checks. Architecture tests now require adapter absence.
- TestRepositoryHistoryProcedures is removed: public.test_case_history,
  version, removed procedures, stored parent counters and cascade do not exist.
  Its valid FK, state, lifecycle and counter guarantees now run in the 96-step
  APIHydra P4 flow and retained owned HTTP/SQL campaigns. No environment-selected
  database test remains under internal/testcase.
- Legacy composite CRUD/delete-last and preserved/incremented-version tests
  are replaced by independent APIHydra reads and
  TestTestCaseCurrentStateAndSameValueTimestamps. Both boolean values, text edits,
  delete-last zero counts, repeated-delete404 and removed-move404 remain tested.
  Invalid/missing-row tests and all seven project/epic/change tests are retained;
  no SQL document-history assertions were dropped.

## P1 centralized error contract

`internal/error` now owns the module sentinels previously declared in services.
The authorized P1 relocation replaces those Go symbols with module-prefixed
central symbols; no module aliases or example APIs remain. `Wrap`, `Database`,
`InvalidPayload`, `Interpret`, `HTTP` and `ServerShutdown` serve existing callers.
Private repository pool interfaces are unit-test seams; P2 retains constructors
and migrates project/epic SQL, requests, routes and mutation responses as below.

| Condition | Retained public interpretation | Cause handling |
| --- | --- | --- |
| Module input validation | 400 `invalid {project/epic/change/test case} payload` | Distinct central sentinel plus actual validator/parser cause where present |
| Bind failure | 400 operation-specific existing payload message | Original Echo/decoder error retained |
| Missing entity | 404 `{module} not found` | pgx no-row cause retained where present |
| Invalid change reference | 400 `invalid change reference` | Existing reference checks retained |
| Project child conflict | 409 `project has dependencies and cannot be deleted` | Existing ErrProjectHasChanges; SQLSTATE 23503 cause retained |
| Epic child conflict | 409 `epic has changes and cannot be deleted` | SQLSTATE 23503 cause retained |
| Missing project or selected config on project/config | 404 `project configuration not found` | ErrProjectConfigNotFound with pgx cause; one inner join |
| Epic create missing/concurrently deleted parent | 404 `epic not found` | INSERT SELECT no-row or FK failure |
| Testcase insert SQLSTATE 23503 | 404 `test case not found` | Original nested `*pgconn.PgError` retained |
| Unexpected internal failure | 500 `Internal Server Error` | Full cause remains inspectable/loggable |
| Router not found / method mismatch | 404 `Not Found` / 405 `Method Not Allowed` | Direct and wrapped router errors interpreted centrally |
| Health ping failure | Existing 503 degraded health JSON | Contextual diagnostic log, no database detail in response |

No-row and FK classification is explicit at the boundary that previously owned
that mapping; an unrelated constraint or query failure does not become 404.
Higher layers pass handled errors unchanged unless adding context or HTTP
interpretation. HTTP output remains in API/startup, never in services/repositories.
Configuration/db panic values remain errors with inspectable underlying causes.

## P2 retained and changed contracts

- Project List/Get join vw_project to project for stored config and last_ref;
  counts/timestamps retain view authority. List order is modified DESC, id DESC.
  Empty lists serialize as [].
- Epic List/Get scan id, project_id, name, done_tc, total_tc, change_count,
  created and modified. Version is removed. Counters are int64 (change_count
  remains int); service derives completed = 100 * done_tc / total_tc, or 0
  for zero total. List order is created ASC, id ASC, scoped to positive project
  ID; an absent project produces []. No mapping lives in the repository.
- Creates return existing domain ID request DTOs as ID-only responses. Updates
  and deletes return error only and HTTP 204 with no body. No mutation rereads.
  The same domain request crosses API/service/repository; services trim names
  and independently reject invalid IDs/blank names. Echo binding and validate
  enforce API format. Context and classified causes are preserved.
- Config returns exactly project.config = config.slug, including project_docs,
  epic_docs, change_docs, change_phases, change_colors and change_types in stored
  order. Create keeps the database default; update remains name-only. No config
  mutation endpoint was added. Missing selected config and missing project both
  mean unavailable config (404); the default row is never a fallback.
- Parent deletion uses one direct DELETE. Actual project/epic/change FKs decide
  conflicts, including concurrent inserts. No child deletion, history call,
  transaction or check-then-delete workflow remains in these two modules.
  public.doc deliberately has no parent FK and is append-only: historical rows
  remain on parent deletion. No new delete-blocking constraint is invented.
- All configured project/epic document types are exposed, but neither module
  previously exposed document APIs. No unused document CRUD is introduced;
  P3 owns existing change/document operations. This removes no supported
  project/epic document operation.
- Options module, startup wiring, routes and options-only ChangePhase/ChangeType
  DTOs are removed after reference checks. Both former routes return 404.
- P1's intentionally broken project/epic 500 setup requests are removed. P2
  uses captured IDs and reserved SQL fixtures >=1000. Parent/child conflicts,
  custom/default/missing config, counts, zero completion and absent lists are
  independent of broken P3/P4 APIs. Legacy conflict scenarios are ported here;
  retained Go HTTP CRUD uses separate reads, including modified advancement.
  Its SQL document-retention test confirms both historical and current records
  survive project and epic deletion, separately from APIHydra counters.

See [checkpoint](backend-refactor-checkpoint.md) for criterion/test mapping and
actual verification, and [API coverage](../apih-tests/coverage.md) for assertions
and tool limitations. All five testcase successful operations are now measured separately from
negative/boundary scenarios.

## R2 duplicate-path reassessment

After merged P4 and reviewed R1, no actionable duplicate paths remain from the
original candidates. P1 centralized errors; P3/P4 removed duplicate change scans,
testcase mutation completion/recalculation/rendering and transaction/state paths.
Specialized change document operations already share SetDocument. Retain clear
SQL/scan/affected-row repetition with distinct DTOs, ordering and causes, local
repository test fakes, and the different owned-process/tool-command runners. No
generic helper or empty R2 branch is warranted. All retained parity tests and
408 APIHydra requests pass. The separate 011 validation-cause repair precedes
planned012 startup boundaries and does not represent a duplicate cleanup.

## R3 startup ownership and composition

The 012 extraction preserves the existing 34 method/path pairs and all 408 apih
requests. Main retains configuration, flags, signals and process exit. Lifecycle
owns lazy pool creation, listener acquisition, server assembly and cleanup in
that order; router construction only composes middleware and module wiring,
without opening/closing caller resources. CORS failures still occur after
pool/listener acquisition, retain centralized causes and release both resources.
Middleware/logging/error envelopes, 10s header/30s request timeouts and 10s shutdown
remain unchanged. Existing TestAPIConstructorRouteInventory now constructs the
actual router using its single retained expectation list. Handler parity tests
prove CORS/trailing-slash/recovery/context/logging behavior and safe error-write
failure logging; lifecycle tests prove failed shutdown unblocks via owned close.
Global configuration is intentionally unchanged until R4. Both coverage gates
pass 924/939 unit and 851/939 integration; 11 baseline lint findings await R5.

## R4 independent application configuration

`pkg/config.New() *Config` now returns an independent value using a fresh local
gookit loader; the package singleton and Get are removed. Main consumes the
returned instance before unchanged flag parsing. The fixed config/dev.yaml path,
YAML/ParseEnv, defaults, nonempty untrimmed environment overrides, final CLI
overrides and cause-preserving configuration panics retain their semantics.
No file-path parameter, loader interface, flag redesign, global reset or change
to R3 router/lifecycle ownership is introduced. Application configuration stays
separate from project-selected database configuration.

Sequential loads prove no field carryover or owner mutation leaks; failed loads
preserve earlier values and later defaults. Stable-input concurrent loads pass
the race detector. Ordinary executable tests preserve config-before-help/invalid
flags and file/environment/CLI precedence without injecting subprocess counters.
All408 APIHydra requests and full legacy HTTP/SQL parity pass. Fresh gates pass
925/939 unit and851/939 integration. Normal lint displays11 baseline diagnostics;
repeated-message limits mean this is not a total unique inventory. R5 will audit
without diagnostic caps, repair the actual debt and retain normal Make checks.

## R6 scoped mutation errors

Spec015 scopes14 one-use service errors into their immediate conditionals:
project update, epic update, nine specialized change writes and testcase
update/update-done/delete. Exact service calls, domain requests, contexts,
validation/cause chains, status/envelope behavior and empty204 remain unchanged.
Existing API test tables now also assert missing request/context/cause/call-order
parity; no syntax test, generic helper or result-variable rewrite was added.
All34 operations/408 APIHydra requests and full legacy parity pass. Required
Make checks are green; fresh unit911/925 and integration837/925 both pass.
The14-statement denominator change is Go instrumentation of scoped declarations,
not removed behavior or excluded production. Separate016 retains final audit
and owned-DB-outage scope.

## Shared doc and config API update

The earlier P3/P4 notes describe historical contracts. The current contract
moves all doc routes out of change. Doc reads filter public.doc directly without
a live-parent preflight. Both lists use descending ID order; current also filters
current=true. Insert validates the owning project's selected configuration, uses
project_docs/epic_docs/change_docs for the respective ref_table, and atomically
appends a row while retiring previous current rows of the same parent/type.
Created/modified fields are now named created_at/updated_at throughout the backend.
User/session/token implementation is deferred; existing SQL defaults remain authoritative.
Change types retain their established normalization but reject values outside
change_types; phases reject values outside change_phases. Config update replaces all six arrays.
Config deletion locks project writes while checking references and deleting.

| POST | /api/v1/doc/list | 200 all history, including comments and soft-deleted docs; ref_id/ref_table; id DESC | current | doc/02-main.yaml |
| POST | /api/v1/doc/list-active | 200 selected docs from vw_doc_active; doc_id DESC; nullable deleted_at | current | doc/02-main.yaml |
| POST | /api/v1/doc/details | 200 one doc by id; 404 absent | current | doc/02-main.yaml |
| POST | /api/v1/doc/insert | 201 ID only; append and select active document; fn_doc_insert(text,bigint,text,text,bool) | current | doc/02-main.yaml |
| POST | /api/v1/doc/comment-list | 200 all comments for ref_table/ref_id, including soft-deleted history; id DESC | current | pass (doc/02-main.yaml) |
| POST | /api/v1/doc/comment-insert | 201 ID only; validated live parent; fn_doc_comment_insert; no active selection | current | pass (doc/02-main.yaml) |
| POST | /api/v1/doc/comment-update | 204 empty; explicit body, including empty string; fn_doc_comment_update; missing/non-comment404 | current | pass (doc/02-main.yaml) |
| POST | /api/v1/doc/delete | 204 empty; fn_doc_delete retains history and removes active selection; missing404 | current | pass (doc/02-main.yaml); missing unit-tested |
| POST | /api/v1/config/list | 200 configs ordered by slug | current | config/02-main.yaml |
| POST | /api/v1/config/details | 200 config by slug; 404 absent | current | config/02-main.yaml |
| POST | /api/v1/config/insert | 201 slug only; 409 duplicate | current | config/02-main.yaml |
| POST | /api/v1/config/update | 204 full-array replacement; immutable slug; 404 absent | current | config/02-main.yaml |
| POST | /api/v1/config/delete | 204; 409 referenced by project; 404 absent | current | config/02-main.yaml |
| POST | /api/v1/change/update-after-change | 204 set nullable prerequisite; 400 invalid FK; 404 absent target | current | change/02-main.yaml |

Change details includes nullable `after_change_id` and `after_change_name`, and
the plain `epic_name` without an ID suffix. The prerequisite display name is
the prerequisite change title followed by ` (#<id>)`; the ID remains the
editable association value. No run scheduler or prerequisite-completion
enforcement is added.
