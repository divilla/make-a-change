# CLI rebuild checkpoint

## P6 review fixes 03 (2026-09-28)

The document screen now limits every rendered row to the terminal width before
applying the viewport offset. Long catalogs, catalog errors and version headers
cannot wrap behind the renderer's row count. The detail header puts version
status and provenance first so they remain visible at narrow widths.
`TestP602DocumentHeadersStayWithinViewportWidth` reproduced a 249-cell `Types:`
row in a 24-column viewport before the fix, then passed after it.

| Command on revision `d4491fa25d57c5966eade8b16447b47bc36b9421` plus review diff | Exit | Evidence |
| --- | ---: | --- |
| Targeted `go test ./internal/documents ./internal/app -count=1` | 0 | Document and shell tests pass; touched-file formatter reports no diff. |
| `make -C cli check` | 2 | Vet, unit race, architecture and tooling pass. Existing format failures remain only in `cmd/mch/main.go` and `internal/app/clipboard.go`; package-comment lint remains in `internal/help/commands.go`, `internal/styles/styles.go` and `internal/ui/layout.go`. |
| `make -C cli coverage` | 2 | Complete unit campaign **3583/4084 (87.7326%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete-program and startup tests pass. |
| `make -C cli integration-coverage` | 2 | Complete 24-program plus one real PTY campaign **3082/4084 (75.4652%)**; strict >90% gate fails. |
| `git diff --check` | 0 | No whitespace errors. |

Both coverage status files say `complete: true`; all required scenarios passed.
The nonzero coverage exits are numerical gate failures. The 115 inputs in each
campaign's provenance manifest share sorted path/hash digest
`0e789db13bca6a93f52c599d81cf6f81372ad6c1f16052f7ae97c8cf988a1e24`.
The toolchain is `go1.26.8-X:nodwarf5`; covered PTY child SHA256 is
`eadd095f625e78ba1103ece65f3eec697e4e6eed411cc5cb7598359afc15781c`.
This checkpoint and implementation log were updated after the campaigns;
production and test inputs remain unchanged. Raw profiles, uncovered ranges,
package totals, scenario results and command journals are under
`cli/.coverage/{unit,integration}`.

| Package | Unit | Terminal |
| --- | ---: | ---: |
| `cmd/mch` | 0/3 | 1/3 |
| `internal/app` | 1693/2014 | 1439/2014 |
| `internal/changes` | 732/869 | 666/869 |
| `internal/documents` | 268/273 | 223/273 |
| `internal/dto` | 0/0 | 0/0 |
| `internal/epics` | 214/214 | 187/214 |
| `internal/help` | 6/6 | 6/6 |
| `internal/navigation` | 25/40 | 18/40 |
| `internal/projects` | 217/227 | 196/227 |
| `internal/styles` | 0/0 | 0/0 |
| `internal/testcases` | 130/132 | 112/132 |
| `internal/ui` | 8/8 | 8/8 |
| `pkg/client` | 290/298 | 226/298 |
| **Total** | **3583/4084** | **3082/4084** |

There are 501 uncovered unit statements and 1002 uncovered terminal statements.
The largest remaining gaps are `internal/app` (321 unit, 575 terminal) and
`internal/changes` (137 unit, 203 terminal). No live backend or database was
used. No commit or push was made.

## P6 review fixes 02 (2026-09-28)

Returning from version details and completing a history refresh now position the
history viewport around the selected version using the actual terminal height.
The shared document model owns this positioning for keyboard movement as well.
`TestP602SelectedVersionVisibleAfterDetailsAndRefresh` selects a deep history
row, scrolls its details, returns to history, refreshes after a new row arrives,
and checks that the same selected ID remains visible.

| Command on revision `71a46075310dc453e6c2af8c747aafd3944b955d` plus review diff | Exit | Evidence |
| --- | ---: | --- |
| Targeted `go test ./internal/documents ./internal/app -run 'TestP602SelectedVersionVisibleAfterDetailsAndRefresh\|TestP602HistorySelectionVisibleWithCatalogError\|TestP602HistorySelectionDetailsAndViewport' -count=1` | 0 | New regression and adjacent viewport tests pass. |
| `make -C cli check` | 2 | Vet, unit race, architecture and tooling pass. Existing format failures: `cmd/mch/main.go`, `internal/app/clipboard.go`; package-comment lint: `internal/help/commands.go`, `internal/styles/styles.go`, `internal/ui/layout.go`. |
| `make -C cli coverage` | 2 | Complete unit campaign **3581/4082 (87.7266%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete-program/startup suite passes. |
| `make -C cli integration-coverage` | 2 | Complete 24-program plus real PTY campaign **3080/4082 (75.4532%)**; strict >90% gate fails. |
| `python3 -B -m unittest discover -s scripts -p 'documentation_test.py' -v` (from `cli/`) | 0 | Three documentation checks pass. |
| `git diff --check` | 0 | No whitespace errors. |

Both coverage profiles report `complete: true`; all required scenarios passed.
During test refinement, an initial targeted run failed on an assertion that
compared a detail line with a history line; the corrected final run passed. A
`go test` attempted from the repository root also failed because the Go module
is under `cli/`.
The 115 input hashes in both provenance manifests share sorted path/hash digest
`915de8f329a58707886c938aa44b0ec231e602a285ef43c2b5a429a5ad72ca72`
(SHA256 over sorted `path\0hash\n` records). Toolchain:
`go1.26.8-X:nodwarf5`; covered PTY child binary SHA256:
`9f9ceb835e24da3e53b8f227faea3ddebf51afff5e8b3b158ad1a2b490365706`.
Raw profiles, uncovered statements/functions, package totals and command
journals are in `cli/.coverage/{unit,integration}`. Exact package counts:

| Package | Unit | Terminal |
| --- | ---: | ---: |
| `cmd/mch` | 0/3 | 1/3 |
| `internal/app` | 1693/2014 | 1439/2014 |
| `internal/changes` | 732/869 | 666/869 |
| `internal/documents` | 266/271 | 221/271 |
| `internal/dto` | 0/0 | 0/0 |
| `internal/epics` | 214/214 | 187/214 |
| `internal/help` | 6/6 | 6/6 |
| `internal/navigation` | 25/40 | 18/40 |
| `internal/projects` | 217/227 | 196/227 |
| `internal/styles` | 0/0 | 0/0 |
| `internal/testcases` | 130/132 | 112/132 |
| `internal/ui` | 8/8 | 8/8 |
| `pkg/client` | 290/298 | 226/298 |
| **Total** | **3581/4082** | **3080/4082** |

Unit has 501 uncovered statements; terminal has 1002. Largest gaps remain
in app (321 unit, 575 terminal) and changes (137 unit, 203 terminal). This
review used no live backend or database. No commit or push was made.

## P6 review fixes 01 (2026-09-28)

Document entry now requires confirmed project or epic details. Project selection
and failed detail reads clear that confirmation, and a successful detail read
restores it. Document `/retry` reloads the owner's catalog as well as history
and current rows, so insertion recovers after a transient catalog failure.
History scrolling positions the selected row from the rendered header, including
the catalog-error line. New unit and shell regressions cover each finding.

| Command on revision `5925b8f18837fda3ec6f391ac5477a9284288122` plus review diff | Exit | Evidence |
| --- | ---: | --- |
| Targeted `go test ./internal/projects ./internal/documents ./internal/app` | 0 | Owner load, catalog retry and selection visibility tests pass. |
| `make -C cli check` | 2 | Vet, unit race, architecture and tooling pass. Untouched format baseline: `cmd/mch/main.go`, `internal/app/clipboard.go`; package-comment lint: `internal/help/commands.go`, `internal/styles/styles.go`, `internal/ui/layout.go`. |
| `make -C cli coverage` | 2 | Complete unit campaign **3569/4073 (87.6258%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete-program/startup suite passes. |
| `make -C cli integration-coverage` | 2 | Complete 24-program plus real PTY campaign **3069/4073 (75.3499%)**; strict >90% gate fails. |
| `git diff --check` | 0 | No whitespace errors. |

Both coverage profiles report `complete: true`; all required scenarios passed,
and the nonzero coverage exits reflect numerical gates. The 115 source-file
hashes in each provenance manifest share sorted path/hash digest
`26d6172355bab2efa5683b6329f36c004d14ddeb16486b91ba3fc44cdbf16cbe`
(SHA256 over sorted `path\0hash\n` records). The toolchain is
`go1.26.8-X:nodwarf5`; the covered PTY child binary SHA256 is
`5501d89d6901cdb677de416e5a8da1421f5169b70339b5d219592e9e22d9c324`.
The checkpoint, contract ledger and implementation log were updated after the
campaigns; production and test inputs remained unchanged.
Raw profiles, uncovered statements/functions, package totals and command
journals are in `cli/.coverage/{unit,integration}`. Exact package counts:

| Package | Unit | Terminal |
| --- | ---: | ---: |
| `cmd/mch` | 0/3 | 1/3 |
| `internal/app` | 1689/2013 | 1438/2013 |
| `internal/changes` | 732/869 | 666/869 |
| `internal/documents` | 258/263 | 213/263 |
| `internal/dto` | 0/0 | 0/0 |
| `internal/epics` | 214/214 | 187/214 |
| `internal/help` | 6/6 | 6/6 |
| `internal/navigation` | 25/40 | 18/40 |
| `internal/projects` | 217/227 | 194/227 |
| `internal/styles` | 0/0 | 0/0 |
| `internal/testcases` | 130/132 | 112/132 |
| `internal/ui` | 8/8 | 8/8 |
| `pkg/client` | 290/298 | 226/298 |
| **Total** | **3569/4073** | **3069/4073** |

Unit has 504 uncovered statements; terminal has 1004. The largest gaps remain
in app (324 unit, 575 terminal) and changes (137 unit, 203 terminal). This
review did not use a live backend or database. No commit or push was made.

## P6 implementation (2026-09-28)

P6/spec026 is implemented on `change/026-cli-document-management` based on
`3be9207cf2344b186248c8a096fdab14d32b04fe`; caller-owned review and
publication remain. The shared document feature opens from project, epic and
change details, loads that owner's configured types, browses current and
historical versions, fetches scoped details, and appends exact editor bytes.
An insert result reaches the UI with its committed ID before a separate
history/current/details refresh. Failed refresh and `/retry` preserve that ID
without replaying insertion. P4 change editors and P5 testcase scenarios pass.

| Command on base revision plus P6 implementation | Exit | Evidence |
| --- | ---: | --- |
| Targeted `go test ./... -count=1` | 0 | All Go packages, including new client/model/app/program and PTY assertions, pass. |
| `make -C cli check` | 2 | Vet, unit race, architecture and tooling pass. Untouched format baseline: `cmd/mch/main.go`, `internal/app/clipboard.go`. Sampled package-comment lint baseline: `cmd/mch/main.go`, `internal/styles/styles.go`, `internal/ui/layout.go`. No touched-file static failure remains. |
| `make -C cli coverage` | 2 | Complete unit campaign **3534/4064 (86.9587%)**, strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete-program/startup tests pass. |
| `make -C cli integration-coverage` | 2 | Complete 24-program plus real PTY campaign **3063/4064 (75.3691%)**, strict >90% gate fails. |
| `git diff --check` | 0 | No whitespace errors. |

Both coverage profiles report `complete: true`; the coverage exits reflect
numerical gates, with no failed or skipped required scenario. Unit and terminal
profiles remain separate. Exact production package statement counts:

| Package | Unit | Terminal |
| --- | ---: | ---: |
| `cmd/mch` | 0/3 | 1/3 |
| `internal/app` | 1663/2013 | 1439/2013 |
| `internal/changes` | 732/869 | 666/869 |
| `internal/documents` | 255/260 | 210/260 |
| `internal/dto` | 0/0 | 0/0 |
| `internal/epics` | 214/214 | 187/214 |
| `internal/help` | 6/6 | 6/6 |
| `internal/navigation` | 25/40 | 18/40 |
| `internal/projects` | 211/221 | 190/221 |
| `internal/styles` | 0/0 | 0/0 |
| `internal/testcases` | 130/132 | 112/132 |
| `internal/ui` | 8/8 | 8/8 |
| `pkg/client` | 290/298 | 226/298 |
| **Total** | **3534/4064** | **3063/4064** |

Unit has 530 uncovered statements; terminal has 1001. The largest remaining
unit gaps are app (350 statements) and changes (137); terminal gaps are app
(574), changes (203), client (72), and documents (50). Full uncovered
statement ranges, function percentages, raw profiles, command journals and
scenario results are preserved under `cli/.coverage/unit/` and
`cli/.coverage/integration/`. The campaigns used Go
`go1.26.8-X:nodwarf5`; both provenance manifests record base revision
`3be9207cf2344b186248c8a096fdab14d32b04fe` and an identical sorted
input-hash digest
`c3de08fbac54edf2ed2bd7d791bb72dfd0a3a924496ce2d270e64451bf798e1c`.
The covered child binary SHA256 is
`390c3460b446f6401a2e5dc03365641f7078166caa4fe3d4cc2a1ef029a01b65`.
The manifest includes five new document programs and the extended PTY child.
Later checkpoint/ledger/plan edits only update documentation. No live backend,
database, Flow resource, agent executable, stage or production environment was
used. The final rebuild still needs the strict coverage gates and tracked
static baselines repaired in owning passes.


## P5 review fixes 02 (2026-09-28)

The testcase feature retains the committed operation and create ID through
repeated failed read-only refreshes. Change-detail `/retry` now uses testcase
refresh while that outcome is pending; after a successful read it shows the
saved/deleted outcome with refreshed feedback, then resumes ordinary change
detail reloads. A new write or scope invalidation clears the pending outcome.
The testcase feature now supplies create/edit placeholder, help and status text
to the shell's generic input widget. Unit, visible app and complete-program
regressions cover both findings. No commit or push was performed.

| Command on revision `74591cc6223f5f0e70c918dc243d016eee4b8811` plus review diff | Exit | Evidence |
| --- | ---: | --- |
| Targeted `go test -count=1 ./internal/testcases ./internal/app` | 0 | Feature outcome and form assertions plus visible shell assertions pass. |
| Targeted `go test -count=1 -timeout=60s ./integration -run '^TestCLIProgramTestCaseCommittedWriteAndStaleRecovery$' -v` | 0 | Two failed read-only retries preserve saved feedback and ID; one create; recovery passes. |
| `make -C cli check` | 2 | Vet, unit race, architecture and tooling pass. Tracked format failures: `cmd/mch/main.go`, `internal/app/clipboard.go`; package-comment lint in `internal/help/commands.go`, `internal/styles/styles.go`, `internal/ui/layout.go`. |
| `make -C cli coverage` | 2 | Complete unit campaign **3149/3578 (88.0101%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete program/startup suite passes. |
| `make -C cli integration-coverage` | 2 | Complete 19-program and real PTY campaign **2689/3578 (75.1537%)**; strict >90% gate fails. |
| `git diff --check` | 0 | No whitespace errors. |

Both profiles report `complete: true`; their nonzero exits are the numerical
gates, with no failed or skipped required scenario. Unit package counts include
app 1553/1806, changes 732/869, testcases 130/132 and client 264/273.
Terminal counts include app 1297/1806, changes 666/869, testcases 112/132
and client 207/273. Unit has 429 uncovered statements and terminal has 889.
The full package/function inventories and raw profiles are in
`cli/.coverage/{unit,integration}`. Both provenance manifests record sorted
`inputs_sha256` digest `ff46c89a4891cc4319a99a3f96a6f30383f582ea50cbf52dfc6f4026e6ae3171`;
the covered child binary SHA256 is
`b191879b3ddebdb51bf556110ea54f11e28eb2cf34b4f97e9ed9321e046e6eca`.
This checkpoint and ledger update follows the measured source and changes no
production or test input. Final rebuild acceptance remains incomplete due to
the tracked static baseline and coverage shortfalls. No stage or production
promotion is authorized.

## P5 review fixes 01 (2026-09-28)

The busy testcase key guard now accepts Esc and Ctrl+C, invalidates the
testcase context and follows the normal cancellation navigation. Other keys,
including a second Enter, remain blocked while the request is busy. The new
manifest-selected complete-program test sends both keys during a stalled create
and a stalled post-create list request. It observes HTTP cancellation, navigation
back to change details and only one create request. No commit or push was made;
the caller owns publication and the later dev merge. The P5 implementation
figures below are historical and are superseded by this review-fix evidence.

| Command on revision `5b252d98bc8c8579ee422ffab63bd58206ca59ca` plus review diff | Exit | Evidence |
| --- | ---: | --- |
| Targeted `go test -count=1 -timeout=60s ./integration -run '^TestCLIProgramTestCaseKeyboardCancelsBusyRequest$' -v` | 0 | All four write/refresh × Esc/Ctrl+C cases pass. |
| `make -C cli check` | 2 | Vet, race unit tests, architecture and tooling pass. Existing formatting failures in `cmd/mch/main.go` and `internal/app/clipboard.go`; sampled package-comment lint in `cmd/mch/main.go`, `internal/styles/styles.go`, `internal/ui/layout.go`. |
| `make -C cli coverage` | 2 | Complete unit campaign **3130/3558 (87.9708%)**, below strict >95%. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | All complete-program/startup scenarios pass. |
| `make -C cli integration-coverage` | 2 | Complete 19-program and real PTY campaign **2669/3558 (75.0141%)**, below strict >90%. |
| `git diff --check` | 0 | No whitespace errors. |

Both campaigns report complete measurements; no required scenario failed or
skipped. Unit package counts: app 1549/1801, changes 732/869, testcases 115/117,
client 264/273, with other packages in the raw report. Terminal package counts:
app 1292/1801, changes 666/869, testcases 97/117, client 207/273, with
other packages in the raw report. Unit has 428 uncovered statements; terminal
has 889. The full package and uncovered inventories, scenario list, command
exits and source hashes are in `cli/.coverage/{unit,integration}`. Both
provenance manifests share sorted `inputs_sha256` digest
`6f289a191c670cf85799e315f4d5ca237d390cc3f0b374171d4f203789435e1a`;
the covered child binary SHA256 is
`4f1fdfc54509e7330f7f3053bb621f7f30a96fffb3c24509be38c4373a5aecf6`.
Go is 1.26.8-X:nodwarf5, golangci-lint 2.13.1, govulncheck 1.7.0 and
socat 1.8.1.1. This checkpoint update follows measurement and changes no
production or test input.

The review finding is fixed; final rebuild acceptance remains incomplete due
to the existing static baseline and numerical coverage shortfalls. No stage or
production promotion is authorized.

## P5 implementation (2026-09-28)

P4/spec023 is actually merged on dev at
`1904de958e2eeb0393b697f07faaa6d177c795a2`. P5/spec024 uses branch
`change/024-cli-testcase-management` at `de96ad39c11ca8f8c93b122225bd4612efdc0163`
plus its uncommitted implementation. The factory caller owns native review,
implementation/review commits and pushes, and the later dev merge. This child
performed no Git publication. Earlier P4 entries below are historical and their
pending-publication status is superseded by the actual merge above.

The testcase adapter now has exact list/create/update/update-done/delete POST
contracts. Create retains its 201 ID; other writes accept empty 204 responses.
Typed list rows preserve large integer IDs through the CLI DTO, owner, scenario, explicit false,
created/updated times and ID order. The testcase feature owns form target and
validation, cancellation/revision state, committed outcomes and write → list →
change-details sequencing. The shell routes results into the embedded change
screen, preserving the selected row and document draft. A committed write followed
by a failed read reports the committed outcome and offers read-only `/retry`;
it cannot replay the write. The [ledger](cli-contracts.md) maps every criterion,
retained assertion and new complete-program/PTY scenario. No live backend or
database was used.

| Final-source command | Exit | Evidence |
| --- | ---: | --- |
| `make -C cli check` | 2 | Vet, race unit tests, architecture and tooling pass. Untouched format baseline: `cmd/mch/main.go`, `internal/app/clipboard.go`. Sampled revive package-comment baseline: `cmd/mch/main.go`, `internal/styles/styles.go`, `internal/ui/layout.go` (earlier run sampled `internal/help/commands.go`). No new static/test failure remains. |
| `make -C cli coverage` | 2 | Complete unit campaign **3129/3553 (88.0664%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete program/startup suite passes. |
| `make -C cli integration-coverage` | 2 | Complete selected program and real PTY campaign **2657/3553 (74.7819%)**; strict >90% gate fails. |
| `git diff --check` | 0 | No whitespace errors. |

Both `cli/.coverage/{unit,integration}/status.json` files say `complete: true`;
no required scenario failed or skipped. The integration manifest selects 18
complete-program tests and one real PTY child; the three new testcase programs
cover lifecycle, committed-write/read-failure recovery, malformed data, stale
rows and cancellation. The PTY child creates a testcase and redraws/scans the
detail viewport. Unit leaves 424 and terminal 896 production statements
uncovered. Significant package gaps: app 1548/1796 unit and 1283/1796 terminal;
changes 732/869 and 666/869; navigation 25/40 and 17/40; client 264/273 and
207/273. The touched testcase package is 115/117 unit and 95/117 terminal.
The full package/function/statement tables and raw profiles are under
`cli/.coverage/{unit,integration}`. Testless zero-statement `dto` and `styles`
remain in the inventory.

Both final-source provenance manifests identify base revision
`de96ad39c11ca8f8c93b122225bd4612efdc0163` and include source/test/tool
hashes; sorted `inputs_sha256` digest is
`825cd42deb0ae6c8ed5ddeb66446aec98d2ba25193ba3f2b30c5b60629c8848e`.
The covered PTY child SHA256 is
`baa15d01665e141ac76025f92e66b5e45831413cc74b27797cb89c8cf0e19b09`.
Go is 1.26.8-X:nodwarf5, golangci-lint 2.13.1, govulncheck 1.7.0, and
socat 1.8.1.1. This checkpoint, plan and implementation-log update follows the
measured source and does not change production or test inputs.

P5 awaits caller-owned native review, any review fixes/revalidation, and dev
merge. P6 documents is the next planned pass. Overall CLI rebuild completion
remains pending because the final coverage thresholds and tracked static
baseline are unmet. No stage or production promotion is authorized.

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
