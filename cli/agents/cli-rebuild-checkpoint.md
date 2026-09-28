# CLI rebuild checkpoint

## Current handoff: P2 reviewed, ready for dev merge

2026-09-28. [Plan](../../agent/cli-rebuild-plan.md),
[architecture](../../docs/cli-architecture.md),
[contract and assertion ledger](cli-contracts.md),
[P2 specification](../../agent/specs/021-cli-projects-typed-transport.md).
P0 merged as `e46bf64`; P1 merged as actual dev `6a3d5bb`.
P2 implementation `61d517b` and review fixes `c76b302`, `8999a43`, `397b142`,
`92c192d` are published on `change/021-cli-projects-typed-transport`.
Native review pass05 against pinned P1 dev found no actionable regressions.
Review independently passed app/project/client/integration tests and diff checks.
Final production revision is `92c192d`; subsequent changes are documentation only.
No future dev squash SHA is claimed. No stage/production promotion.

## Established contracts

- All six project POST operations use current typed wire shapes. Details/list
  carry integer IDs/counts/last_ref, time values and config slug. Create returns
  only the 201 ID; update/delete accept empty 204. Config exposes its slug plus
  project_docs, epic_docs, change_docs, change_phases, change_colors, change_types.
  No invented request fields, envelopes, `/get`, global options or alternate
  project routes remain. HTTP accepts caller context and injected clients, has a
  finite 15-second bound, rejects malformed/missing/wrong-shaped data, and keeps
  response status and error causes including close errors. Redirects are rejected.
- `internal/projects` owns validation, drafts, operation identities, sequencing
  and results. List/details/create/edit/delete/config are discoverable. Writes
  execute once; a failed follow-up read retains created ID or saved/deleted
  outcome. `/retry` reads only. Detail/config views expose all returned fields.
- Root shell owns selection, routing, shared prompt/editor mechanics and adapter
  construction. Catalogs clear and reload for each selected project, respect
  backend ordering and colors, and have no global/default fallback. Changes
  render missing phase colors with terminal styling rather than invented colors.
  Project/entity and operation generations reject obsolete results; superseded
  feature reads cancel. Program shutdown cancels active project HTTP requests.
- P1 retained behavior remains tested: exact document/editor bytes, raw failed
  drafts and literal slash data, unchanged-editor no-op, cleanup/redraw, committed
  change IDs, and asynchronous atomic local selection saves with permissions,
  honest save failures, queued-save serialization and orderly quit draining.
- Current route/assertion migrations and every acceptance ID are mapped in the
  ledger. Existing program/PTY harnesses are extended; direct adapter tests remain
  unit-only. No backend, DB, frontend, cli-proto, dependencies, retained `.mch/`
  resources or root helper changed. No live backend/database/agent was used.

## P2 review fixes 04 verification

The deletion-persistence finding is fixed. A failed local write of `project_id: 0`
now retains the committed deletion status, list-refresh error and read-only retry
guidance. It separately reports that selection was cleared in memory but could
not be saved. No HTTP contract changed.

`TestP204DeleteSelectedProjectPreservesOutcomeOnConfigFailure` covers successful
and failed list refreshes, direct and queued clears, quit cancellation, unchanged
local data and exactly-once deletion across a read retry. All four cases reproduced
the original failure (Go test exit1) and pass after the fix. The existing
manifest-selected `TestCLIProgramProjectCRUDAndPartialSuccess` adds both local
save-failure cases through keyboard navigation, a real atomic-replacement failure,
visible deletion/refresh/persistence diagnostics and a list-only retry. No scenario
or assertion was removed. An initial program assertion assumed unwrapped terminal
text; it was corrected to match the displayed fragments before final verification.

All required test scenarios finish without skips. Independent unit measurement:
**2365/2852 (82.9243%)**; terminal measurement:
**1655/2852 (58.0295%)**. Strict >95% and >90% gates fail honestly.
Numerical shortfalls alone may continue through the authorized intermediate dev
merge; overall rebuild completion remains unmet.

| Final command | Exit | Evidence |
| --- | ---: | --- |
| Scoped `golangci-lint fmt --no-config --enable gofumpt --enable goimports` | 0 | Touched app production, unit and program Go files formatted; diff inspected. |
| `cd cli && go test ./internal/app ./internal/projects -count=1` | 0 | Targeted unit and retained regressions. |
| `cd cli && go test ./integration -run '^TestCLIProgramProjectCRUDAndPartialSuccess$' -count=1` | 0 | New complete-program cases and retained CRUD/partial success. |
| `make -C cli check` | 2 | Only baseline format/lint below; vet, unit race, architecture, 36 Python tooling tests and Go fixtures pass. |
| `make -C cli coverage` | 2 | Complete measurement, strict unit threshold fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete-program and standalone startup tests pass. |
| `make -C cli integration-coverage` | 2 | Complete program/startup/real PTY measurement, strict terminal threshold fails. |
| `git diff --check` and scoped documentation tests | 0 | Whitespace, local links, commands and manifest/ledger mapping checked. |

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cli/cmd/mch` | 0/3 | 1/3 |
| `cli/internal/app` | 1364/1624 | 869/1624 |
| `cli/internal/changes` | 477/599 | 380/599 |
| `cli/internal/dto` | 0/0 | 0/0 |
| `cli/internal/epics` | 4/5 | 4/5 |
| `cli/internal/help` | 6/6 | 6/6 |
| `cli/internal/navigation` | 23/40 | 12/40 |
| `cli/internal/projects` | 211/221 | 183/221 |
| `cli/internal/styles` | 0/0 | 0/0 |
| `cli/internal/testcases` | 5/6 | 5/6 |
| `cli/internal/ui` | 12/13 | 8/13 |
| `cli/pkg/client` | 263/335 | 187/335 |

Unit leaves **487** statements uncovered; terminal leaves **1197**.
Largest gaps remain app legacy operations/errors, changes parsing/rendering,
and unported client decoding paths; cmd remains 0/3 unit. Projects unit coverage
is 211/221 (95.4751%). All production packages remain in both denominators,
including packages without tests and zero-statement packages.

Raw review logs: `cli/.coverage/review-021-04/`. Both final
`cli/.coverage/{unit,integration}/status.json` record complete measurements with
numerical gate failure. Independent raw
profiles, uncovered blocks/functions, exact package counts, command journals,
source manifests and covered child binary hashes are retained there.

Tested Go1.26.8-X:nodwarf5, golangci-lint2.13.1, govulncheck1.7.0,
socat1.8.1.1. Both final source-manifest digests (SHA256 of sorted JSON
`inputs_sha256`) are
`32f0d31a981338d132cf56ac76e27373b5412516992229a2866a124f0a86a1b6`.
Covered child binary SHA256:
`bcbed74ac51649afe486355a5fc82ebda9a17e8bad3f4cf30a84856847476b88`.
Subsequent checkpoint/ledger/plan/log edits are documentation only. Docker and benchmarks
were not rerun because toolchain/dependencies/performance contracts did not change.
P0's Docker mounted-file permission issue remains historical environment evidence.

## Baseline owners and next action

Six untouched formatting files remain: cmd/mch/main.go and app/clipboard.go
(final cleanup/P10); epics/api.go and model.go (P3); testcases/api.go and model.go
(P5). Final revive samples package comments in help/commands.go, styles/styles.go and
ui/layout.go (cleanup/P10); the sampled list can vary.
P2 repaired all touched project/client/app/DTO baselines and the downstream
changes model/view/test formatting needed for the catalog-color correction.
No new static or test failure remains.

Next: caller publishes this documentation checkpoint, merges through the dev
helper, verifies the actual dev SHA, then starts
P3 epics. Remaining route migrations are explicitly owned in the ledger: P3 epics,
P4 changes, P5 testcases, P6 documents, P7 configs/health, P8 clarification/process,
P9 spec review/fix, P10 combined acceptance, then evidence-based R1–R6 cleanup.
Keep final strict coverage gates and all retained P1 regressions through every pass.
