# CLI rebuild checkpoint

## P4 review fixes 03 (2026-09-28)

The P1 finding is fixed: clearing a change update prompt retains its field, so
replacement PR URLs and prerequisite IDs cannot fall through to spec insertion.
Escape or a second Ctrl+C on the cleared prompt cancels the editor. The prompt
hint now describes clearing and cancellation accurately. No HTTP contract changed.
The [ledger](cli-contracts.md) maps unit and complete-program regressions. Existing
uncommitted review-fixes02 work was preserved; statistics count only this pass's
diff against its starting workspace. No commit or push was performed.

Final verification used revision `55c18dc093f633970cd2ae5b7a527cb4ba42a134` plus
the uncommitted workspace. Formatting used pinned golangci-lint gofumpt/goimports
on the five touched Go files. The new unit regression initially failed for both
fields; final results below include the fix. The initial program assertion used
a status phrase that wraps at 80 columns; its corrected visible marker passes.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/app ./internal/changes -count=1` (inside `cli/`) | 0 | Unit regressions and retained app/feature tests pass. |
| `go test ./integration -run '^TestCLIProgramChangeCRUDAndPartialSuccess$' -count=1 -timeout=60s` (inside `cli/`) | 0 | Exact field updates after clearing, no unexpected document route, and partial-success recovery pass. |
| `make -C cli check` | 2 | Untouched formatting baseline: cmd/mch/main.go, app/clipboard.go, testcases/api.go, testcases/model.go. Final revive output reports package comments in cmd/mch/main.go, styles/styles.go, ui/layout.go. Vet, race, architecture and tooling pass. |
| `make -C cli coverage` | 2 | Complete unit campaign: **3028/3530 (85.7790%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete program/startup suite passes. |
| `make -C cli integration-coverage` | 2 | Complete program and real PTY campaign: **2550/3530 (72.2380%)**; strict >90% gate fails. |
| `python3 -B -m unittest discover -s scripts -p 'documentation_test.py' -v` (inside `cli/`) | 0 | Three documentation/link/inventory checks pass. |
| `git diff --check` | 0 | Workspace diff passes whitespace checks. |

Both status.json files report complete measurements; no required scenario failed
or skipped. Relative to review fixes02, app is unit1516/1770 and terminal1213/1770;
other package counts are unchanged. Uncovered statements: unit502, terminal980.
Package/function gaps and raw profiles are in `cli/.coverage/{unit,integration}`;
command logs and actual exits are in `cli/.coverage/spec023/review-fixes03`.
Both pre-documentation input manifests (sorted JSON inputs_sha256) hash to
`66921874a748c42926f55dec83d93e463c2b938c08f58f54cd93a88a4d01117b`.
Covered child SHA256:
`3a67985d04b7c627defbb3775d5760df73433ca43a38745ab01e374fd05edcb8`.
Provenance records source hashes, revision/diff, commands and tool versions.
Subsequent checkpoint, ledger, plan and implementation-log edits are documentation.

Final rebuild validation remains incomplete due to the existing static baseline
and coverage shortfalls. Caller owns further review, publication and the authorized
intermediate dev merge; P5 remains next. No live backend or database was used.

## P4 review fixes 02 (2026-09-28)

Both findings are fixed. Brief editor completion returns to the creation form
without submitting; explicit title and optional UUID remain editable before Enter
saves. Exact raw brief bytes survive confirmation and editor reopening. Identity
rows join the scrollable detail body whenever pinning them would leave no body
space. Selection/copy indexes remain stable, and normal-height panes retain pins.
The [ledger](cli-contracts.md) names regression and complete-program assertions.
No HTTP contract, dependency, manifest, backend or live service changed.

Final verification used revision `55c18dc093f633970cd2ae5b7a527cb4ba42a134` plus
its uncommitted test diff. Workspace snapshots during this pass contain early tests
and production changes; this agent invoked no Git commit or push. The complete
findings-fix statistics use pre-pass `0caa36e` so those snapshots are included.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/app ./internal/changes -count=1` (inside `cli/`) | 0 | Creation confirmation, short viewport, resize, wrapped feedback and existing unit regressions pass. |
| `go test ./integration -run '^TestCLIProgramChangeCRUDAndPartialSuccess$' -count=1 -timeout=60s` (inside `cli/`) | 0 | Keyboard creation/updates and partial-success assertions pass; final full campaigns include the subsequently retained plain-brief variant. |
| `make -C cli check` | 2 | Only untouched baseline formatting in cmd/mch/main.go, app/clipboard.go, testcases/api.go and testcases/model.go; revive package comments in help/commands.go, styles/styles.go and ui/layout.go. Vet, race, architecture and tooling pass. |
| `make -C cli coverage` | 2 | Complete unit campaign: **3015/3529 (85.4350%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete program/startup tests pass. |
| `make -C cli integration-coverage` | 2 | Complete program and real PTY campaign: **2546/3529 (72.1451%)**; strict >90% gate fails. |
| `python3 -B -m unittest discover -s scripts -p 'documentation_test.py' -v` (inside `cli/`) | 0 | Three documentation/link/inventory checks pass. |
| `git diff 0caa36e --check` | 0 | Complete findings-fix diff passes whitespace checks. |

Both status.json files report complete measurements; no required scenario failed
or skipped. Relative to the historical package table, app is now unit1503/1769,
terminal1209/1769, and changes is unit727/865, terminal662/865. All other package
counts are unchanged. Remaining uncovered statements: unit514, terminal983.
Exact package/function gaps and raw profiles remain under
`cli/.coverage/{unit,integration}`; command logs and actual Make exits are under
`cli/.coverage/spec023/review-fixes02`. Both pre-documentation input manifest
SHA256 values (sorted JSON inputs_sha256) are
`f915cfc1768969d18fde26d6168933e72b89aed4b88f812615dc495617b11a4b`.
Covered child SHA256:
`be29fe32c57cdaa7d6aa10f1c18059bc0cf8dcb8348821fc60fdf13c7da05ca4`.
Provenance includes source hashes, revision/diff, commands and tool versions.
Later checkpoint, ledger, plan and implementation-log changes are documentation.
Formatting used pinned golangci-lint gofumpt/goimports on touched Go files only.

Final rebuild verification remains incomplete because the known static baseline
and strict coverage targets remain unmet. The caller owns further review,
publication and the authorized intermediate dev merge; P5 remains next.

## P4 review fixes 01 (2026-09-28)

Both reported findings are fixed against `c30dc5bd916679f295b8e5c3946831b50a513704`
plus the uncommitted review diff. Leaving or reinitializing change creation clears
the active title/UUID subfield. Mutation feedback limits the quoted field value
to 42 terminal cells after escaping control characters; complete saved values
remain in feature detail state, with committed-step and read-retry feedback intact.
No HTTP contract, dependency, integration manifest or existing assertion changed.
The new regression tests first reproduced stale subfields and 54/56-line views
on an 80×24 terminal, then passed after the fixes.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/app ./internal/changes -count=1` (inside `cli/`) | 0 | Existing tests and both review regressions pass. |
| `make -C cli check` | 2 | Existing formatting failures in cmd/mch/main.go, app/clipboard.go, testcases/api.go and testcases/model.go; revive package comments in help/commands.go, testcases/api.go and ui/layout.go. All untouched. Vet, race, architecture and tooling pass. |
| `make -C cli coverage` | 2 | Complete unit campaign: **2994/3512 (85.2506%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete program/startup scenarios pass. |
| `make -C cli integration-coverage` | 2 | Complete program and real PTY campaign: **2523/3512 (71.8394%)**; strict >90% gate fails. |

Both status.json files report complete measurements, with no failed or skipped
required scenarios. Fresh package totals differ from the historical table below
only for app (unit1498/1764, terminal1204/1764) and changes (unit711/853,
terminal644/853). Remaining uncovered statements: unit518, terminal989.
Exact package/function gaps and raw diagnostics remain under
`cli/.coverage/{unit,integration}`; command logs and exit files are under
`cli/.coverage/spec023/review-fixes01`.
Both pre-documentation input manifest SHA256 values (sorted JSON inputs_sha256):
`e73534d12a192fefe44b56dbe2900c1debf86f3ca1c5d1de5555df4b4b8ef8bb`.
Covered child binary SHA256:
`866a2a522ac40c1efac0676a6cc8d954f35e7c5784b43783e8da8bc8891226e8`.
Campaign provenance records all source hashes and tool versions. Subsequent
checkpoint, ledger, plan and implementation-log edits are documentation only.
Full rebuild verification remains incomplete due to the known static baseline
and coverage shortfalls. Caller owns further review, commits, pushes and dev merge;
this fix pass performs no Git publication.

## Historical implementation handoff before review

[Plan](../../agent/cli-rebuild-plan.md), [architecture](../../docs/cli-architecture.md),
[ledger](cli-contracts.md), [P4 specification](../../agent/specs/023-cli-change-management.md).
P0 merged e46bf64; P1 6a3d5bb; P2 1658624; P3 actual dev
`80c67871aaa29ed92a967aa70b449b29bd88c406` after native review pass04 clean.
P4 branch `change/023-cli-change-management` started with spec commit
`79d1305f44f3d9130a4d7b86addbb88dff8ccd84`. After quota interruption, the helper
resumed from preserved snapshot `3ad7aa7c01718822ec4cfa3fb1b43fbf0fbcf835`.
Acceptance inspection, fresh validation and the missing implementation-log block
are finished. This resume changes handoff documentation only; production code,
tests, dependencies and the scenario manifest match the tested snapshot.
Native review and dev merge remain pending. Final rebuild acceptance remains
incomplete because the strict coverage targets and static baseline are unmet.

## Implemented behavior awaiting native review

All eleven typed change routes, exact nullable/identity/count/time fields,
brief terminology, explicit title/brief/optional UUID creation, field updates,
nullable associations, false open, empty types and delete are implemented.
Changes own state/sequencing/stale-result checks and committed outcomes; separate
ChangeView projects ordinary document and testcase reads. Supporting documents
current/insert enables retained editors and configured custom types. Full document
history/all owners remains P6, testcase mutation migration/ownership remains P5.
The ledger maps every acceptance ID and retained/revised assertion. P1–P3 editor,
config draining, partial-write and viewport regressions remain in the suites.

Acceptance inspection checked P4-01/02 against current backend change/domain
contracts, P4-03 against forms and all eleven keyboard-driven route effects,
P4-04/05 against feature sequencing, scoped outcomes, cancellation, stale reads,
viewports and retained raw editor regressions, and P4-06/07 against document
current/insert contracts, separate testcase reads and the manifest inventory.
No additional production correction was identified in this implementation pass;
this inspection does not replace the caller's native review.

## Historical resume validation (2026-09-28)

Both campaigns ran on snapshot `3ad7aa7` before these final documentation edits.
All production/test/tooling inputs in both provenance manifests match current files.
Unit **2990/3508 (85.2338%)**, terminal **2520/3508 (71.8358%)**;
strict gates fail honestly. Both status.json files report complete measurements.
No required test scenario failed or skipped. Testless packages remain in the
denominator. These totals do not establish final rebuild completion or review approval.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `make -C cli check` | 2 | Only four untouched formatting files and package-comment lint; vet, race, architecture and tooling pass. |
| `make -C cli coverage` | 2 | Complete unit campaign; strict numerical shortfall. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete program/startup scenarios pass. |
| `make -C cli integration-coverage` | 2 | Complete program and real PTY campaign; strict numerical shortfall. |
| `go test -run '^$' -bench '^BenchmarkChangeLongRendering$' -benchmem -benchtime=100ms ./internal/changes` (inside `cli/`) | 0 | 5k characters ~0.70ms; 50k ~4.98ms; 100k ~9.64ms. |
| `python3 -B -m unittest discover -s scripts -p 'documentation_test.py' -v` (inside `cli/`) | 0 | Final documentation links, Make targets and route/assertion inventory. |
| `git diff --check 79d1305` | 0 | Complete implementation diff, including final handoff documentation. |

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cli/cmd/mch` | 0/3 | 1/3 |
| `cli/internal/app` | 1495/1761 | 1202/1761 |
| `cli/internal/changes` | 710/852 | 643/852 |
| `cli/internal/documents` | 6/6 | 4/6 |
| `cli/internal/dto` | 0/0 | 0/0 |
| `cli/internal/epics` | 214/214 | 187/214 |
| `cli/internal/help` | 6/6 | 6/6 |
| `cli/internal/navigation` | 25/40 | 17/40 |
| `cli/internal/projects` | 211/221 | 183/221 |
| `cli/internal/styles` | 0/0 | 0/0 |
| `cli/internal/testcases` | 5/6 | 5/6 |
| `cli/internal/ui` | 8/8 | 8/8 |
| `cli/pkg/client` | 310/391 | 264/391 |

Unit leaves518 statements uncovered; terminal988. Exact gaps, raw profiles,
commands and source hashes are under cli/.coverage/{unit,integration}; command
logs/exits for this resume are under cli/.coverage/spec023/resume. Earlier failed
PTY campaign retained under spec023/integration-incomplete; its row expectation
was adapted to measured seven-row layout, then the complete campaign reran.
Remaining gaps include process exit/error paths, editor/config failure branches,
legacy testcase paths (P5), change validation/rendering branches, navigation and
malformed transport cases. Each campaign's uncovered.txt and functions.log contain
the exact statement and function inventory; no package is excluded for its gaps.
Both pre-handoff input manifest SHA256 (sorted JSON inputs_sha256):
`902f030317ee1fe0401de4035aae7751b66ad5bdd97a07fa63b60bd716647abe`.
Production-only input digest (compact sorted JSON):
`5411b8b3ca1118c0f6d4e2cac906f9b163d98951230e21f96438a4d7689797b3`.
Covered child binary SHA256:
`e7bb16805eb90e5e35f20b4a740ea8451bdbc60d68d3ad97d4d7a2f2c0440c97`.
Go1.26.8-X:nodwarf5, golangci-lint2.13.1, govulncheck1.7.0, socat1.8.1.1.
The ledger, checkpoint and implementation log are included in provenance input
hashes; their subsequent documentation-only edits do not change measured source.

Baseline format owners: cmd/mch/main.go and app/clipboard.go (P10/cleanup),
testcases/api.go and model.go (P5). Fresh revive sampled styles/styles.go,
testcases/api.go and ui/layout.go; package comments remain owned by P5/P10.
All reported static files are untouched by P4. No new static/test failure remains
in the recorded run; native review is pending.
No backend/database/frontend/cli-proto/dependencies/.mch/root helper changes.

## Exact continuation

The caller owns the implementation commit/push, native review through
scripts/codex-review-loop.pl SPEC --base origin/dev, review fixes and revalidation,
then scripts/merge-to-dev.pl and verification of the actual dev SHA.
The implementation log counts the full diff from79d1305, including the preserved
snapshot and this resume, excluding the log itself. The implementation child
performed no Git publication or nested factory invocation.
Continue P5 testcases, P6 documents, P7 configs/health, P8 clarification,
P9 spec review/fix, P10 acceptance and evidence-based R1–R6 cleanup. Never predict
squash hashes or claim final targets met. User authorization remains active.
