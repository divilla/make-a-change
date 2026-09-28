# Backend refactor checkpoint — 014 local patterns (R5 reviewed)

2026-09-28; `change/014-backend-local-patterns`, specification HEAD `99c9b8a`
over merged R4 `d35ea6852fecc0d6a518bdb37ab188ffa127e2a7`. Factory implementation `1df4646` is published. Native review against the pinned
R4 base passed on its first pass without findings or fixes. All required checks
and both strict coverage gates pass. The supervisor is publishing this checkpoint
before the authorized squash merge to dev. No stage/production promotion occurred.
The first native-review launch was rejected before execution because automatic
approval review timed out; after verifying the clean published state and that
no review had started, one permitted retry succeeded. No implementation retry
or duplicate review process occurred. Transcripts: `/tmp/mch-r5-code-spec.log`
and `/tmp/mch-r5-review-loop.log`.

Next: complete the authorized014 merge, then audit R6 compact-handler candidates
for015; keep the separate016 final
failure-integration campaign out of this pass.

## Fresh baseline and repaired owners

The checkout was clean on the expected branch. Before editing, repeated normal
format/check plus uncapped lint matched the supervisor's baseline. Logs:
`/tmp/mch-r5-implementation-baseline-{format,check,uncapped}.log`.

| Actual finding at baseline | Owner and repair |
| --- | --- |
| Three errcheck findings at shared/client.go:43,96,117 | NewClient/Get/Post each retain one immediate deferred close; private closeResponseBody reports Close errors with nonfatal testify assertion and method/path context. |
| Missing shared package comment at shared/client.go:1 | One package comment describes legacy HTTP client/fixture cleanup. |
| Five exported-comment findings at shared/client.go:19,28,61,88,105 | Client, NewClient, CleanupProject, Get and Post now describe their actual responsibilities. Existing BaseURL comment remains. |
| Four missing package comments at epic/api.go:1, health/api.go:1, project/api.go:1, markdown/markdown.go:1 | One concise package comment at each owning source location. |
| No formatter findings | Pinned formatting applied only to the six edited/new Go files; no unrelated mechanical diff. Domain already has its package comment and was not edited. |

Uncapped lint reported **13 issues: 3 errcheck, 10 revive**. Normal check displayed
11 because repeated diagnostics are capped; epic/project/health/markdown account
for the full four-package inventory. This resolves the earlier checkpoint's
candidate debt without assuming that its displayed subset was exhaustive.

## Scope and acceptance evidence

Only shared harness close-error reporting changes runtime behavior. API/domain/
SQL, status/envelope shapes, ID-only201/empty204, configuration selection,
append-only documents, ordering/nullability and resource ownership remain.
Timeout, environment base URL, payloads, status returns, original require/assert
calls, decode diagnostics and project fixture cleanup order are unchanged.
No dependencies, tools, thresholds, suites, server hooks or DB fixtures changed.

| Criterion | Evidence |
| --- | --- |
| R5-01 baseline and clean check | Fresh capped/uncapped inventory above; final normal check passes formatting, zero lint issues, vet, uncached race tests, all45 Python tooling tests and Go suite-validator tests. No suppression/configuration changes. |
| R5-02 owned response cleanup | TestClientResponseCleanup proves NewClient health method/path, five-second timeout and base-URL override; GET/POST each close once with nil output or decoded JSON, retaining status/payload/header behavior. Each successful acquisition registers its defer before subsequent reads/assertions. |
| R5-03 failure verification | TestCloseResponseBody records genuine Close errors through testify's testing interface, checks useful context, one close and continuing defers. TestClientCleanupFailures launches the real test executable: health Close failure, GET decode failure, POST decode failure and POST read failure all produce exit1, a close diagnostic and exactly one close; primary errors/POST body text and other registered cleanup remain visible. Deliberately failing children are asserted by passing parent tests. Complete legacy HTTP/SQL suite passes. |
| R5-04 comments/formatting | Pinned formatter and unchanged revive checks pass; retained behavioral suites pass. No comment-text tests or unrelated comment rewrites. |
| R5-05 contract parity | Existing production unit/race tests and all five unchanged APIHydra suites pass; all408 requests/34 operations retained. Existing route-ledger tooling audit passes. Source audit finds no Go transaction APIs or prohibited service imports, and error constructors remain confined to internal/error. |
| R5-06 compact delivery | This checkpoint, only the official plan's top status, and backend/implementation-log.md record results and review/R6 handoff. Log location follows spec014's explicit backend-only override. |

The focused harness tests remain under api-tests/shared and outside production
coverage. Private recording bodies/transports test the existing client without
changing its public surface. Test-only default-transport substitution is scoped
to sequential tests and restored by cleanup. No real external service is used
by these focused tests. Existing integration runners retain owned resources.

## Commands and results

Commands run from root except those marked backend. Every final command below
exited0. Final logs are `/tmp/mch-r5-{targeted,check-final,coverage-final,
deps-audit-final,api-test-final,legacy-api-test}.log`.

| Command actually run | Exit/result |
| --- | --- |
| `GOLANGCI_LINT_CACHE=/tmp/mch-r5-lint-cache make -C backend format-check` (baseline) | 0, no differences |
| `GOLANGCI_LINT_CACHE=/tmp/mch-r5-lint-cache make -C backend check` (baseline) | 2 at lint; 11 displayed issues |
| Backend: `GOLANGCI_LINT_CACHE=/tmp/mch-r5-lint-cache golangci-lint run --no-config --default standard --enable revive --timeout 5m --max-same-issues=0 --max-issues-per-linter=0 ./cmd/... ./internal/... ./pkg/... ./api-tests/...` (baseline) | 1; complete13-finding inventory |
| Backend: `golangci-lint fmt --no-config --enable gofumpt --enable goimports api-tests/shared/client.go api-tests/shared/client_test.go internal/epic/api.go internal/health/api.go internal/project/api.go pkg/markdown/markdown.go` | 0; diff inspected |
| Backend: `go test -short -count=1 -race ./api-tests/shared` | 0, focused harness cleanup/parity tests |
| `GOLANGCI_LINT_CACHE=/tmp/mch-r5-lint-cache make -C backend check` | 0; normal unchanged checks all enabled and green |
| `make -C backend coverage` | 0; 925/939, >95% gate passes |
| `make -C backend deps-audit` | 0; no vulnerabilities |
| `make -C backend api-test` | 0; five APIHydra suites pass, 851/939, >=90% gate passes |
| `make -C backend legacy-api-test` | 0; change/epic/health/project/shared/testcase packages pass |
| `git diff --check` | 0; tracked/untracked scope inspected |

No failed implementation attempts, unavailable prerequisites, unrun required
commands, or skipped retained scenarios remain. The intentional child-process
failures prove failure propagation; tooling's expected simulated cleanup-error
messages also belong to passing regression tests. Native review remains the
factory's next step. No performance/toolchain change requires benchmarks/Docker.

## Fresh production statement counts and gaps

| Production package | Unit covered/total | APIHydra covered/total |
| --- | --- | --- |
| cmd/server | 77/91 | 75/91 |
| internal/change | 398/398 | 378/398 |
| internal/domain | 0/0 | 0/0 |
| internal/epic | 115/115 | 107/115 |
| internal/error | 48/48 | 36/48 |
| internal/health | 21/21 | 16/21 |
| internal/project | 117/117 | 110/117 |
| internal/testcase | 108/108 | 99/108 |
| pkg/config | 31/31 | 22/31 |
| pkg/markdown | 10/10 | 8/10 |
| **Aggregate** | **925/939 (98.5091%)** | **851/939 (90.6283%)** |

All14 unit gaps are main.go process orchestration, lines15–33. All other
statement-bearing production packages have full unit statement coverage; domain
has no executable statements. Existing ordinary-executable tests assert startup
behavior without injecting subprocess counters into production profiles.

APIHydra retains88 gaps: server16, change20, epic8, error12, health5, project7,
testcase9, config9, markdown2. Server gaps cover acquisition/CORS/serve/shutdown,
main error exit and error-write diagnostics. Config gaps cover load/decode panic,
defaults, explicit environment overrides and leading-colon address handling.
Other gaps include SQL query/scan/iteration/Exec errors, entropy/concurrent parent
failures, degraded health, Markdown conversion errors and central nil helpers.
These are retained for the separate final failure-integration reassessment;
neither aggregate target is unmet in R5.

Fresh profiles, structural denominator, reports, counters and provenance are in
`backend/.coverage/unit` and `backend/.coverage/api`; legacy results are only in
`backend/.coverage/legacy`. Go source/test hashes recorded by both measured runs
match the final implementation, including the untracked new harness test. Each
integration runner stopped its own server and disposable PostgreSQL cluster;
no shared database reset, unit/legacy profile mixing or package exclusions.
