# P4 route, schema and error ledger

Authority: read-only `../../docs/backend-architecture.md`, `../../db/init.sql`
and `../../db/seed.sql`. The inventory is **34 registered method/path pairs**:
32 after P2 plus documents and set-document. APIHydra exercises all 16 change,
11 project/epic, five testcase and two health operations successfully (34/34).
No authentication middleware or invented authentication contract exists.

| Method | Path | Current behavior / database source | Target and deferred pass | APIHydra |
| --- | --- | --- | --- | --- |
| GET | /api/v1/health | 200 health JSON after pool ping; 503 degraded JSON on ping error | Retain; exact body unit tested | pass |
| GET | /api/health | Same health alias and database ping | Retain | pass |
| POST | /api/v1/project/list | 200 array; current view columns; explicit deterministic ordering | P2 aligned | pass (p2-steps.yaml) |
| POST | /api/v1/project/get | 200 database-backed details | P2 aligned | pass (p2-steps.yaml) |
| POST | /api/v1/project/config | 200 selected config slug and all six ordered arrays; unavailable join 404 | P2 aligned, no fallback | pass (p2-steps.yaml) |
| POST | /api/v1/project/create | 201 {id}; one INSERT returning ID | P2 aligned | pass (p2-steps.yaml) |
| POST | /api/v1/project/update | 204 empty; one name/modified UPDATE, including same name | P2 aligned | pass (p2-steps.yaml) |
| POST | /api/v1/project/delete | 204 empty; one DELETE, FK conflict 409, missing 404 | P2 aligned | pass (p2-steps.yaml) |
| POST | /api/v1/epic/list | 200 array; current view columns; explicit deterministic ordering | P2 aligned | pass (p2-steps.yaml) |
| POST | /api/v1/epic/get | 200 database-backed details | P2 aligned | pass (p2-steps.yaml) |
| POST | /api/v1/epic/create | 201 {id}; one INSERT returning ID | P2 aligned | pass (p2-steps.yaml) |
| POST | /api/v1/epic/update | 204 empty; one name/modified UPDATE, including same name | P2 aligned | pass (p2-steps.yaml) |
| POST | /api/v1/epic/delete | 204 empty; one DELETE, FK conflict 409, missing 404 | P2 aligned | pass (p2-steps.yaml) |
| POST | /api/v1/change/list | 200 current vw_change_list columns; service int64 completion; modified DESC,id; [] for absent project | P3 aligned | pass (p3/steps.yaml) |
| POST | /api/v1/change/get | 200 current vw_change_details; flat fields, nullable references, no inline docs/version/testcases; 404 missing | P3 aligned | pass (p3/steps.yaml) |
| POST | /api/v1/change/rendered-artifacts | 200 artifacts wrapper; ordered deduplicated IDs; live-parent current doc spec/pr, sanitized HTML; absent IDs omitted | P3 aligned | pass (p3/steps.yaml) |
| POST | /api/v1/change/create | 201 exact {id}; selected config must support backlog/brief; UUIDv7 default or preserved caller UUID; fn_change_insert only | P3 aligned | pass (p3/steps.yaml) |
| POST | /api/v1/change/update-epic | 204; targeted parent/epic project preflight, sp_change_epic_update; nil detaches; no config | P3 aligned | pass (p3/steps.yaml) |
| POST | /api/v1/change/update-phase | 204; selected-config phase validation then sp_change_phase_update | P3 aligned | pass (p3/steps.yaml) |
| POST | /api/v1/change/update-open | 204; explicit bool; direct UPDATE open/modified; affected-row 404 | P3 aligned | pass (p3/steps.yaml) |
| POST | /api/v1/change/update-change-types | 204; ordered trim/dedup/filter using selected config; direct UPDATE types/modified, including clears | P3 aligned | pass (p3/steps.yaml) |
| POST | /api/v1/change/update-title | 204; existence preflight then sp_change_title_update; DB whitespace normalization; no config | P3 aligned | pass (p3/steps.yaml) |
| POST | /api/v1/change/update-brief | 204; service maps brief to ChangeDocumentSetRequest; sp_change_doc_set appends | P3 aligned | pass (p3/steps.yaml) |
| POST | /api/v1/change/update-spec | 204; service maps spec to ChangeDocumentSetRequest; sp_change_doc_set appends | P3 aligned | pass (p3/steps.yaml) |
| POST | /api/v1/change/update-pr | 204; service maps pr to ChangeDocumentSetRequest; sp_change_doc_set appends | P3 aligned | pass (p3/steps.yaml) |
| POST | /api/v1/change/update-pr-url | 204; nonblank http(s) URL; direct UPDATE pr_url/modified; affected-row 404; no config | P3 aligned | pass (p3/steps.yaml) |
| POST | /api/v1/change/delete | 204; direct DELETE; actual testcase FK 409; missing 404; docs retained; no config | P3 aligned | pass (p3/steps.yaml) |
| POST | /api/v1/change/documents | 200 current docs ordered doc_type,id with raw body and sanitized html; [] if none; live-parent preflight; no config | P3 aligned | pass (p3/steps.yaml) |
| POST | /api/v1/change/set-document | 204; explicit agent_edit, nonblank kind/body, selected-config validation; sp_change_doc_set appends even identical body | P3 aligned | pass (p3/steps.yaml) |
| POST | /api/v1/test-case/list | 200 ordered six-column public.testcase array after live-parent check; [] for no cases; missing parent404 | P4 aligned | pass (p4/steps.yaml) |
| POST | /api/v1/test-case/create | 201 exact {id}; one INSERT returning ID; default false/timestamps; missing parent FK404 | P4 aligned | pass (p4/steps.yaml) |
| POST | /api/v1/test-case/update | 204 empty; one scenario/modified UPDATE; affected rows0 means404; same-value writes execute | P4 aligned | pass (p4/steps.yaml) |
| POST | /api/v1/test-case/update-done | 204 empty; one done/modified UPDATE; omitted/null done=false; affected rows0 means404 | P4 aligned | pass (p4/steps.yaml) |
| POST | /api/v1/test-case/delete | 204 empty; one testcase DELETE; affected rows0 means404; no cascade or reload | P4 aligned | pass (p4/steps.yaml) |


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
| Module input validation | 400 `invalid {project/epic/change/test case} payload` | Distinct central sentinel per module |
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
