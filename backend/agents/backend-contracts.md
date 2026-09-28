# P0 route and schema ledger

Authority: read-only `../../docs/backend-architecture.md`, `../../db/init.sql`
and `../../db/seed.sql`. P0-01/P0-06 inventory: **33 registered method/path pairs**.
Only the two health aliases have passing APIHydra replacements. All other pairs
remain in the operation denominator; the table describes code inspection, not
successful integration execution. There is no authentication middleware in the
current server; no authentication contract is invented here.

| Method | Path | Current behavior / database source | Target and deferred pass | APIHydra |
| --- | --- | --- | --- | --- |
| GET | /api/v1/health | 200 health JSON after pool ping; 503 degraded JSON on ping error | Retain; exact body unit tested | pass |
| GET | /api/health | Same health alias and database ping | Retain | pass |
| POST | /api/v1/project/list | 200 list intended; selects last_ref absent from vw_project | P2: join project.last_ref; retain view count/timestamps | blocked |
| POST | /api/v1/project/get | 200 entity intended; same invalid view projection | P2: current-schema details | blocked |
| POST | /api/v1/project/create | Inserts project then broken details read; intended 201 entity | P2: 201 ID only; no reload | blocked |
| POST | /api/v1/project/update | Updates name/modified then broken details read; intended 200 entity | P2: empty 204; error-only methods | blocked |
| POST | /api/v1/project/delete | Guarded project delete against change/epic children; empty 204 | P2: preserve constraints/errors | untested |
| POST | /api/v1/epic/list | Selects obsolete version/completed from vw_epic; intended 200 list | P2: view fields/counts; derived completion if retained | blocked |
| POST | /api/v1/epic/get | Same obsolete epic projection; intended 200 entity | P2: current-schema details | blocked |
| POST | /api/v1/epic/create | Insert epic then obsolete details projection; intended 201 entity | P2: 201 ID only | blocked |
| POST | /api/v1/epic/update | Transaction/history path; intended 200 entity | P2: name/modified update, empty 204, no Go transaction | blocked |
| POST | /api/v1/epic/delete | Guarded epic delete; empty 204 | P2: preserve child integrity | untested |
| POST | /api/v1/options/change-phases-list | Reads removed change_phase; hardcoded color cases | P2: remove; project configuration from config.change_phases/change_colors | blocked |
| POST | /api/v1/options/change-types-list | Reads removed change_type | P2: remove; project-selected config.change_types | blocked |
| POST | /api/v1/change/list | 200 list from vw_change_list; P0 fixed total_tc scan and derived completion | P3: review count widths/domain; preserve ordering/nullability | untested |
| POST | /api/v1/change/get | P0: 200 flat ChangeDetails from vw_change_details; missing row maps to 404 | P3: separate document and testcase reads | untested |
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
| POST | /api/v1/test-case/list | Uses public.test_case (absent); intended 200 list | P4: public.testcase, retained fields | blocked |
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
