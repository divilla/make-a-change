# P2 route, schema and error ledger

Authority: read-only `../../docs/backend-architecture.md`, `../../db/init.sql`
and `../../db/seed.sql`. The inventory is **32 registered method/path pairs**:
33 after P1, plus project/config, minus two options routes. APIHydra includes all
11 P2 operations and both health aliases. Remaining change/testcase successes
await P3/P4; those operations stay in the denominator. There is no authentication
middleware or invented authentication contract. See the checkpoint for actual
campaign exits and measurements.

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
| POST | /api/v1/change/list | 200 list from vw_change_list; P0 fixed total_tc scan and derived completion | P3: review count widths/domain; preserve ordering/nullability | untested |
| POST | /api/v1/change/get | P0: 200 flat ChangeDetails from vw_change_details; missing row maps to 404 | P3: separate document and testcase reads | 400/404 pass; success untested |
| POST | /api/v1/change/rendered-artifacts | Reads obsolete change.brief/spec/pr and renders | P3: explicit reads of current doc rows, safe rendering | blocked |
| POST | /api/v1/change/create | Calls fn_change_insert then obsolete full-entity scan | P3: fn_change_insert returns ID; 201 ID only | blocked |
| POST | /api/v1/change/update-epic | Old state/transaction/recalculation path; intended 200 entity | P3: current epic_id + sp_change_epic_update; validate project relation; 204 | blocked |
| POST | /api/v1/change/update-phase | Removed lookup/reference/state paths; intended 200 entity | P3: project config + sp_change_phase_update; 204 | blocked |
| POST | /api/v1/change/update-open | Old state/transaction path; intended 200 entity | P3: change.open and modified; 204 | blocked |
| POST | /api/v1/change/update-change-types | P0: error-only direct update, 204 empty, 404 missing; service still reads removed change_type | P3: project-selected config validation; retain normalization | blocked |
| POST | /api/v1/change/update-title | Old state/transaction path around sp_change_title_update | P3: use existing procedure; 204, no reload | blocked |
| POST | /api/v1/change/update-brief | Calls removed sp_change_brief_update; old state/transaction | P3: sp_change_doc_set with brief; 204 | blocked |
| POST | /api/v1/change/update-spec | Calls removed sp_change_spec_update; old state/transaction | P3: sp_change_doc_set with spec; 204 | blocked |
| POST | /api/v1/change/update-pr | Calls removed sp_change_pr_update; old state/transaction | P3: sp_change_doc_set with pr; 204 | blocked |
| POST | /api/v1/change/update-pr-url | Old state/transaction path before pr_url update | P3: change.pr_url and modified; 204 | blocked |
| POST | /api/v1/change/delete | Old test_case/procedure/recalculation path | P3/P4: explicit deletion respecting testcase FK and doc ownership; 204 | blocked |
| POST | /api/v1/test-case/list | Uses public.test_case (absent); intended 200 list | P4: public.testcase, retained fields | 404 pass; success blocked |
| POST | /api/v1/test-case/create | Removed fn_test_case_insert; composite mutation response | P4: testcase insert; 201 ID only | blocked |
| POST | /api/v1/test-case/update | Removed sp_test_case_update and old state | P4: testcase.scenario; 204 | blocked |
| POST | /api/v1/test-case/update-done | Removed procedure, old history/composite response | P4: testcase.done; 204 | blocked |
| POST | /api/v1/test-case/delete | Removed sp_test_case_delete; composite response | P4: testcase delete; 204, independent reads | blocked |


## Limited P0 field correction

`ChangeDetails` now embeds `ChangeListItem` and adds `pr_url` and `created`.
It has no `change`/`test_cases` envelope, obsolete `version`, or fabricated
empty document/rendered fields. The old interface's wrapper could not be
implemented by the old repository slice and required unrelated convenience
reads. The authorized minimal correction exposes only this view's data.
`completed` remains a real database-backed derivation:
`coalesce(100 * done_tc / nullif(total_tc, 0), 0)`; it is not a compatibility zero.
Null ref/slug/epic fields remain nullable. `modified` and `created` come from
the views. Current int16 counter widths remain a P3 follow-up.

`Change` itself and old mutation scans remain for P3/P4. The legacy
`TestScanChangeCurrentSchema`/`TestGetStateCurrentSchema` names describe old
mock layouts, not evidence that those queries match today's database. Their
success must not be cited as SQL compatibility. P0 adds distinct current-view
query/scan tests and changes the old service rendering assertion to current
flat details. Explicit document rendering still has its separate tests.

Configuration maps project.config to config.slug, with no default fallback;
project_docs, epic_docs, change_docs, change_phases, change_colors and change_types
all come from config. Documents map ref_id/ref_table/doc_type/body/current/
agent_edit/created to public.doc and the sp_*_doc_set procedures. No new schema,
Go transaction, authentication behavior, or dependency is introduced.

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
and tool limitations. Error-only change/testcase scenarios remain diagnostic;
they are not successful operation coverage.
