# 007 — Backend project and epic alignment

## Authorized execution contract

Implement this specification now. User approved the entire sequential factory implementation/review/merge process and explicit backend contract alignment. P1 is merged in origin/dev at 131e075. Read AGENTS.md, docs/backend-architecture.md, this specification, and backend/agents/backend-refactor-checkpoint.md. Reuse the P1 error APIs. The user's scope overrides change-code defaults: do not read PRD, frontend, CLI, skeleton, or unrelated architecture references; backend architecture is authoritative. Do not stop to request permission for the explicit API/domain contract changes below.

All implementation writes stay under backend/. This committed specification, the existing official agent/backend-refactor-plan.md (top status only), and backend-local documents are authorized. Root scripts, SQL/db files, docs, dependencies and root AGENTS.md are read-only. Append the change-code implementation log to backend/implementation-log.md, not root implementation-log.md; use its prescribed date/count format and exclude the log itself. Do not commit or push: the calling factory script owns that.

Numerical coverage targets (>95% unit, >=90% APIHydra integration statement counts) apply to the final refactor, not this intermediate branch. Implement meaningful tests, measure both suites, keep strict gate failures honest, and finish this pass despite numerical shortfalls. Fix failures introduced by this pass; report unrelated existing format/lint debt. No artificial coverage, skipped failing scenarios, profile mixing, or test weakening. Continue implementation without asking solely for coverage.

Use the change-code and apih-build skills with the explicit backend-local paths and final-only coverage override above. Installed APIHydra docs are available in /home/vito/go/src/apihydra at binary revision c3947513e2a4b948ce732dec60e090a5e83be744; confirm actual installed capabilities rather than assuming latest documentation. Its curl defaults require retries:-1 for nonreplayed mutation/error testing.

Keep a compact backend checkpoint and refreshed backend contract ledger, including each P2 criterion mapped to meaningful tests and actual verification exits/counts. Replace P1's intentionally broken project/epic expected500 scenarios with successful migrated contracts. Do not keep fixtures dependent on those former failures.

## Scope decisions

- Branch/spec: change/007-backend-project-epic-alignment and agent/specs/007-backend-project-epic-alignment.md.
- Preserve the existing five POST operations under each of /api/v1/project and /api/v1/epic. Add only POST /api/v1/project/config, accepting domain.ProjectIDRequest. Remove the two options routes, module, startup wiring, and obsolete options-only domain DTOs after reference checks.
- Project/epic document procedures exist but neither module currently exposes document APIs. Architecture explicitly says not to invent unused CRUD for symmetry. P2 should expose all configured document-type arrays but not introduce new project/epic document endpoints. Record this deliberate scope decision in the contract ledger; P3 handles existing change/document behavior. This is not removal of supported project/epic document behavior.
- No application dependency changes, schema changes, Go transactions, CLI/frontend/PRD reads or edits.

## Concrete acceptance criteria

P2-01: Project reads return database-backed ID, name, configured slug, last_ref, created, modified, and change_count. Preserve last_ref by querying its real table column, not the view. List returns [] for empty data and a documented deterministic order (modified DESC, id DESC is consistent with current vw_project intent).

P2-02: Epic reads return ID, project_id, name, done_tc, total_tc, change_count, created, modified. Remove version, which has no current database authority. Preserve completed as an explicitly service-derived integer percentage from database counters; use 0 when total is zero. Record this retained-contract choice: completed = 100 * done / total, mapped in service; widen int16 count fields to int/int64 because count(*) returns bigint. Do not put mapping in repository. Lists scoped to positive project ID return [] for no matches (including an absent project, matching current list semantics), ordered created,id to preserve existing repository ordering.

P2-03: Creates return only a domain ID DTO and HTTP 201. Updates/deletes return error only in service/repository and HTTP 204 with genuinely empty body. No Get after a mutation. Same-name update succeeds and updates modified, as one normal UPDATE; no obsolete history/version procedure calls.

P2-04: Positive IDs and trimmed nonempty names are independently validated by service methods, including direct callers. API uses Echo and validate for request-format rules. The same domain request reaches API/service/repository wherever applicable; stop passing raw int/name across those boundaries. Request contexts propagate unchanged. All bind, validation, database, missing-row, and conflict handling use the P1 centralized-error public API.

P2-05: Project config returns slug plus all six arrays from exactly the project's selected config slug: project_docs, epic_docs, change_docs, change_phases, change_colors, change_types. Preserve array order and entries; no hardcoded colors, lookup tables, or default fallback. Missing project or missing selected config returns an explicit centralized missing-resource error. A fixture with selected slug absent while default exists must fail, not return default. Expose Project.Config in project details to show the stored selection. Add a centralized ErrProjectConfigNotFound with a safe 404 message when the selected config row is missing; a nonexistent project on this endpoint may use that same unavailable-config result, documented and tested. Do not add configuration creation/update operations solely for fixtures: existing create keeps the DB default and update remains name-only.

P2-06: Project deletion is one direct DELETE. Actual FK dependencies on epic or change yield 409, missing ID yields 404, empty project deletion succeeds. Epic deletion is one direct DELETE; change.epic_id FK yields 409, missing ID yields 404, unreferenced epic succeeds. No child deletion, no history write, no multi-call pseudo-atomic workflow. This preserves parent-conflict intent and lets PostgreSQL enforce concurrent safety. The old ErrProjectHasChanges wording was inaccurate for epics; use `project has dependencies and cannot be deleted` for the existing central project conflict sentinel, updating all affected tests and error ledger. public.doc has deliberately no FK and is append-only: keep retained document records when deleting a parent; do not erase historical docs or invent a delete-blocking constraint. Record this explicitly.

P2-07: Existing options endpoints are unregistered (404); config has project ownership and no stale global-options DTOs/routes. Startup remains buildable and route inventory reconciles old 33 operations to 32 after +1 project config / -2 options (recount current code after P1).

P2-08: Meaningful unit tests cover every criterion; APIHydra tests execute each retained/new P2 operation, successful writes followed by independent reads, and negative validation/missing/config/dependency cases. Both coverage suites run honestly. Numerical targets apply at final result; intermediate coverage failure alone must not stop merging.

## SQL details

Project List/Get can select `v.id, v.name, p.config, p.last_ref, v.created, v.modified, v.change_count FROM public.vw_project v JOIN public.project p ON p.id=v.id`, with WHERE v.id=$1 for Get and explicit ORDER BY for List. This fixes the real last_ref bug without deleting a valid field. An equally simple base-table SELECT with count subquery is valid, but the view should remain the source of existing count behavior.

Project Create: `INSERT INTO public.project(name) VALUES($1) RETURNING id`. Project Update: `UPDATE public.project SET name=$2, modified=now() WHERE id=$1`; zero affected rows maps missing. Project Delete: `DELETE FROM public.project WHERE id=$1`; map FK violation centrally, zero affected rows missing. This avoids current conditional-delete/second-query race entirely.

Config: `SELECT c.slug,c.project_docs,c.epic_docs,c.change_docs,c.change_phases,c.change_colors,c.change_types FROM public.project p JOIN public.config c ON c.slug=p.config WHERE p.id=$1`. Inner-join absence deliberately produces an error for either missing project or missing configuration; no fallback query. Use the explicit config-unavailable error selected above for missing join result; do not add a fallback query. Use domain.Config expanded from its current four fields, or a deliberately named domain.ProjectConfig. P3 will consume it too.

Epic List/Get: explicit current view column list `id,project_id,name,done_tc,total_tc,change_count,created,modified`. No version/completed SQL columns. Repository scans database values; service derives completed.

Epic Create should avoid repository business preflight `ensureProject`: `INSERT INTO public.epic(project_id,name) SELECT id,$2 FROM public.project WHERE id=$1 RETURNING id`; zero rows gives missing project and ordinary FK protection still applies. This performs one atomic statement. Preserve the existing epic-create missing-parent 404 `epic not found` using INSERT SELECT. Map a concurrent FK failure to that same existing missing-parent contract.

Epic Update: `UPDATE public.epic SET name=$2,modified=now() WHERE id=$1`. Epic Delete: `DELETE FROM public.epic WHERE id=$1`. Check RowsAffected zero; direct FK delete conflict remains 409.

Use a narrow pgx-backed DB interface (Query, QueryRow, Exec) for actual repository unit tests, because real *pgxpool.Pool is not mockable without a seam. Interface exists to test database boundary failures and parameter/scan correctness, not generic CRUD abstraction. NewRepo accepts the interface or wraps concrete pool in an internal testable field. Keep simple explicit scans/SQL, no new mocking dependencies.

## Required tests and fixtures

Unit API tests: each route's success status/body; exact create key set {id}; update/delete zero-length bodies; malformed JSON, wrong typed fields, invalid IDs/names; consistent error envelopes/status; options unregistered; request context propagates. Unit service tests: trimmed input reaches repo, blank/invalid values make no calls, repo errors preserved, config missing not substituted, completed zero/nonzero mapping, mutations never reread. Unit repository tests: parameter order, column scan layout incl config/last_ref and large counts, list ordering/empty slice, query/scan/iteration errors, deferred row closure, Exec error and zero/one RowsAffected, pgx missing rows, foreign-key conflict, no follow-up reads. A fake must assert interactions and relevant results rather than replaying implementation lines for coverage.

Add backend/apih-tests/fixtures.sql loaded by backend/scripts/api_coverage.py after unchanged db/init.sql and db/seed.sql, only into the owned cluster. Fail fixture SQL with ON_ERROR_STOP. It can seed fixed IDs >=1000 without colliding with normal identity-created IDs (small dynamically captured created objects); document reserved fixture IDs. Include:
- a custom config with every one of six arrays different from default; include custom phases/colors to prove no hardcoded mapping;
- project 1001 selected custom config; project 1002 selected default; project 1003 selected nonexistent slug;
- empty project 1004; project 1005 with empty epic 1101; project 1006 with change 1201 (project deletion conflict);
- epic 1102 and linked change 1202 with two testcases, one done, to verify 1/2 counters and completed50; a separate zero-testcase epic;
- if testing append-only retention policy, a doc row for an otherwise deletable project/epic, though verification of retained SQL records belongs to DB contract tests because no read route exists.
Use valid current public.change/public.testcase columns, no old test_case/history/version fixtures. Avoid fixtures coupling to still-broken P3 API; SQL setup is legitimate, real requests exercise P2.

APIHydra workflows live in a P2 steps file with serial producer/consumer order, all IDs interpolated into request bodies. Every P2 endpoint uses POST bodies, so installed apih's no-path-interpolation limitation poses no blocker. Create a project -> capture id -> get -> update -> get -> create epic -> capture id -> list/get -> update -> get -> delete project expects409 -> delete epic204 -> get epic404 -> delete project204 -> get project404. Separate fixed fixtures exercise config custom/default/missing, populated counts, project direct-change conflict, epic-change conflict, empty list, invalid IDs and whitespace names. Invalid JSON/types exercise bind failures. Always explicit expected_status and substantive expected_body for reads/errors. APIHydra's subset comparison cannot prove ID-only/exact-key or 204-empty; API unit tests prove these and coverage.md must disclose the integration assertion limit.

Preserve only useful legacy Go HTTP scenarios; old full-create body expectations must be aligned or their retained scenarios ported before removal. They do not contribute APIHydra coverage. Runner changes require existing tooling-test scenarios updated to exercise fixture order/failure propagation without real external services. Run required check, coverage, deps-audit, api-test; report known unrelated baseline failures and actual measured counts, continue agreed factory/review/merge sequence.

## Verification and delivery

Run the required make -C backend check, coverage, deps-audit, api-test commands after implementation and any review fix. If check short-circuits on known unrelated format differences, also run make -k check with GOLANGCI_LINT_CACHE in /tmp to expose all checks. Format only touched Go files unless their directly dependent generated changes require more. Run tooling-test for fixture-runner changes. Store logs under backend/.coverage/verification or /tmp/mch-p2*. Record exact covered/total counts with package gaps; no fresh invalid campaign accepted. Inspect final diff/scope and git diff --check. This pass ends ready for factory commit/review, never promotion.
