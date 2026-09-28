# Backend refactor checkpoint — 013 configuration isolation (R4 reviewed)

2026-09-28; `change/013-backend-config-isolation` over fresh merged R3
`b48efed0c41ddfc5aad8e5b04bf179f82348f864`. Factory implementation `7686129`
and contract/coverage report `23d5b43` are published. Native review against the
pinned R3 base passed on its first pass with no findings or fixes; the reviewer
also reran focused config/server tests with the race detector. The supervisor
is publishing this checkpoint before the authorized squash merge to dev.
Both strict coverage gates pass. Overall verification remains incomplete only
for displayed baseline lint diagnostics assigned to R5. No transient retry was
needed and no stage/production promotion occurred.

Next after merge: R5 `014-backend-local-patterns`, then R6 reassessment and the
separate016 final failure-integration verification. Factory/review logs:
`/tmp/mch-r4-code-spec.log` and `/tmp/mch-r4-review-loop.log`.

## Scope and contracts

- `pkg/config.New() *Config` constructs a local gookit loader and local Config.
  The package cfg singleton and Get are removed; main consumes New's result.
  Private default/environment helpers now use the same receiver naming as the
  accessors. The touched package also has an accurate package/New comment.
- Fixed `config/dev.yaml`, YAML driver, ParseEnv, mapstructure tags, defaults,
  nonempty untrimmed environment overrides, error-wrapped panics, accessor
  behavior and last-applied CLI flags remain. No global loader API/reset remains.
- Fresh checkout inspection confirmed R3's lifecycle/router separation and the
  pinned gookit v2.2.9 New/instance APIs before editing. Main changes only the
  New/Get call pair; flag parsing, signals, startup/cleanup and process exits
  retain their order. Application configuration stays separate from project DB
  configuration. No SQL, route, dependency, fixture, runner or public test seam
  changes. The New/Get contract change is explicitly authorized by spec013.
- Implementation/tests/checkpoint/log stay under backend; only the official
  plan's top status changes outside it, as authorized. The implementation log
  follows the specification's explicit backend path override.

## Acceptance evidence

| Criterion | Tests and checks |
| --- | --- |
| R4-01 independent ownership | `TestConfigurationIndependentLoads`: full file A, partial file B and empty file; distinct pointers even for equal values, earlier object preservation, owner mutation and future-load independence. Source search confirms only main consumes application New and no Get/global loader calls remain. |
| R4-02 file/decode parity | `TestConfigurationPanicCausesAndRecovery`: missing file, unreadable directory at the file path, malformed YAML, incompatible port after other fields; exact configuration context, nonnil unwrap, PathError/path and missing-file identity. Every case surrounds a failed load with successful loads and checks earlier values and default recovery. |
| R4-03 precedence | `TestConfigurationEnvironmentPrecedence`: file/default values survive unset and empty overrides, nonempty values remain untrimmed. `TestConfigurationEnvironmentInterpolation`: variable expansion, fallback, empty expansion followed by defaults and explicit override precedence. `TestMainConfigurationAndFlagOrdering`: ordinary executable covers omitted/empty flags, nonempty flags over both file/environment, config failure before help/invalid flags, and valid help/invalid flags. |
| R4-04 accessors | `TestConfigurationAccessors`: leading colon, empty/whitespace/double-colon ports; ordered/duplicate/trimmed origins, nil for empty/all blank, receiver mutations and fresh derived slices. Existing router CORS tests retain application behavior. |
| R4-05 ordinary API/error isolation | Config tests use only New with sequential TempDir/Chdir/Setenv and fixed-path files; no global Reset, private loader access, path parameter, injection hook or duplicate compatibility API. Recovery tests exercise partially decoded fields. |
| R4-06 concurrency | `TestConfigurationConcurrentIndependence`: 32 goroutines read one immutable fixture and stable environment, mutate only their returned Config, then collect all results before assertions/cleanup; distinct pointers, initial values, owner-specific mutations and later load asserted under race detector. |
| R4-07 startup/integration | Existing router/lifecycle tests remain unchanged and pass. New ordinary-executable tests assert exit codes, stderr diagnostics and empty stdout on early failures/help. APIHydra and legacy runners prove successful real startup using explicit port/private DB flags, readiness and all retained workflows. No subprocess counters enter unit/APIHydra measurements. |
| R4-08 verification/scope | Final formatting, Make targets, contract tooling, source-hash verification, tracked/untracked scope audit and whitespace check. This compact checkpoint and top plan status record results; factory retains review/commit/merge ownership. |

## Commands and results

Commands below run from root unless the backend working directory is specified.
Final logs: `/tmp/mch-r4-{check-final,check-all,coverage-final,deps-audit-final,api-test-final,legacy-api-test}.log`.

| Command actually run | Exit/result |
| --- | --- |
| Backend: `go test -short -count=1 ./pkg/config ./cmd/server` | 0 baseline; `/tmp/mch-r4-baseline.log` |
| Backend: `golangci-lint fmt --no-config --enable gofumpt --enable goimports pkg/config/config.go pkg/config/config_test.go cmd/server/main.go cmd/server/main_config_test.go` | 0; only touched Go files, diff inspected |
| Backend: `go test -short -count=1 -race ./pkg/config ./cmd/server` | Final 0; `/tmp/mch-r4-targeted-final.log` |
| `GOLANGCI_LINT_CACHE=/tmp/mch-r4-lint-cache make -C backend check` | 2, 11 displayed baseline lint diagnostics; formatting passes |
| `GOLANGCI_LINT_CACHE=/tmp/mch-r4-lint-cache make -k -C backend check` | 2, same displayed baseline diagnostics; format/vet/race pass, all 45 Python tooling tests and Go suite-validator tests pass |
| `make -C backend coverage` | 0; 925/939, strict >95% passes |
| `make -C backend deps-audit` | 0; no vulnerabilities |
| `make -C backend api-test` | 0; all five unchanged APIHydra suites, 408 requests/34 operations, 851/939; >=90% passes |
| `make -C backend legacy-api-test` | 0; change/epic/health/project/testcase HTTP/SQL packages pass, shared has no tests |
| `scripts/codex-review-loop.pl agent/specs/013-backend-config-isolation.md --base origin/dev` | 0; native pass1 clean, focused race tests repeated |
| `git diff --check` | 0 |

Intermediate attempts remain visible: the first targeted command was accidentally
run from root (exit 1, no root Go module); corrected backend invocation exposed
four overly specific test-only port diagnostic expectations (exit 1). Assertions
now match the selected port value rather than guessing OS error punctuation.
Initial `make -C backend check` exited 2 with receiver-naming findings introduced
by this pass and a pre-existing config package-comment finding; these are fixed.
It also emitted read-only default lint-cache warnings; final invocations use the
writable cache above. Initial coverage/audit/API targets passed, then all required
targets ran again after the final Go edit. No new failure remains, no failing
scenario was skipped, and no required external service/tool was unavailable.

The final normal check displays three unchecked Body.Close calls and six missing
comments in `api-tests/shared/client.go`, plus health/markdown package comments
(3 errcheck, 8 revive). These 11 displayed diagnostics match known baseline debt,
not the total unique inventory: golangci-lint limits repeated messages, so package
names can rotate. R5 must first enumerate the actual debt with its unchanged lint
selection plus `--max-same-issues=0 --max-issues-per-linter=0`, then repair it and
keep normal Make checks unchanged. No R4 failure is suppressed.
No benchmark/performance or toolchain/dependency compatibility change required
benchmarks or Docker checks.

## Fresh statement counts and gaps

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

All 14 uncovered unit statements are main.go process orchestration (lines 15–33).
The uninstrumented ordinary-executable assertions prove flag/error behavior but
intentionally add no counters. Config/router/lifecycle unit statements are fully
covered. Adding local Config initialization and removing the separate main Get
call leave the production denominator at 939; no code was excluded.

APIHydra's 88 uncovered statements: server16, change20, epic8, error12, health5,
project7, testcase9, config9, markdown2. Config gaps are load/decode panic branches,
three defaults, three explicit environment overrides and leading-colon Addr.
Server gaps remain acquisition/CORS/serve/shutdown errors, main error exit and
error/write diagnostics. Other packages retain SQL query/scan/iteration/Exec
errors, entropy, concurrent FK/parent failures, degraded health, Markdown errors
and central nil-helper cases. Domain has no executable statements. The separate
final failure-integration pass can address legitimate integration gaps; neither
coverage gate is below its required threshold in this pass.

Fresh profiles, structural denominators, package reports, counters and provenance
are under `.coverage/unit` and `.coverage/api`; `.coverage/legacy` is separate.
Both measured production inventories and recorded Go source/test hashes match
the final source, including the untracked new test. Both integration runners
owned, stopped and removed their disposable PostgreSQL cluster and server.
No shared DB reset, stale profiles, unit/legacy mixing or threshold suppression.
