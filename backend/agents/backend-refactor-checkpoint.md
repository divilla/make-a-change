# Backend refactor checkpoint — 012 startup boundaries (R3)

2026-09-28; `change/012-backend-startup-boundaries`, merged base `4c25d70`.
Factory implementation `8bc82d4` is published. Native review against the pinned
dev base completed with no actionable findings or fixes; unit and repeated
race-enabled server tests passed. The supervisor is publishing this checkpoint
before the authorized squash merge to dev. No transient retry was needed and
no stage/production promotion occurred. Required verification remains incomplete
only for the known 11 R5 lint findings; both strict coverage gates pass.

Next after merge: R4 `change/013-backend-config-isolation`, preserving fixed
config/dev.yaml, precedence, wrapped panic behavior and startup ordering.
Factory/review transcripts: `/tmp/mch-r3-code-spec.log` and
`/tmp/mch-r3-review-loop.log`.

## Scope and ownership

- `cmd/server/main.go` retains unchanged config/flag/signal/process-exit logic.
- `cmd/server/lifecycle.go` owns `application`, `run`, `start`, and
  `newHTTPServer`: pool/listener acquisition, failed-start cleanup, HTTP serving,
  10s shutdown deadline, exactly-once cleanup, 10s header/30s request read limits.
- `cmd/server/router.go` owns private `newRouter(pool, allowedOrigins, logger)`,
  unchanged middleware/module wiring, and unchanged `jsonErrorHandler`.
  Composition neither connects/pings nor closes the caller's pool. Startup
  still acquires pool then listener before router setup, sets the normal stdout
  timestamped global logger, and closes both resources on CORS failure.
- No public API, dependency, SQL, transaction, config precedence, route, module
  constructor, APIHydra fixture/suite, or process contract changed. Existing
  validation causes and URL parser repairs are untouched. No production test
  endpoint, lifecycle interface, mutable hook, flag wrapper or DB-outage campaign.

## Acceptance and behavior evidence

| Criterion | Meaningful verification |
| --- | --- |
| R3-01 ownership/lazy DB/failure precedence | `TestRouterDoesNotAcquireOrClosePool` uses a concrete lazy pool with BeforeConnect to prove zero construction attempts and retained caller ownership on success/failure; `TestStartFailures` retains parse cause and distinguishes occupied-listen before invalid CORS; `TestStartCORSFailureReleasesListener` rebinds the released port with an unreachable DSN. Pool-close branches remain direct in start and run under unit/race coverage. |
| R3-02 exact operations/dependencies | Existing `TestAPIConstructorRouteInventory` now invokes production `newRouter`, retaining its sole exact 34-pair list and unfiltered `Router().Routes()` comparison. All module constructors and dependency arguments are preserved; APIHydra exercises the full wiring. No framework-only entries needed filtering. |
| R3-03 middleware/HTTP/log/context | `TestRouterMiddlewareParity` checks allowed/denied and second configured origins, exact preflight method/header lists, empty204, Vary/Allow, no redirect on trailing slash, exact400/404/405 envelopes, URI/status logs. `TestRouterRecoveryPreservesContextAndLogs` verifies request-context identity, recovered safe500, CORS and internal/request logging. Unit-only routes are separate router instances from the inventory. |
| R3-04 central errors/write failures | Existing `TestInstalledJSONErrorContracts` now uses production composition for 400/404/409/500 and cause-wrapped envelopes. `TestJSONErrorWriteFailure` rejects an ordinary write and checks the single safe attempted body, original internal diagnostic, and exact centralized `write error response: broken response transport` log. Global logger captures restore state and tests remain serial. |
| R3-05 lifecycle/limits | Existing `TestLifecycle`, `TestStartCancelRealServer`, `TestHTTPServerPreservesReadTimeout`; deadline assertion strengthened to10s. `TestLifecycleFailedShutdownClosesBeforeWaiting` requires close to unblock serve, preserves shutdown cause and exactly-once order. |
| R3-06 reliable route audit | Inspected unchanged `scripts/contracts_test.py` before extraction: every registration remains in `internal/*/api.go` using its recognized a.g/e receivers. Source-to-ledger audit still passes alongside the new actual-production-constructor inventory. No count-only weakening or route filter. |
| R3-07 scope/delivery | Final required commands, source hash comparison, tracked/untracked scope review and diff whitespace check; this checkpoint, top plan status and backend implementation log record results. Native review against pinned dev 4c25d70 passed without findings. |

Baseline `go test -short -count=1 ./cmd/server` passed. Before production edits,
a temporary real-startup characterization test exercised allowed/denied origins,
preflights, trailing slash and router errors. Log:
`/tmp/mch-r3-characterization.log`. It established denied simple requests still
reach validation400 without permission headers; denied preflight204 has no
permission headers; allowed preflight204 advertises the exact existing lists;
trailing slash removes without redirect and logs the normalized URI. Temporary
test was replaced by the permanent handler assertions above.

## Commands and outcomes

Commands are from root except explicit backend working-directory commands.
Logs are `/tmp/mch-r3-{check,check-all,coverage,deps-audit,api-test,legacy-api-test}.log`.

| Command actually run | Exit/result |
| --- | --- |
| Backend: `go test -short -count=1 ./cmd/server` | Baseline0; intermediate1 for an incorrect test-only Echo.Close assumption, removed; final0 |
| Root: `go test -short -count=1 -run TestCharacterizeStartupHTTP -v ./cmd/server` | 1, no root Go module; corrected backend invocation0 |
| Backend: `go test -short -count=1 -run TestCharacterizeStartupHTTP -v ./cmd/server` | 0; pre-extraction HTTP evidence retained in temporary log |
| `golangci-lint fmt --no-config --enable gofumpt --enable goimports` with the six changed server Go files explicitly listed | 0; touched files only, diff inspected |
| `make -C backend check` | 2: existing11 lint findings; default cache also emitted read-only warnings |
| `GOLANGCI_LINT_CACHE=/tmp/mch-r3-lint-cache make -k -C backend check` | 2: only existing11 lint findings; formatting/vet/race pass, all45 Python tooling tests and Go suite-validator tests pass |
| `make -C backend coverage` | 0;924/939, strict >95% passes |
| `make -C backend deps-audit` | 0; no vulnerabilities |
| `make -C backend api-test` | 0; complete unchanged five APIHydra suites (408 requests/34 operations),851/939, >=90% passes |
| `make -C backend legacy-api-test` | 0; change/epic/health/project/testcase HTTP/SQL packages pass; shared has no tests |
| `scripts/codex-review-loop.pl agent/specs/012-backend-startup-boundaries.md --base origin/dev` | 0; native pass 1 clean, no fixes; reviewer repeated unit/race checks |
| `git diff --check` | 0 |

Lint debt is unchanged: three unchecked Body.Close calls plus six missing
comments in api-tests/shared/client.go and missing health/markdown package
comments (3 errcheck,8 revive). No new lint failure or suppression. No final
application scenario failed, was skipped or blocked. No benchmark/toolchain
compatibility change required benchmarks or Docker checks.

Fresh profiles, reports, counters and provenance are under `.coverage/unit`
and `.coverage/api`; `.coverage/legacy` stays separate. Both measured Go source
inventories match the final production and test file hashes, including new
untracked files. Each integration runner owned/stopped/removed its disposable
PostgreSQL cluster and server. No shared DB reset, stale counter reuse,
production exclusion, or mixing of unit/legacy counters into APIHydra.

## Fresh statement counts and remaining gaps

| Production package | Unit covered/total | APIHydra covered/total |
| --- | --- | --- |
| cmd/server | 77/92 | 76/92 |
| internal/change | 398/398 | 378/398 |
| internal/domain | 0/0 | 0/0 |
| internal/epic | 115/115 | 107/115 |
| internal/error | 48/48 | 36/48 |
| internal/health | 21/21 | 16/21 |
| internal/project | 117/117 | 110/117 |
| internal/testcase | 108/108 | 99/108 |
| pkg/config | 30/30 | 21/30 |
| pkg/markdown | 10/10 | 8/10 |
| **Aggregate** | **924/939 (98.4026%)** | **851/939 (90.6283%)** |

Extraction adds four executable composition statements; no denominator omission.
All15 uncovered unit statements are process-only main.go lines15–34: config,
flags, signals, lifecycle invocation and error exit. Router/lifecycle statements
are fully unit-covered. No main wrapper was added for percentages.
APIHydra gaps total88: server16, change20, epic8, error12, health5, project7,
testcase9, config9, markdown2. Server gaps are startup/serve/shutdown failures,
failed resource acquisition/CORS, error exit and error/write diagnostics.
Other packages retain SQL query/scan/iteration/Exec errors, entropy, concurrent
FK/parent failures, degraded health, config defaults/failures, Markdown errors
and central nil-helper cases. Domain has no executable statements. Both gates
use actual counts, not rounded package averages.

## Factory handoff

Implementation and native review are complete; documentation publication and
the authorized merge-to-dev are the supervisor's final actions. No production
source changed after the recorded measurements. R4 next replaces global
configuration lookup with independent returned instances while preserving the
fixed path, precedence and panic behavior. R5 then repairs conventions/lint
debt; R6 reassesses compact code. Preserve the ledger's R1/R2/011 evidence.
No stage or production promotion.
