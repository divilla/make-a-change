# Backend validation-cause checkpoint — 011 reviewed

2026-09-28; `change/011-backend-validation-causes`, specification HEAD
`83ae94e5fe9d968a883f900907a22f0f0162ce6e` over fetched origin/dev
`16bbf0b305cc263da1bf4f9b0fd0bda04d3fca4d` (merged R1). Initial tree was clean.
This is the bounded error-contract repair, not R2 duplicate cleanup. R2 remains
no actionable findings. Factory implementation `db415e2` is published; native
review against pinned `16bbf0b` passed with no actionable findings or fixes.
Overall verification remains incomplete only for the known R5 lint debt. Both
coverage gates pass. The supervisor is publishing this checkpoint before the
authorized squash merge to dev; next is R3 startup boundaries. No transient
retry, next-pass implementation or promotion occurred. Transcripts:
`/tmp/mch-vc-code-spec.log` and `/tmp/mch-vc-review-loop.log`.

## Contract and acceptance evidence

The [ledger](backend-contracts.md#011-validation-cause-repair) records the one
spec-authorized shared helper and preserved contracts. Central Validation uses
existing semantic/cause wrapping, API handlers preserve real validate.Errors,
and change URL validation preserves real parser errors. No public HTTP envelope,
accepted input, validation order, collaborator call count, route, DTO, SQL,
transaction, dependency or protected reference changed. All30 tag-validation
handlers were migrated (project5, epic5, testcase5, change15). Business-only
validation returns the original sentinel; bind wrapping/messages stay intact.
All writes remain under backend except the authorized plan top status.

| Criterion | Named test evidence |
| --- | --- |
| VC-01 central helper, direct/wrapped causes, nil behavior, ordinary error and cause identity | internal/error TestValidationCauses: actual malformed escape and syntax parser errors, direct/wrapped semantic and cause, each nil permutation, exact cause pointer, no Echo HTTP error |
| VC-02 all API validator paths, module sentinel, field/rule and safe envelope | TestAPIValidationCauses in project, epic, testcase and change: actual registered handler requests for every migrated path, errors captured before HTTP writing, errors.Is sentinel/errors.As validate.Errors, JSON field/min or required rule; no repository/config calls |
| VC-03 parser cause, business validation, ordering, accepted forms and calls | change TestUpdatePRURLValidationCauses: real *url.Error/EscapeError, syntax failure, ID before parser, blank/empty/missing host/relative/unsupported scheme, uppercase/trimmed/userinfo/query/fragment acceptance, exactly one valid write, original repository failure; retained TestServiceRejectsInvalidDirectInput and TestServiceCollaboratorFailuresAndContext |
| VC-04 preserved bind/generic envelopes, existing cause evidence and unknown masking | Four TestAPIValidationCauses matrices assert exact generic400/bind-operation JSON; strengthened TestDocumentAPIShapeAndExplicitBooleans preserves real validator cause and adds sentinel/required-rule proof; retained TestChangeAPIContracts, TestChangeHandlerReturnCauses, TestProjectHandlerErrorContracts, TestEpicHandlerErrorContracts, TestTestCaseHandlerErrorContracts and central TestHTTPContractsAndCauses cover unknown masking and cause retention |
| VC-05 parity, measurements and handoff | All commands below; retained TestAPIConstructorRouteInventory proves34 operations; unchanged APIHydra suite includes malformed URL `%` and safe400; full retained legacy Go HTTP/SQL suite passes; ledger, plan status and implementation log updated |

No extra APIHydra scenario was needed. Suites remain backend/apih-tests with
retries:-1; installed apih/manual revision remains
c3947513e2a4b948ce732dec60e090a5e83be744. All408 requests/34 operations are retained.
APIHydra cannot inspect Go causes; those assertions belong to the unit tests.

## Commands and results

Logs: `/tmp/mch-vc-*.log`. Fresh profiles/reports/results/provenance are under
backend/.coverage/unit and backend/.coverage/api; legacy evidence is separately
under backend/.coverage/legacy. The runners owned, stopped and removed their
private PostgreSQL clusters and servers. No external database was reset and no
unit/legacy profile was merged into APIHydra. Final production hashes match
both measured source inventories; no production code changed after measurements.

| Command actually run | Exit/result |
| --- | --- |
| `git fetch origin dev` | Initial sandbox exit255 (read-only FETCH_HEAD); approved retry exit0, required base confirmed |
| Backend: `go test -short -count=1 ./internal/error ./internal/project ./internal/epic ./internal/testcase ./internal/change` | Baseline0; intermediate test compile/JSON-field expectation failures1 corrected; final0 |
| Same focused Go command accidentally invoked from root | 1, no root Go module; corrected backend invocation passes |
| Backend: `golangci-lint fmt --no-config --enable gofumpt --enable goimports` with the13 changed Go files explicitly listed | 0; no unrelated formatting edits; final helper-test-only formatting rerun0 |
| `GOLANGCI_LINT_CACHE=/tmp/mch-vc-lint-cache make -C backend check` | 2: baseline11 lint plus initial test-only SA1007 warning about deliberately malformed constant URL; warning fixed using real parser input table |
| `GOLANGCI_LINT_CACHE=/tmp/mch-vc-lint-cache make -k -C backend check` final | 2: only baseline11 lint findings; formatting/vet/race pass, all45 Python tooling tests and Go suite-validator tests pass |
| `make -C backend coverage` (initial and final after test fix) | 0 each; fresh917/935, strict >95% passes |
| `make -C backend deps-audit` | 0, no vulnerabilities found |
| `make -C backend api-test` | 0, complete408 requests/34 operations;848/935, >=90% passes |
| `make -C backend legacy-api-test` | 0, complete retained change/epic/health/project/testcase HTTP/SQL packages; shared has no tests |
| `python3 -B -m unittest discover -s backend/scripts -p 'contracts_test.py' -v` after documentation updates | 0, both contract checks pass |
| `git diff --check` | 0 |

Baseline lint reports three unchecked Body.Close calls and six comment findings
in api-tests/shared/client.go, plus missing health and markdown package comments
(3 errcheck,8 revive; report limiting can change displayed package comments).
All are unchanged R5 debt. No new lint issue remains or was suppressed.
No final application scenario failed, was skipped or blocked. Benchmarks,
Docker compatibility and additional outage scenarios are outside this repair.

## Fresh statement counts and gaps

| Production package | Unit covered/total | APIHydra covered/total |
| --- | --- | --- |
| cmd/server | 70/88 | 73/88 |
| internal/change | 398/398 | 378/398 |
| internal/domain | 0/0 | 0/0 |
| internal/epic | 115/115 | 107/115 |
| internal/error | 48/48 | 36/48 |
| internal/health | 21/21 | 16/21 |
| internal/project | 117/117 | 110/117 |
| internal/testcase | 108/108 | 99/108 |
| pkg/config | 30/30 | 21/30 |
| pkg/markdown | 10/10 | 8/10 |
| **Aggregate** | **917/935 (98.0749%)** | **848/935 (90.6952%)** |

The denominator adds seven real statements: central helper5 and URL validation2.
Nothing is omitted or artificially executed for coverage. Unit gaps remain18
server main/start/error-handler statements. API gaps total87: server15,
change20, epic8, error12, health5, project7, testcase9, config9, markdown2.
They cover startup/config/shutdown/error paths, SQL query/scan/iteration/Exec
failures, entropy, concurrent FK/parent failures, degraded health, Markdown
failures and central nil helper contracts (covered directly by units). Domain
has no executable statements. Actual counts, not rounded percentages, pass.

Factory next: commit/push and native review --base origin/dev; supervisor owns
the authorized merge-to-dev. Next category after review/merge is012 startup
boundaries (R3), then R4 config, R5 conventions and R6 reassessment. No stage or
production promotion. Preserve R1 removal evidence and R2 no-action rationale
in the ledger; this checkpoint replaces the previous iteration transcript.
