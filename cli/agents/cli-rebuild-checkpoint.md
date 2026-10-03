# CLI rebuild checkpoint

## 031 review fixes 07: history terminal-control filtering (2026-10-03)

The P1 finding is valid and fixed. History decodes the complete captured bat
output and permits printable Unicode, tabs/newlines and complete SGR syntax
colors only. OSC clipboard/hyperlink commands, non-SGR CSI, other escape/string
controls, C0/C1 and malformed/incomplete controls cannot reach the viewport.
Filtering precedes line splitting, clipping and scroll bounds; stored document
and comment bodies and captured output remain intact. HTTP contracts are unchanged.
The assertion ledger records the feature/root regressions and the existing
complete-program scenario extended with stored OSC 52 and erase-scrollback input
passed through real bat. Retained color, resize, scrolling, version and PTY tests
pass. No manifest change was needed because the existing scenario was extended.

Tested base revision: `f246b6bf0c3f05facb688c716d7a305d0d329440` plus this
uncommitted fix. Both campaigns recorded the same production source SHA-256 for
`internal/documents/history.go`:
`808af512ffa23a8e2b33ff7b06b3f7ee5144f7c0eb730da9455c7898e7bcce1c`.
Full source/test hashes, tool versions, command exits, profiles, scenario inventory,
package totals and uncovered functions/statements are in the independent
`cli/.coverage/{unit,integration}/` artifacts. The covered child binary SHA-256 is
`5fff35c4b0dd3cc70530d0054f8e228773336657fa37064cba697b0c8044484f`.
Only this checkpoint and the implementation log changed after the campaigns.

| Command | Exit and result |
| --- | --- |
| `make -C cli check` before edits | 0; baseline checks all pass. |
| `go test -count=1 ./internal/documents -run '^Test031HistoryFilters'` from `cli/`, before fix | 1; intentionally reproduces controls leaking and incorrect multiline control scroll bounds; `/tmp/031-review-07-reproduction.log`. |
| `golangci-lint fmt --no-config --enable gofumpt --enable goimports internal/documents/history.go internal/documents/history_test.go internal/app/docs_refactor_test.go integration/docs_refactor_program_test.go` from `cli/` | 0; formats only touched Go files. |
| `go test -count=1 ./internal/documents ./internal/app -run '^Test031History'` from `cli/` | 0 on final source; initial run exposed an expected trailing SGR reset missing from the new clipping assertion, which was corrected. |
| `go test -count=1 -timeout=1m ./integration -run '^TestCLIProgram031DocumentCommentsAndHistory$'` from `cli/` | 0; real bat and fake HTTP server, `/tmp/031-review-07-target-program.log`. |
| `make -C cli check` | 0; formatting, lint, vet, unit race, architecture and tooling all pass; `/tmp/031-review-07-check.log`. |
| `make -C cli coverage` | 0; **5593/6434 (86.9288%)**, independent 80% gate passes; `/tmp/031-review-07-coverage.log`. |
| `make -C cli deps-audit` | 0; no vulnerabilities; `/tmp/031-review-07-deps-audit.log`. |
| `make -C cli integration-coverage` | 0; **4943/6434 (76.8262%)**, independent 70% gate passes; all 34 complete-program/startup scenarios and the real PTY scenario pass; `/tmp/031-review-07-integration-coverage.log`. |

Both coverage campaign statuses are complete with exit 0. No final failing,
skipped or blocked scenarios. Go is `go1.26.8-X:nodwarf5`, golangci-lint is
2.13.1 and govulncheck is v1.7.0. The new filter has 12/14 unit-covered statements;
only its defensive decoder zero-progress fallback is unexercised. Remaining
production gaps are reported below and detailed in each `uncovered.txt` and
`functions.log`; zero-statement packages remain in the audited inventory.

| Package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cli/cmd/mch` | 0/3 | 1/3 |
| `cli/internal/agent` | 297/357 | 270/357 |
| `cli/internal/app` | 2541/3069 | 2337/3069 |
| `cli/internal/changes` | 862/989 | 742/989 |
| `cli/internal/configurations` | 170/183 | 167/183 |
| `cli/internal/documents` | 502/527 | 417/527 |
| `cli/internal/dto` | 0/0 | 0/0 |
| `cli/internal/epics` | 221/221 | 191/221 |
| `cli/internal/health` | 48/50 | 47/50 |
| `cli/internal/help` | 6/6 | 6/6 |
| `cli/internal/navigation` | 25/40 | 19/40 |
| `cli/internal/projects` | 224/234 | 200/234 |
| `cli/internal/styles` | 0/0 | 0/0 |
| `cli/internal/testcases` | 130/132 | 112/132 |
| `cli/internal/ui` | 21/21 | 17/21 |
| `cli/pkg/briefprocess` | 128/161 | 84/161 |
| `cli/pkg/client` | 402/424 | 318/424 |
| `cli/pkg/documentprocess` | 16/17 | 15/17 |

Backend code is unchanged; backend suites were not rerun. No live backend or
database access, Git publication, stage/production promotion or deployment.

## CLI check cleanup (2026-09-30)

The CLI formatting and package-comment baseline is fixed. The formatter
updated `cmd/mch/main.go` and `internal/app/clipboard.go`; package comments
were added to `cmd/mch`, `internal/help`, `internal/styles`, and `internal/ui`.
The changes do not alter runtime behavior. The current `make -C cli check`
passes, including format-check, lint, vet, race, architecture, and tooling.

| Command | Exit and result |
| --- | --- |
| `make -C cli format` | 0; formatted the two previously failing Go files. |
| `make -C cli check` | 0; all constituent checks pass. |
| `make -C cli coverage` | 0; **4987/5784 (86.2206%)**, at least 80% gate passes. |
| `make -C cli deps-audit` | 0; no vulnerabilities found. |
| `make -C cli integration-coverage` | 0; complete-program and real PTY campaign **4416/5784 (76.3485%)**, at least 70% gate passes. |
| `git diff --check` | 0; no whitespace errors. |

The user-modified `backend/config/dev.yaml` remains untouched. Earlier
checkpoint entries record the prior static-check baseline at their timestamps.

## Persistent Changes list filters, selection, and display (2026-09-30)

The Changes list summary now names the four menu filters in their menu order.
Phase, Types, Epic, and Find stay active through detail/Main navigation, list
reloads, and project selection until explicitly cleared. `/clear-filters`
clears all four; `@clear` in Phase, Types, or Epic clears just that filter.
The selector caption and command use the plural Types name. A blank Find entry
does not clear its saved query. Ordinary prompt typing adds a temporary word-prefix
query without replacing the saved one. Returning from details restores the
selected change by ID. The Changes list menu and help omit `/retry` and
`/brief-new`; returning to Main and reopening `/changes` reloads the list.
After a failed delete refresh, the list status gives this same reload path.
List Epic cells show the epic name without an ID suffix. List and detail Types
use AccentPurple; list `%` and detail Complete use AccentBlue. Unit tests cover
each rule, and the complete-program recovery tests follow the new reload path.

| Command | Exit and result |
| --- | --- |
| Focused `go test` on affected app/changes/integration cases | 0; filter, selection, display, and program recovery cases pass. |
| `make -C cli check` | 2; vet, race, architecture, and tooling pass. The existing formatting failures in `cmd/mch/main.go` and `internal/app/clipboard.go` and package-comment lint failures in `internal/help/commands.go`, `internal/styles/styles.go`, and `internal/ui/layout.go` remain. |
| `make -C cli coverage` | 0; **4987/5784 (86.2206%)**, at least 80% gate passes. |
| `make -C cli deps-audit` | 0; no vulnerabilities found. |
| `make -C cli integration-coverage` | 0; complete-program and real PTY campaign **4416/5784 (76.3485%)**, at least 70% gate passes. |
| `git diff --check` | 0; no whitespace errors. |

An initial terminal campaign failed two program scenarios that still invoked
`/retry` from the Changes list. The scenarios now return to Main and reopen
the list; the final complete campaign passes. Coverage profiles and package
totals are under `cli/.coverage/{unit,integration}/`. The user-modified
`backend/config/dev.yaml` was left untouched.

## Change detail identity and association rows (2026-09-30)

The detail table now begins ID, Ref UUID, Slug, Epic, Phase, Types, After Change,
a separator, and Title. There is no separate Ref row. Slug shows the full
`ref_slug` in AccentCyan and retains suffix-only editing. Types and Title use
the Foreground color. Epic shows only the plain `vw_change_details.epic_name`.
After Change shows the nullable `after_change_name` and opens its editor with
the association ID; null opens a blank input. Unit tests cover row order,
styling, display and prompt values; complete-program and PTY tests use the
current row positions.

| Command | Exit and result |
| --- | --- |
| `make -C cli check` | 2; vet, race, architecture, and tooling pass. The existing format failures in `cmd/mch/main.go` and `internal/app/clipboard.go`, and package-comment lint in `cmd/mch`, `internal/help`, `internal/styles`, and `internal/ui`, remain. The specific three lint findings vary by run. |
| `make -C cli coverage` | 0; **4937/5738 (86.0404%)**, at least 80% gate passes. Earlier failing runs expected stdout before stderr in `TestP803RunnerCancellationProgressAndReaping`; that test now accepts either child stream first. |
| `make -C cli deps-audit` | 0; no vulnerabilities found. |
| `make -C cli integration-coverage` | 0; complete-program and real PTY campaign **4387/5738 (76.4552%)**, at least 70% gate passes. An initial run exposed stale row indices and label expectations; a later run was invalidated by a checkpoint edit during measurement. |

## Checked selector row background (2026-09-30)

The checked Types option now uses the selected row's foreground color when
highlighted. Its marker and option text therefore share the menu background
across the full row. Unhighlighted checked options keep the green check.
The focused Types selector test compares the complete ANSI-rendered selected
line to a single styled row.

| Command | Exit and result |
| --- | --- |
| `make -C cli check` | 2; vet, race, architecture, and tooling pass. Only the recorded format and package-comment lint baseline remains. |
| `make -C cli coverage` | 0; **4929/5730 (86.0209%)**, at least 80% gate passes. |
| `make -C cli deps-audit` | 0; no vulnerabilities found. |
| `make -C cli integration-coverage` | 0; complete program and real PTY campaign **4384/5730 (76.5096%)**, at least 70% gate passes. |
| `git diff --check` | 0. |

## Selected selector marker (2026-09-30)

Selected single-choice options now use the larger centered dot `[●]` in the
shared selector renderer. Focused Phase and Epic selector tests assert the
marker; Types keeps its green check.

| Command | Exit and result |
| --- | --- |
| `make -C cli check` | 2; vet, race, architecture, and tooling pass. Only the recorded format and package-comment lint baseline remains. |
| `make -C cli coverage` | 0; **4927/5728 (86.0161%)**, at least 80% gate passes. |
| `make -C cli deps-audit` | 0; no vulnerabilities found. |
| `make -C cli integration-coverage` | 0; complete program and real PTY campaign **4380/5728 (76.4665%)**, at least 70% gate passes. |
| `git diff --check` | 0. |

## Change `ref_slug` contract (2026-09-30)

Change list and details decode the nullable `ref_slug` field instead of separate
`ref` and `slug` fields. The displayed reference is taken from its prefix. The
Slug editor displays the full `ref_slug`, lets the user change only its suffix,
and sends that suffix alone as `slug` to `/change/update-slug`. The demo seed
stores only the suffix. The view builds `006-some-slug` for ref 6 and
`1116-some-slug` for ref 1116. A change without an assigned ref has a null
`ref_slug`, so its editor cannot prefill the stored suffix from the read API.

| Command | Exit and result |
| --- | --- |
| `make -C cli check` | 2; vet, race, architecture, and tooling pass. The same existing format failures in `cmd/mch/main.go` and `internal/app/clipboard.go`, and package-comment lint in `internal/help`, `internal/styles`, and `internal/ui`, remain. |
| `make -C cli coverage` | 0; **4927/5728 (86.0161%)**, at least 80% gate passes. |
| `make -C cli deps-audit` | 0; no vulnerabilities found. |
| `make -C cli integration-coverage` | 0; complete program and real PTY campaign **4383/5728 (76.5189%)**, at least 70% gate passes. |
| Focused isolated PostgreSQL `ref_slug` and demo seed tests | 0; both tests pass. |
| `git diff --check` | 0. |

## Historical inline change detail prompts and cancellation (2026-09-29)

Phase, Epic, and Types selectors now render on ChangeDetailsScreen. Title,
PR URL, After change, and Slug use labeled inline prompts. Other text form
prompts and the configuration editor use purple `Label >` captions and green
input. Slug displays an
immutable numeric prefix from Ref, or the change ID until Ref is assigned;
only lowercase `[a-z0-9_-]` suffix characters enter the prompt. Selectors
render `[·]` for the current single choice and a green check inside brackets
for selected Types. At that stage, Epic was shown as `name #id` in the details
row and chooser; the current detail row uses the plain view name.
Project ID and Epic ID detail rows and the two extra spacers around the prompt
were removed. Escape and Ctrl+C cancel active prompts in place; inactive
prompts navigate to the previous screen. The PTY scenario checks these keys
across health, configuration, project, epic, document, brief, change detail,
and test case screens, including slug save and invalid input.

| Command | Exit and result |
| --- | --- |
| `make -C cli check` | 2; vet, race, architecture, and tooling pass. Only the existing format failures in `cmd/mch/main.go` and `internal/app/clipboard.go`, and package-comment lint in `internal/help`, `internal/styles`, and `internal/ui`, remain. |
| `make -C cli coverage` | 0; **4928/5729 (86.0185%)**, at least 80% gate passes. |
| `make -C cli deps-audit` | 0; no vulnerabilities found. |
| `make -C cli integration-coverage` | 0; complete program and real PTY campaign **4382/5729 (76.4880%)**, at least 70% gate passes. |
| `git diff --check` | 0. |

An earlier complete-program campaign failed old expectations for the removed
ChangeUpdateScreen and old cancel behavior. Those assertions were updated and
the full campaign passed. One later unit coverage run failed an unrelated
agent-process timing assertion while running alongside backend coverage; its
profile was incomplete and was discarded. The subsequent isolated campaign
produced the passing unit measurement above. No live backend or database was
used for CLI tests.

## Changes list empty filtered results (2026-09-29)

A prompt or selector filter with zero matching changes now keeps the boxed
changes table and its column labels. An AccentRed `No changes match filters.`
message starts at the Title column position, and the box ends with
`Rows 0-0 of 0`. A truly empty unfiltered list retains `No changes.`. Unit
tests cover the filtered prompt, both loaded and empty source lists, message
color and alignment, box width, and the zero-row count. No live backend or
database was used.

| Command from repository root | Exit and result |
| --- | --- |
| Focused `go test ./internal/changes ./internal/app` (inside `cli/`) | 0; relevant view and prompt tests pass. |
| `make -C cli check` | 2; vet, race unit tests, architecture, and tooling pass; only the previously recorded untouched format and package-comment lint baseline remains. |
| `make -C cli coverage` | 0; complete unit campaign **4854/5626 (86.2780%)**, above strict >80%. |
| `make -C cli deps-audit` | 0; no vulnerabilities found. |
| `make -C cli integration-coverage` | 0; complete program plus real PTY campaign **4274/5626 (75.9687%)**, above strict >70%. |
| `git diff --check` | 0; no whitespace errors. |

The independent final profiles and package totals are under
`cli/.coverage/{unit,integration}/`. The remaining static baseline is listed
in the match-background checkpoint below.

## Changes list prompt spacing (2026-09-29)

The ChangesListScreen no longer inserts a blank row between the prompt and
footer. Its measured table viewport gains that row, so the boxed list shows
one additional data row or an interior blank row when fewer changes exist.
The short-height unit layout test and real PTY scrolling scenario use the new
row counts. Other screens retain their spacing. No live backend or database
was used.

| Command from repository root | Exit and result |
| --- | --- |
| Focused `go test` for `internal/app` layout and real PTY scrolling (inside `cli/`) | 0 each. |
| `make -C cli check` | 2; vet, race unit tests, architecture, and tooling pass; only the previously recorded untouched format and package-comment lint baseline remains. |
| `make -C cli coverage` | 0; complete unit campaign **4847/5619 (86.2609%)**, above strict >80%. |
| `make -C cli deps-audit` | 0; no vulnerabilities found. |
| `make -C cli integration-coverage` | 0; complete program plus real PTY campaign **4267/5619 (75.9388%)**, above strict >70%. |
| `git diff --check` | 0; no whitespace errors. |

The final independent profiles and package totals are under
`cli/.coverage/{unit,integration}/`. The remaining static baseline is described
in the next checkpoint entry.

## Changes list match backgrounds (2026-09-29)

The shared palette now defines `MutedPurple` (`#5F5F87`), `MutedRed`
(`#875F5F`), and `MutedGreen` (`#5F875F`). The selected row keeps the purple
background. Each visible query-word prefix in Ref, Phase, Types, Epic, and
Title is rendered white on green, including on that selected row. Rendering is
local to the changes list; no table component changes were needed. Focused
tests verify all five fields on selected and unselected rows, punctuation in
titles, preserved table width, and the final boxed view. No live backend or
database was used.

| Command from repository root | Exit and result |
| --- | --- |
| Focused `go test ./internal/changes ./internal/app` (inside `cli/`) | 0; rendering and prompt tests pass. |
| `make -C cli check` | 2; vet, race unit tests, architecture, and tooling pass. The only failures are the previously recorded untouched format and package-comment lint files listed below. |
| `make -C cli coverage` | 0; complete unit campaign **4845/5617 (86.2560%)**, above strict >80%. |
| `make -C cli deps-audit` | 0; no vulnerabilities found. |
| `make -C cli integration-coverage` | 0; complete program plus real PTY campaign **4262/5617 (75.8768%)**, above strict >70%. |
| `git diff --check` | 0; no whitespace errors. |

The static baseline remains `cmd/mch/main.go` and `internal/app/clipboard.go`
for formatting, plus package comments in `internal/help/commands.go`,
`internal/styles/styles.go`, and `internal/ui/layout.go`. Both final coverage
campaigns are complete, with independent profiles and package totals under
`cli/.coverage/{unit,integration}/`.

## Changes list prompt finder (2026-09-29)

On uncommitted HEAD `c188389`, the changes list menu no longer offers
`/find`. Typing in the prompt now filters the visible rows immediately; each
normalized query word must prefix a word in the displayed Ref, Phase, Types,
Epic, or Title. Matching ignores case and excludes hidden brief/spec fields.
Backspace updates the results, Ctrl+C clears the prompt filter, and Enter
opens the selected matching change. The existing `/find-filter` command remains
available. Unit tests cover normalization, multiword matching, menu contents,
live editing, clearing, and selection; the complete-program and PTY campaigns
pass. No live backend or database was used.

| Command from repository root | Exit and result |
| --- | --- |
| Focused `go test` for affected packages and delayed program cases, plus focused real PTY test | 0 each; the corrected assertions pass. |
| `make -C cli check` | 2; vet, race unit tests, architecture and tooling pass. Only the previously recorded untouched format failures in `cmd/mch/main.go` and `internal/app/clipboard.go` and package-comment lint findings in `internal/help/commands.go`, `internal/styles/styles.go`, and `internal/ui/layout.go` remain. |
| `make -C cli coverage` | 0; complete unit campaign **4808/5580 (86.1649%)**, above the current strict >80% gate. |
| `make -C cli deps-audit` | 0; no vulnerabilities found. |
| `make -C cli integration-coverage` | 0; complete program plus real PTY campaign **4259/5580 (76.3262%)**, above the current strict >70% gate. |
| `git diff --check` | 0; no whitespace errors. |

The initial terminal campaign stopped at an obsolete PTY assertion naming the
removed menu item; it passed after that assertion was updated. One later unit
campaign was incomplete because the existing
`TestP803RunnerCancellationProgressAndReaping` timing test missed its expected
`started` stderr marker. A fresh rerun completed and produced the unit total
above. Both final coverage profiles and package totals are under
`cli/.coverage/{unit,integration}/`; neither total includes hits from the other
campaign. The static baseline still prevents a passing `check` result.

## P8 review fixes 04: original brief and duplicate agent fields (2026-09-28)

Both review findings were valid. Existing-change preflight still shows the
backend brief as the initial draft, but the first actual user edit becomes the
immutable original sent to the runner; the backend version remains separate.
The process adapter now rejects duplicate top-level JSON fields before decoding
the result, including conflicting `ready_for_spec` values. Focused controller
and adapter tests cover exact bytes through an approved rewrite and duplicate
readiness/revision fields. No backend or live agent was used.

Final source is uncommitted on HEAD
`9604fe597a55a3a28deeeb27022655c5c6248a05`. Unit and terminal campaigns
record this revision, the complete input hashes, tool versions and separate raw
profiles under `cli/.coverage/`. Source SHA-256: `internal/agent/model.go`
`70c80494cc655015e1ed250681ee114bc096349eea121396e2fbfea27477693c`,
`pkg/briefprocess/runner.go`
`d96993bd5ad32118e842e80f758d81b7a22c4f6c5364ee69a5acf1774a7ae81c`.
The manifest is unchanged. Final commands from the repository root:

| Command | Exit and result |
| --- | --- |
| `go test ./internal/agent ./pkg/briefprocess -count=1` (inside `cli/`) | 0; focused tests pass. |
| `make -C cli check` | 2; vet, uncached race unit tests, architecture and tooling tests pass. Only the recorded untouched format baseline (`cmd/mch/main.go`, `internal/app/clipboard.go`) and package-comment lint baseline (`internal/help/commands.go`, `internal/styles/styles.go`, `internal/ui/layout.go`) fail. |
| `make -C cli coverage` | 2; complete unit campaign **4660/5452 (85.4732%)**, below strict >95%. `internal/agent` 265/319; `pkg/briefprocess` 116/149. |
| `make -C cli deps-audit` | 0; no vulnerabilities found. |
| `make -C cli integration-test` | 0; complete-program suite passes. |
| `make -C cli integration-coverage` | 2; all 31 selected program scenarios and the real PTY child pass without skips, crashes or missing counters; complete terminal campaign **4051/5452 (74.3030%)**, below strict >90%. `internal/agent` 241/319; `pkg/briefprocess` 0/149. |

The two coverage status files report complete campaigns with failing numerical
gates. Broader gaps remain in `internal/app` (unit 2085/2585, terminal
1883/2585), `internal/changes` (732/869, 666/869), and
`internal/navigation` (25/40, 18/40); the full package reports are in the
artifacts. These shortfalls and the untouched static baseline prevent a final
rebuild completion claim. The authorized intermediate policy permits the caller
to continue its dev review/merge sequence with these results visible. No commit,
push, stage or production promotion was performed here.

## P8 brief clarification implementation (028, 2026-09-28)

[Specification](../../agent/specs/028-cli-brief-clarification.md). P7 is merged
on dev at `b20068e7fb5c8e3a8a796fe8348024506fc1f7f1`. P8 implementation is
on `change/028-cli-brief-clarification`, based on HEAD
`6ec2446c8cda5e45d91fa7cea4747b24f1f841d0` with uncommitted P8 files.
This checkpoint records the implementation campaign before review or helper
publication; it does not claim a dev merge. The user-authorized helper owns
commits, pushes, review and merge. No stage or production promotion occurred.

The fixed `internal/agent` controller now owns new/existing brief intake,
project/change/document identity, exact original user input, a mutable draft,
current backend version, committed IDs, revision/generation, questions,
answers, blockers and explicit ready-for-spec handoff. `brief` and `spec`
catalog entries and new-change `backlog` are required. New creates use the
backend's initial brief write; existing user edits and accepted agent revisions
append `brief` documents with the correct `agent_edit` flag. A committed write
survives read/runner failure, and `/retry` repeats only the failed read or
agent step. `/reload` revalidates a changed backend brief. Existing ordinary
change/document/health actions remain independent of the runner. No spec
document or P9 action is started.

The process adapter reads only the two P8 prompts when invoked. Each invocation
uses an owned `.mch/tmp/brief-*` directory with distinct original/current
brief, context, question, answer and output paths. Its JSON output requires
the complete schema and matching revision. The subprocess is cancelable,
bounded and reaped; cancellation cleans known owned files, while failures
retain the reported path. Cleanup refuses unowned or symlink paths. The
shell offers `/brief-new` from Main/Changes and `/brief-clarify` from loaded
details, editor handoff, question answers, scrolling, ready status and safe
return. The architecture boundary test rejects agent → shell/adapter imports
and adapter → agent imports.

Fresh commands on the implementation source (all from repository root):

| Command | Exit and result |
| --- | --- |
| `make -C cli check` | Exit 2. Vet, uncached race unit tests, architecture and tooling tests pass. Format check still fails only on untouched `cli/cmd/mch/main.go` and `cli/internal/app/clipboard.go`; lint still reports only the existing package-comment findings in `cli/internal/help/commands.go`, `cli/internal/styles/styles.go`, and `cli/internal/ui/layout.go`. No new static failure. |
| `make -C cli coverage` | Exit 2 solely at strict final gate. Complete fresh unit profile: **4600/5413 (84.9806%)** production statements; target strictly >95%. |
| `make -C cli deps-audit` | Exit 0; no vulnerabilities found. |
| `make -C cli integration-test` | Exit 0; complete-program tests pass. |
| `make -C cli integration-coverage` | Exit 2 solely at strict final gate. Complete fresh terminal profile: **4034/5413 (74.5243%)** production statements; target strictly >90%. Its manifest ran 31 complete-program scenarios plus the real PTY child with no skipped or failed scenario. |

The profiles remain independent and complete. Package counts are in
`cli/.coverage/{unit,integration}/result.json` and `report.txt`, with raw
profiles, scenario events, provenance and uncovered statements/functions in
those ignored directories. Unit gaps include `internal/agent` 243/301,
`internal/app` 2059/2580, `internal/changes` 732/869,
`internal/navigation` 25/40 and `pkg/briefprocess` 104/133. Terminal gaps
include `internal/agent` 227/301, `internal/app` 1880/2580,
`internal/changes` 666/869, `internal/navigation` 18/40,
`pkg/briefprocess` 0/133 and `pkg/client` 295/394. Production inventory
includes untested packages; `dto` and `styles` have zero executable statements.
The terminal suite uses injected fake HTTP/runner collaborators plus the real
PTY executable; the concrete process adapter is covered by isolated unit
tests and still has no terminal hits. No live backend or real Codex process was
used or designated.

The coverage provenance ties the campaigns to HEAD above, Go
`go1.26.8-X:nodwarf5 linux/amd64`, all input SHA-256 values and the exact
scenario inventory. Selected source hashes: `internal/agent/model.go`
`07166cd8c1a21cefd064e05b8a838c3c09b7cb27ebb93755133fb5c75a1c3a70`,
`internal/app/brief.go`
`c33ee3adf67c780948b892c2d18b7addf5618120c08f82e055e9cbf7dec52cce`,
`pkg/briefprocess/runner.go`
`b54323e9dcaa3b4dbe95e628391036ee8273bdd2bf7a55af1ea458b1b3b5cf0c`,
and `scripts/terminal-scenarios.json`
`aadeeba6e404bec53767c2bcaf51eed6729c234a5b332a0db7f22086f10080ff`.
`cli/agents/cli-contracts.md` maps every P8 acceptance bullet to named unit
and terminal evidence. Numerical shortfalls are permitted for this intermediate
dev pass under `AGENTS.md` but are not a final rebuild completion claim.
Next: review P8 with the authorized helper, fix findings and rerun affected
checks/coverage, then let the factory perform its dev merge. P9 owns spec
write/review/fix and must consume only this explicit ready handoff.

## P7 review fix 08 (2026-09-28)

Leaving configuration management after an update still cancels its in-flight
selected-project catalog read and invalidates the old result. Navigation now
starts a separate read-only `/project/config` request with the new generation,
so the selected project's document, phase and type options reload on the main
screen. Quitting starts no replacement. The unit regression holds the first
request open, observes cancellation, rejects the late result, and verifies the
replacement's options. No live backend or database was used; this child made
no commit or push.

| Command on `a0d30a90a9992c722a055b3f5dcb620540438ce9` plus review diff | Exit | Evidence |
| --- | ---: | --- |
| `go test -count=1 ./internal/app -run '^TestP703ConfigurationExitCancelsCatalogRefreshAndIgnoresLateResult$' -v` | 0 | Return and quit cases pass; the return case observes two catalog reads and restored options. |
| `make -C cli check` | 2 | Vet, unit race, architecture and tooling pass. Existing format failures remain in `cmd/mch/main.go` and `internal/app/clipboard.go`; untouched package-comment lint remains in `internal/help/commands.go`, `internal/styles/styles.go` and `internal/ui/layout.go`. |
| `make -C cli coverage` | 2 | Complete unit campaign **4184/4763 (87.8438%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete-program and startup suite passes. |
| `make -C cli integration-coverage` | 2 | Complete program plus real PTY campaign **3642/4763 (76.4644%)**; strict >90% gate fails. |

Both coverage campaigns report `complete: true`. The terminal campaign ran all
28 program scenarios and its real PTY scenario; no scenario failed or skipped.
Unit leaves 579 statements and terminal leaves 1121. Both raw profiles,
uncovered statements/functions, per-source SHA256 hashes, source revision,
scenario results, and command journals are under `cli/.coverage/{unit,integration}`.
The measured `internal/app/update.go` SHA256 is
`716be681c86edc0e954675f7923b8ae5823559cd829a3a7e22a0d297a2de2dac`;
the regression test SHA256 is
`f7b3e726275f334dcfcc3ec150a383c1a91a37bdc7c5708065e75e16a4c2c4c8`.
The covered child binary SHA256 is
`8582fef674783149dd009c206c226472eff6c4c7bc4cbcd69466955b64bea32b`.
Toolchain: Go `go1.26.8-X:nodwarf5`, golangci-lint `2.13.1`, govulncheck
`v1.7.0`; socat was present for the PTY campaign. Documentation updates
followed measurement; the measured Go source did not change.

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cmd/mch` | 0/3 | 1/3 |
| `internal/app` | 1990/2364 | 1716/2364 |
| `internal/changes` | 732/869 | 666/869 |
| `internal/configurations` | 170/183 | 167/183 |
| `internal/documents` | 268/273 | 223/273 |
| `internal/dto` | 0/0 | 0/0 |
| `internal/epics` | 214/214 | 187/214 |
| `internal/health` | 48/50 | 47/50 |
| `internal/help` | 6/6 | 6/6 |
| `internal/navigation` | 25/40 | 18/40 |
| `internal/projects` | 217/227 | 196/227 |
| `internal/styles` | 0/0 | 0/0 |
| `internal/testcases` | 130/132 | 112/132 |
| `internal/ui` | 8/8 | 8/8 |
| `pkg/client` | 376/394 | 295/394 |
| **Total** | **4184/4763** | **3642/4763** |

The final CLI rebuild still requires the documented static-baseline repairs
and coverage thresholds. The caller owns publication and the dev merge.

## P7 review fixes 07 (2026-09-28)

The selected-project catalog refresh started by a configuration update or
read-only retry now owns a cancelable request. Leaving configuration management
or quitting cancels it and invalidates its generation, so a late result cannot
change another screen's status or error. A unit regression holds the request
open, observes cancellation on return and quit, and injects a stale result.
The complete-program CRUD scenario now checks every insert and update field,
including ordered and explicit empty arrays and the fixed slug. It attempts a
slug change through keyboard navigation and verifies no update is sent.
No live backend or database was used; this child made no commit or push.

| Command on `49ecc9efd9dd25b820963e1e67f8ec160f08af78` plus review diff | Exit | Evidence |
| --- | ---: | --- |
| Targeted `go test -count=1 ./internal/app -run '^TestP703ConfigurationExitCancelsCatalogRefreshAndIgnoresLateResult$'` and `go test -count=1 ./integration -run '^TestCLIProgramConfigurationCRUDAndCatalogRefresh$' -v` | 0 each | Cancellation and exact keyboard payload regressions pass. |
| `make -C cli check` | 2 | Vet, unit race, architecture and tooling pass. Existing format failures remain in `cmd/mch/main.go` and `internal/app/clipboard.go`; package-comment lint remains in untouched packages, with nondeterministic reporting order. No touched-file static finding remains. |
| `make -C cli coverage` | 2 | Complete unit campaign **4176/4755 (87.8233%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete-program and startup suite passes. |
| `make -C cli integration-coverage` | 2 | Complete program plus real PTY campaign **3636/4755 (76.4669%)**; strict >90% gate fails. |
| `python3 -B -m unittest discover -s scripts -p documentation_test.py -v` (from `cli/`) | 0 | Ledger links, routes and documented commands pass after the documentation update. |
| `git diff --check` | 0 | No whitespace errors. |

Both coverage campaigns report `complete: true`; the terminal campaign ran all
28 program scenarios and its real PTY scenario without skips or failures. Unit
leaves 579 statements and terminal leaves 1119. The largest gaps are
`internal/app` (374 unit, 646 terminal) and `internal/changes` (137 unit,
203 terminal). The raw profiles, exact source SHA256 inventory, uncovered
statements/functions, scenario results, command journals and package totals
are in `cli/.coverage/{unit,integration}`. The covered child binary SHA256 is
`4649fa61c04f718dd685e9471201a1d3e2f7ce3de49a8d04bbc68a7b8d4d0fed`.
Toolchain: Go `go1.26.8-X:nodwarf5`, golangci-lint `2.13.1`, govulncheck
`v1.7.0`, socat `1.8.1.1`. Documentation updates followed measurement; the
measured Go source did not change.

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cmd/mch` | 0/3 | 1/3 |
| `internal/app` | 1982/2356 | 1710/2356 |
| `internal/changes` | 732/869 | 666/869 |
| `internal/configurations` | 170/183 | 167/183 |
| `internal/documents` | 268/273 | 223/273 |
| `internal/dto` | 0/0 | 0/0 |
| `internal/epics` | 214/214 | 187/214 |
| `internal/health` | 48/50 | 47/50 |
| `internal/help` | 6/6 | 6/6 |
| `internal/navigation` | 25/40 | 18/40 |
| `internal/projects` | 217/227 | 196/227 |
| `internal/styles` | 0/0 | 0/0 |
| `internal/testcases` | 130/132 | 112/132 |
| `internal/ui` | 8/8 | 8/8 |
| `pkg/client` | 376/394 | 295/394 |
| **Total** | **4176/4755** | **3636/4755** |

Final CLI rebuild completion still requires the documented static baseline
repairs and coverage thresholds. The caller owns publication and intermediate
dev merge.

## P7 review fixes 05 (2026-09-28)

Configuration form and details scrolling now clamp the stored offset to the
last rendered page, using the same wrapped lines and viewport dimensions as
the view. Repeated PageDown at the bottom no longer delays PageUp. Feature and
app regression tests cover long values, both screens, short content and a
previously oversized offset. No live backend or database was used; this child
made no commit or push.

| Command on `f3d5b38a16082edee24d490df34e9c23a8edbc00` plus review diff | Exit | Evidence |
| --- | ---: | --- |
| Targeted `go test ./internal/configurations ./internal/app -run 'TestP702ConfigScrollClampsStoredOffsetToLastPage\|TestP702ConfigurationPageUpRespondsAfterRepeatedPageDown\|TestP702ConfigurationFormKeepsActiveFieldVisibleAndPages' -count=1` | 0 | All targeted regressions pass. |
| `make -C cli check` | 2 | Vet, unit race, architecture and tooling pass; pre-existing format failures in `cmd/mch/main.go`, `internal/app/clipboard.go`, and package-comment lint in `internal/help/commands.go`, `internal/styles/styles.go`, `internal/ui/layout.go`. |
| `make -C cli coverage` | 2 | Complete unit campaign **4136/4726 (87.5159%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete-program and startup suite passes. |
| `make -C cli integration-coverage` | 2 | Complete program plus real PTY campaign **3605/4726 (76.2802%)**; strict >90% gate fails. |
| `git diff --check` | 0 | No whitespace errors. |

Both coverage status files report `complete: true`. All 28 selected program
scenarios and the PTY scenario completed without skips or failures. Unit leaves
590 statements and terminal leaves 1121. The largest gaps are `internal/app`
(385 unit, 647 terminal) and `internal/changes` (137 unit, 203 terminal).
Raw profiles, per-source SHA256 hashes, source revision, exact uncovered ranges
and functions, scenario results and command journals are in
`cli/.coverage/{unit,integration}`. The covered child binary SHA256 is
`30b5aac0ece62507030e5c8f5a6560c5ae6881feef999e509ea385b66e3da00c`.
Toolchain: `go1.26.8-X:nodwarf5`. Documentation updates followed measurement;
the measured Go source did not change.

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cmd/mch` | 0/3 | 1/3 |
| `internal/app` | 1942/2327 | 1680/2327 |
| `internal/changes` | 732/869 | 666/869 |
| `internal/configurations` | 170/183 | 166/183 |
| `internal/documents` | 268/273 | 223/273 |
| `internal/dto` | 0/0 | 0/0 |
| `internal/epics` | 214/214 | 187/214 |
| `internal/health` | 48/50 | 47/50 |
| `internal/help` | 6/6 | 6/6 |
| `internal/navigation` | 25/40 | 18/40 |
| `internal/projects` | 217/227 | 196/227 |
| `internal/styles` | 0/0 | 0/0 |
| `internal/testcases` | 130/132 | 112/132 |
| `internal/ui` | 8/8 | 8/8 |
| `pkg/client` | 376/394 | 295/394 |
| **Total** | **4136/4726** | **3605/4726** |

Final CLI rebuild completion still requires the documented static baseline
repairs and coverage thresholds. The caller owns publication and intermediate
dev merge.

## P7 review fixes 04 (2026-09-28)

Configuration form navigation now scrolls to the active field after Tab,
Shift+Tab, form entry and terminal resize. PageUp/PageDown scroll the form.
The list adapter preserves the backend's database-collation order and rejects
duplicate slugs without imposing Go string order. Fake-server and form
regressions cover both findings. The P7 spec and assertion ledger reflect the
backend's actual ordering contract. No live backend or database was used, and
this child made no commit or push.

| Command on `85896495bcd7dae345ae625dcd4ad5a830d6bcef` plus review diff | Exit | Evidence |
| --- | ---: | --- |
| Targeted `go test ./internal/app ./internal/configurations ./pkg/client -run 'TestP702ConfigurationFormKeepsActiveFieldVisibleAndPages\|TestP701SixArraysExplicitEmptyAndMalformedResponses' -count=1` | 0 | Both regressions pass; feature package had no matching named test. |
| `make -C cli check` | 2 | Vet, unit race, architecture and tooling pass. Existing format baseline: `cmd/mch/main.go`, `internal/app/clipboard.go`; package-comment lint: `internal/help/commands.go`, `internal/styles/styles.go`, `internal/ui/layout.go`. |
| `make -C cli coverage` | 2 | Complete unit campaign **4124/4718 (87.4099%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete-program and startup suite passes. |
| `make -C cli integration-coverage` | 2 | Complete program plus real PTY campaign **3595/4718 (76.1975%)**; strict >90% gate fails. |
| `git diff --check` | 0 | No whitespace errors. |

Both coverage status files report `complete: true`; the 28 selected program
scenarios and one PTY scenario completed. Unit leaves 594 statements and
terminal leaves 1123. The largest gaps are `internal/app` (389 unit, 648
terminal) and `internal/changes` (137 unit, 203 terminal). Raw profiles,
per-source SHA256 hashes, scenario results, exact uncovered ranges/functions,
package totals and command journals are in `cli/.coverage/{unit,integration}`.
The covered child binary SHA256 is
`494a14ae5bd2439bf6c84cdaefb0f343ea92b886de4233fbdecc9c165c459c91`.
Toolchain: `go1.26.8-X:nodwarf5`. Documentation updates followed measurement;
the measured Go source did not change.

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cmd/mch` | 0/3 | 1/3 |
| `internal/app` | 1936/2325 | 1677/2325 |
| `internal/changes` | 732/869 | 666/869 |
| `internal/configurations` | 164/177 | 159/177 |
| `internal/documents` | 268/273 | 223/273 |
| `internal/dto` | 0/0 | 0/0 |
| `internal/epics` | 214/214 | 187/214 |
| `internal/health` | 48/50 | 47/50 |
| `internal/help` | 6/6 | 6/6 |
| `internal/navigation` | 25/40 | 18/40 |
| `internal/projects` | 217/227 | 196/227 |
| `internal/styles` | 0/0 | 0/0 |
| `internal/testcases` | 130/132 | 112/132 |
| `internal/ui` | 8/8 | 8/8 |
| `pkg/client` | 376/394 | 295/394 |
| **Total** | **4124/4718** | **3595/4718** |

Final CLI rebuild completion still requires the documented static baseline
repairs and coverage thresholds. The caller owns publication and intermediate
dev merge.

## P7 review fixes 02 (2026-09-28)

Re-entering backend configurations after canceling an in-flight save or delete
now opens the list rather than the old editor or confirmation. When the selected
project's slug is still loading, a committed update conservatively refreshes its
resolved catalog and invalidates the pending startup catalog result. The create
form accepts `/` as the first slug character; Ctrl+G opens its command menu.
Three new unit tests cover these cases, and the shared command-menu test now uses
the form's shortcut. The [ledger](cli-contracts.md) maps the assertions. No live
backend or database was used, and this child made no commit or push.

| Command on `d1d3ce65fa2f4538b55a0ea647ffead6562a5003` plus review diff | Exit | Evidence |
| --- | ---: | --- |
| Targeted `go test ./internal/app -run 'TestP703UpdateRefreshesCatalogWhileProjectIdentityLoads\|TestP702ReentryClearsCanceledConfigurationModals\|TestP702CreateSlugAcceptsLeadingSlashAndKeepsCommands' -count=1` | 0 | All three regressions pass. |
| `go test ./internal/app -count=1` | 0 | All app tests pass after the shared test update. |
| `make -C cli check` | 2 | Vet, race, architecture and tooling pass. Existing format baseline: `cmd/mch/main.go`, `internal/app/clipboard.go`; package-comment lint: `internal/help/commands.go`, `internal/styles/styles.go`, `internal/ui/layout.go`. |
| `make -C cli coverage` | 2 | Complete unit campaign **4050/4657 (86.9659%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete-program and startup suite passes. |
| `make -C cli integration-coverage` | 2 | Complete program plus real PTY campaign **3554/4657 (76.3152%)**; strict >90% gate fails. |

Both campaign status files report `complete: true`; all 28 selected program
scenarios and the one PTY scenario finished without a skip or failed assertion.
Unit leaves 607 statements and terminal leaves 1103. The largest gaps remain
`internal/app` (402 unit, 636 terminal) and `internal/changes` (137 unit,
203 terminal). The 69 production Go inputs have sorted path/hash SHA256
`e64b4980394b081cdd49869ab8d452bf4df8eb866295bfbfffbfe981229f35ed`;
raw profiles, individual source hashes, exact uncovered ranges and functions,
package totals, scenario results and command journals are in
`cli/.coverage/{unit,integration}`. The covered child binary SHA256 is
`a4072fb2f183a513d452c0e009985dfc0c2c71a326527199e7c87d75ed646d54`.
Toolchain: `go1.26.8-X:nodwarf5`. Documentation updates followed measurement;
the measured Go source did not change.

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cmd/mch` | 0/3 | 1/3 |
| `internal/app` | 1891/2293 | 1657/2293 |
| `internal/changes` | 732/869 | 666/869 |
| `internal/configurations` | 135/148 | 138/148 |
| `internal/documents` | 268/273 | 223/273 |
| `internal/dto` | 0/0 | 0/0 |
| `internal/epics` | 214/214 | 187/214 |
| `internal/health` | 48/50 | 47/50 |
| `internal/help` | 6/6 | 6/6 |
| `internal/navigation` | 25/40 | 18/40 |
| `internal/projects` | 217/227 | 196/227 |
| `internal/styles` | 0/0 | 0/0 |
| `internal/testcases` | 130/132 | 112/132 |
| `internal/ui` | 8/8 | 8/8 |
| `pkg/client` | 376/394 | 295/394 |
| **Total** | **4050/4657** | **3554/4657** |

Final CLI rebuild completion still requires the documented static baseline
repairs and coverage thresholds. The caller owns publication and intermediate
dev merge.

## P7 review fixes 01 (2026-09-28)

Separate `/`, arrow and Enter key events now reach the command dropdown on
backend configuration and health screens. The configuration form supports
Shift+Tab to move backward and wraps forward/backward among editable fields;
editing never selects the immutable slug. Field switches retain the current
value, so a rejected draft can be corrected without reopening it. Regression
tests exercise configuration edit/delete, both health routes and correction of
an earlier invalid array. The existing complete-program scenarios now send
separate dropdown keystrokes for configuration edit and both health routes.
The P7 ledger maps these assertions. No live
backend or database was used, and this child made no commit or push.

| Command on `6e77505619dc83c1b48d0a75e032910ade57793f` plus review diff | Exit | Evidence |
| --- | ---: | --- |
| Targeted `go test ./internal/app -run 'TestP702ConfigurationDropdownKeysReachActions\|TestP704HealthDropdownKeysSelectBothRoutes\|TestP702ConfigurationDraftCanRevisitEarlierFields' -count=1` | 0 | All three regressions pass. |
| `make -C cli check` | 2 | Vet, race, architecture and tooling pass. Existing format baseline: `cmd/mch/main.go`, `internal/app/clipboard.go`; package-comment lint: `internal/help/commands.go`, `internal/styles/styles.go`, `internal/ui/layout.go`. |
| `make -C cli coverage` | 2 | Complete unit campaign **4034/4653 (86.6968%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete program/startup suite passes. |
| `make -C cli integration-coverage` | 2 | Complete program plus real PTY campaign **3550/4653 (76.2949%)**; strict >90% gate fails. |
| `git diff --check` | 0 | No whitespace errors. |

Both coverage status files report `complete: true`; no required scenario failed
or skipped. The nonzero coverage exits are numerical gate failures. Unit leaves
619 statements and terminal leaves 1103. The largest gaps are `internal/app`
(407 unit, 636 terminal) and `internal/changes` (137 unit, 203 terminal).
The 126 measured inputs share sorted path/hash SHA256
`099f4a1e2f8f9984984fe9b208ae90b7570b23b7d13442b525276d7ff6751a89`;
the covered child binary SHA256 is
`fa93cf9b76731bdc75c821e951df4cab2420fd196652497d37c28208353ccdbd`.
Toolchain: `go1.26.8-X:nodwarf5`. Raw profiles, source hashes, exact uncovered
ranges, package totals, scenario results and command journals are in
`cli/.coverage/{unit,integration}`. This checkpoint, ledger and implementation
log were updated after the measured source and tests.

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cmd/mch` | 0/3 | 1/3 |
| `internal/app` | 1882/2289 | 1653/2289 |
| `internal/changes` | 732/869 | 666/869 |
| `internal/configurations` | 128/148 | 138/148 |
| `internal/documents` | 268/273 | 223/273 |
| `internal/dto` | 0/0 | 0/0 |
| `internal/epics` | 214/214 | 187/214 |
| `internal/health` | 48/50 | 47/50 |
| `internal/help` | 6/6 | 6/6 |
| `internal/navigation` | 25/40 | 18/40 |
| `internal/projects` | 217/227 | 196/227 |
| `internal/styles` | 0/0 | 0/0 |
| `internal/testcases` | 130/132 | 112/132 |
| `internal/ui` | 8/8 | 8/8 |
| `pkg/client` | 376/394 | 295/394 |
| **Total** | **4034/4653** | **3550/4653** |

Final CLI rebuild completion remains blocked by the documented static baseline
and coverage thresholds. The caller owns publication and intermediate dev merge.

## P7 implementation awaiting review (2026-09-28)

[Spec 027](../../agent/specs/027-cli-configuration-health.md) is implemented on
`change/027-cli-configuration-health` from P6 dev
`5aba38f60c3a3fe364c7921029609be2cd1d5668`. The implementation checkout
is at `10da1a82e5e2a8868247d2b55ca8ba66ba4d36bb` plus its uncommitted P7
changes. The caller owns the implementation commit/push, native review with
`--base origin/dev`, fixes and dev merge; this child made no Git publication.

The backend configuration screen provides keyboard list, exact-slug details,
create, complete-array update and confirmed delete independently of local
`/config` and project `/project-config`. It retains committed write feedback
through separate read-only refreshes, blocks stale rows, refreshes the selected
project's catalog after update, and retries catalog read failures. Health
checks both GET routes, presents healthy/degraded status and preserves the last
valid result across a failed refresh. State and presentation are owned by
`internal/configurations` and `internal/health`; the shell composes navigation.
The [ledger](cli-contracts.md) maps P7-01–P7-06 to named unit and terminal
assertions. No live backend or database was used. The complete-program suite
uses fake HTTP servers and the PTY child uses an owned process and fake server.

| Command from repository root on final production/test inputs | Exit | Evidence |
| --- | ---: | --- |
| `make -C cli check` | 2 | Vet, unit race, architecture and tooling pass. Only untouched formatting baseline in `cmd/mch/main.go` and `internal/app/clipboard.go`; sampled package-comment lint in `cmd/mch/main.go`, `internal/help/commands.go`, `internal/styles/styles.go` and `internal/ui/layout.go`. No new static/test regression. |
| `make -C cli coverage` | 2 | Complete unit campaign **4000/4646 (86.0956%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete program/startup suite passes. |
| `make -C cli integration-coverage` | 2 | Complete 28-program plus real PTY campaign **3539/4646 (76.1731%)**; strict >90% gate fails. |
| `git diff --check` | 0 | No whitespace errors. |

Both `cli/.coverage/{unit,integration}/status.json` files report
`complete: true`; no required scenario failed or skipped. The nonzero coverage
exits are numerical gate failures, not incomplete profiles. All 126 measured
input paths in each provenance manifest share a sorted JSON path/hash SHA256
digest `a54171d3913ca2e98f080d5354286bf93e325103610704344d4f69f29595d8cf`.
The toolchain is `go1.26.8-X:nodwarf5`; covered child binary SHA256 is
`5d4c0314b391861e3aeece3a808dffc5fc020e0c2be6e8db095521edf0191c68`.
Raw profiles, exact uncovered statements/functions, package inventory, scenario
results and command journals are under `cli/.coverage/{unit,integration}`.
This checkpoint, plan and implementation log are documentation updates after
the measured source and tests.

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cmd/mch` | 0/3 | 1/3 |
| `internal/app` | 1848/2282 | 1642/2282 |
| `internal/changes` | 732/869 | 666/869 |
| `internal/configurations` | 128/148 | 138/148 |
| `internal/documents` | 268/273 | 223/273 |
| `internal/dto` | 0/0 | 0/0 |
| `internal/epics` | 214/214 | 187/214 |
| `internal/health` | 48/50 | 47/50 |
| `internal/help` | 6/6 | 6/6 |
| `internal/navigation` | 25/40 | 18/40 |
| `internal/projects` | 217/227 | 196/227 |
| `internal/styles` | 0/0 | 0/0 |
| `internal/testcases` | 130/132 | 112/132 |
| `internal/ui` | 8/8 | 8/8 |
| `pkg/client` | 376/394 | 295/394 |
| **Total** | **4000/4646** | **3539/4646** |

Unit leaves 646 statements and terminal leaves 1107. The largest remaining
gaps are `internal/app` (434 unit, 640 terminal), `internal/changes` (137 unit,
203 terminal), and terminal `pkg/client` (99); exact ranges are in the campaign
uncovered reports. These are final-rebuild targets, not a reason to hide a gate
failure in this intermediate pass. P8 brief clarification follows caller-owned
P7 review and dev merge. No stage or production promotion is authorized.

## P7 review fixes 03 (2026-09-28)

Review of [spec 027](../../agent/specs/027-cli-configuration-health.md) found
three valid issues. Configuration editor bytes now survive textarea preview,
field navigation and save, so a slug containing an embedded tab is sent exactly.
Returning from details clears the detail retry; `/retry` on the list only reads
the list. List selection remains visible across arrow/page navigation, refresh
and return, even when an earlier slug wraps; details open at their heading.
Three named regression tests are mapped in the [contract ledger](cli-contracts.md).
No backend, database, live server or Git publication was used in this child.

| Command on final production/test inputs | Exit | Result |
| --- | ---: | --- |
| Targeted app/configuration regression tests | 0 | All three named tests pass. |
| `make -C cli check` | 2 | Vet, unit race, architecture and tooling pass; only the existing format baseline in `cmd/mch/main.go`, `internal/app/clipboard.go` and package-comment lint in `internal/help/commands.go`, `internal/styles/styles.go`, `internal/ui/layout.go` fail. |
| `make -C cli coverage` | 2 | Complete unit campaign **4088/4689 (87.1828%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete program/startup suite passes. |
| `make -C cli integration-coverage` | 2 | Complete 28-program plus real PTY campaign **3576/4689 (76.2636%)**; strict >90% gate fails. |
| `git diff --check` | 0 | No whitespace errors. |

Both `.coverage/{unit,integration}/status.json` files report `complete: true`;
all required scenarios and counters finished without test failure or skip. The
126 measured input path/hash pairs match across campaigns; their sorted JSON
SHA256 is `4c9ea20948acfe36c16e786d713848fdca16ba13e9b75beacad011d5bf7c677c`.
Measured checkout revision is `d3f51ae9367265b451d8d04aa2d6e9802deaa191`
plus the uncommitted review diff. Go is `go1.26.8-X:nodwarf5`; the covered PTY
child SHA256 is `f271393b3d040a22fe2073f8243cb51bfc10022a7b5981142c184ed38eda0006`.
Raw profiles, exact uncovered statements/functions, package inventory, scenario
results and command journals are under `cli/.coverage/{unit,integration}`.

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cmd/mch` | 0/3 | 1/3 |
| `internal/app` | 1917/2312 | 1670/2312 |
| `internal/changes` | 732/869 | 666/869 |
| `internal/configurations` | 147/161 | 147/161 |
| `internal/documents` | 268/273 | 223/273 |
| `internal/dto` | 0/0 | 0/0 |
| `internal/epics` | 214/214 | 187/214 |
| `internal/health` | 48/50 | 47/50 |
| `internal/help` | 6/6 | 6/6 |
| `internal/navigation` | 25/40 | 18/40 |
| `internal/projects` | 217/227 | 196/227 |
| `internal/styles` | 0/0 | 0/0 |
| `internal/testcases` | 130/132 | 112/132 |
| `internal/ui` | 8/8 | 8/8 |
| `pkg/client` | 376/394 | 295/394 |
| **Total** | **4088/4689** | **3576/4689** |

Final CLI coverage targets remain unmet: 601 unit and 1113 terminal statements
are uncovered. The largest package gaps are `internal/app` (395 unit, 642
terminal) and `internal/changes` (137 unit, 203 terminal); terminal `pkg/client`
has 99 uncovered statements. The existing static baseline and these strict
numerical gates remain visible for later passes. The caller owns review-fix
commit/push and any dev merge; P8 follows that sequence. No stage or production
promotion is authorized.

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

## P7 review fixes 06: active configuration field (2026-09-28)

The P7-02 finding was valid: the configuration form's input band always showed
the start of a long array, hiding newly entered text and the cursor. The band
now displays a cursor-following, terminal-safe window of the current input.
`TestP702ConfigurationInputFollowsCursorInLongArray` covers insertion inside a
long array, Home/End movement, visible redraw, preserved draft bytes and width.
The P7-02 ledger includes this assertion. No live backend or database was used;
the caller owns commit/push and dev merge.

| Command on review-fix source | Exit | Result |
| --- | ---: | --- |
| Targeted P7-02 app tests | 0 | All pass. |
| `make -C cli check` | 2 | Vet, race, architecture and tooling pass. Only the established format baseline in `cmd/mch/main.go` and `internal/app/clipboard.go`, and package-comment lint in `internal/help/commands.go`, `internal/styles/styles.go`, `internal/ui/layout.go` fail. Touched Go files have no formatting diff. |
| `make -C cli coverage` | 2 | Complete unit campaign **4161/4742 (87.7478%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Program/startup suite passes. |
| `make -C cli integration-coverage` | 2 | Complete 28-program plus real PTY campaign **3618/4742 (76.2969%)**; strict >90% gate fails. |
| `git diff --check` | 0 | No whitespace errors. |

Both campaign status files report `complete: true`; all required scenarios
finished. The numerical gates account for both nonzero coverage exits. Both
126-input provenance maps have sorted compact JSON SHA256
`5066c5e43230ff635f849e58b13bf1152b8b6ba60645d46f172846528c5faf31`.
The measured checkout revision is `6cd85af406f43497c39cddf73f4d7a0f75419bff`
plus the uncommitted review diff. Go is `go1.26.8-X:nodwarf5`; the covered
child binary SHA256 is
`1131dbed1e3e6754fb936df16b6d3e75b3b461415985fa008f97b28af346fa7a`.
Raw profiles, package totals, uncovered statements/functions, scenarios and
command journals are in `.coverage/{unit,integration}`. This checkpoint and
implementation log are documentation edits after the measured source and tests.

## P8 review fixes 01: brief input and refresh retry (2026-09-28)

All three findings were valid. The brief command dropdown now owns keys while
open; direct brief typing has no character cap; and a successful refresh returns
to draft when its committed revision differs from the current human draft.
`TestBriefCommandDropdownRoutesSelectionAndEscape`,
`TestBriefDirectInputPreservesMoreThanDefaultLimit`, and
`TestP805RefreshRetryKeepsUnsavedEditInDraft` cover the corrections. The P8
assertion ledger maps these tests. No live backend or database was used.

| Command on review-fix source | Exit | Result |
| --- | ---: | --- |
| Focused app and agent tests | 0 | All three new regressions and the existing committed-refresh regression pass. |
| `make -C cli check` | 2 | Vet, race, architecture and tooling pass. The established formatting baseline in `cmd/mch/main.go` and `internal/app/clipboard.go`, plus package-comment lint in `internal/help/commands.go`, `internal/styles/styles.go` and `internal/ui/layout.go`, still fail. No touched file is implicated. |
| `make -C cli coverage` | 2 | Complete unit campaign **4630/5423 (85.3771%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete program/startup suite passes. |
| `make -C cli integration-coverage` | 2 | Complete 31-program plus real PTY campaign **4038/5423 (74.4606%)**; strict >90% gate fails. |
| `git diff --check` | 0 | No whitespace errors. |

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cli/cmd/mch` | 0/3 | 1/3 |
| `cli/internal/agent` | 250/308 | 230/308 |
| `cli/internal/app` | 2082/2583 | 1881/2583 |
| `cli/internal/changes` | 732/869 | 666/869 |
| `cli/internal/configurations` | 170/183 | 167/183 |
| `cli/internal/documents` | 268/273 | 224/273 |
| `cli/internal/dto` | 0/0 | 0/0 |
| `cli/internal/epics` | 214/214 | 187/214 |
| `cli/internal/health` | 48/50 | 47/50 |
| `cli/internal/help` | 6/6 | 6/6 |
| `cli/internal/navigation` | 25/40 | 18/40 |
| `cli/internal/projects` | 217/227 | 196/227 |
| `cli/internal/styles` | 0/0 | 0/0 |
| `cli/internal/testcases` | 130/132 | 112/132 |
| `cli/internal/ui` | 8/8 | 8/8 |
| `cli/pkg/briefprocess` | 104/133 | 0/133 |
| `cli/pkg/client` | 376/394 | 295/394 |

Both campaign status files report `complete: true`; all selected scenarios
passed without skips. Raw profiles, exact uncovered statements/functions,
scenario records and command journals remain under `.coverage/{unit,integration}`.
Unit has 793 uncovered production statements; terminal has 1385. The measured
checkout revision is `51bc5f24733a6016fded6e1929b34834f9b65e19` plus the
uncommitted review diff. Both 134-input provenance maps have sorted compact JSON
SHA256 `3bb19f799ae75813f17278556be0060cdc433ac117270fa55f87ca91627cd8bc`.
The covered child binary SHA256 is
`a6cfd6b5b67918635dda745f1cc07e52d5b4418360acd6485869a37e44a29414`.
The P8 ledger, checkpoint and implementation log were edited after campaign
measurement; measured production and test source hashes remain current. The
caller owns all commits, pushes and dev merging.

## P8 review fixes 02: committed brief confirmation (2026-09-28)

The finding was valid. `/confirm` now requests a read-only refresh while a
committed change or brief ID awaits verification. The controller also converts
any `Write` request in that state to `Refresh`, so another caller cannot append
the same brief. The program scenarios confirm again after failed reads following
both new-change creation and existing-change insert; they assert one write for
each. Controller and shell tests cover repeated failed reads and draft
preservation. Fake HTTP and agent
collaborators were used; no live backend or database was accessed.

| Command on review-fix source | Exit | Result |
| --- | ---: | --- |
| Focused agent, app and program tests | 0 | Committed-refresh and existing-insert confirmation regressions pass. |
| `make -C cli check` | 2 | Vet, race, architecture and tooling pass. Existing format failures remain in `cmd/mch/main.go` and `internal/app/clipboard.go`; existing package-comment lint failures remain in `internal/help/commands.go`, `internal/styles/styles.go` and `internal/ui/layout.go`. |
| `make -C cli coverage` | 2 | Complete unit campaign **4635/5427 (85.4063%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete program/startup suite passes. |
| `make -C cli integration-coverage` | 2 | Complete 31-program plus real PTY campaign **4041/5427 (74.4610%)**; strict >90% gate fails. |
| `git diff --check` | 0 | No whitespace errors. |

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cli/cmd/mch` | 0/3 | 1/3 |
| `cli/internal/agent` | 252/310 | 231/310 |
| `cli/internal/app` | 2085/2585 | 1883/2585 |
| `cli/internal/changes` | 732/869 | 666/869 |
| `cli/internal/configurations` | 170/183 | 167/183 |
| `cli/internal/documents` | 268/273 | 224/273 |
| `cli/internal/dto` | 0/0 | 0/0 |
| `cli/internal/epics` | 214/214 | 187/214 |
| `cli/internal/health` | 48/50 | 47/50 |
| `cli/internal/help` | 6/6 | 6/6 |
| `cli/internal/navigation` | 25/40 | 18/40 |
| `cli/internal/projects` | 217/227 | 196/227 |
| `cli/internal/styles` | 0/0 | 0/0 |
| `cli/internal/testcases` | 130/132 | 112/132 |
| `cli/internal/ui` | 8/8 | 8/8 |
| `cli/pkg/briefprocess` | 104/133 | 0/133 |
| `cli/pkg/client` | 376/394 | 295/394 |

Both campaign status files report `complete: true`; all selected scenarios
passed without skips. Raw profiles, exact uncovered statements and functions,
scenario records and command journals remain under `.coverage/{unit,integration}`.
Unit has 792 uncovered statements; terminal has 1386. The measured checkout
revision is `16ce6d15c53c166f085867db34edd1c245798e7e` plus the uncommitted
review diff. Both 134-input provenance maps have sorted compact JSON SHA256
`c978d6c04043992be96b1546c42856439c03550f1edaa820396bc3aab90df07b`.
The covered child binary SHA256 is
`d878833013fd37e03327d7d817146811a2a842369175099ec47c4b2de8494ff9`.
Go is `go1.26.8-X:nodwarf5`; golangci-lint is 2.13.1 and govulncheck is 1.7.0.
This checkpoint, ledger and implementation log are documentation edits after
campaign measurement; measured production and test source hashes remain current.
The caller owns commits, pushes and dev merging.

## P8 review fixes 03: brief conflicts and complete-program rounds (2026-09-28)

All five findings were valid. Post-create refresh now rejects a current brief
whose body differs from the created brief, retaining the committed change ID for
read-only retry. Approving an unchanged rewrite now rechecks the selected
project's `brief`/`spec` catalog, change owner and current brief before advancing.
Non-ready output without an unresolved blocker is a recoverable runner error.
The program campaign shows two separate answer/resolve rounds with a visible
follow-up blocker and a failed create with a retained draft, visible error,
successful keyboard retry and exact create/read call sequence. Controller tests
cover the new conflict, verification and protocol paths; the P8 ledger maps
the evidence. Fake backend and runner collaborators were used; no live backend
or database was accessed.

| Command on review-fix source | Exit | Result |
| --- | ---: | --- |
| Focused agent and brief program tests | 0 | All targeted P8 controller and keyboard scenarios pass. |
| `make -C cli check` | 2 | Vet, race, architecture and tooling pass. Existing format failures remain in `cmd/mch/main.go` and `internal/app/clipboard.go`; existing package-comment lint failures remain in `internal/help/commands.go`, `internal/styles/styles.go` and `internal/ui/layout.go`. |
| `make -C cli coverage` | 2 | Complete unit campaign **4646/5434 (85.4987%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete program/startup suite passes. |
| `make -C cli integration-coverage` | 2 | Complete 31-program plus real PTY campaign **4049/5434 (74.5123%)**; strict >90% gate fails. |
| `git diff --check` | 0 | No whitespace errors. |

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cli/cmd/mch` | 0/3 | 1/3 |
| `cli/internal/agent` | 263/317 | 239/317 |
| `cli/internal/app` | 2085/2585 | 1883/2585 |
| `cli/internal/changes` | 732/869 | 666/869 |
| `cli/internal/configurations` | 170/183 | 167/183 |
| `cli/internal/documents` | 268/273 | 224/273 |
| `cli/internal/dto` | 0/0 | 0/0 |
| `cli/internal/epics` | 214/214 | 187/214 |
| `cli/internal/health` | 48/50 | 47/50 |
| `cli/internal/help` | 6/6 | 6/6 |
| `cli/internal/navigation` | 25/40 | 18/40 |
| `cli/internal/projects` | 217/227 | 196/227 |
| `cli/internal/styles` | 0/0 | 0/0 |
| `cli/internal/testcases` | 130/132 | 112/132 |
| `cli/internal/ui` | 8/8 | 8/8 |
| `cli/pkg/briefprocess` | 104/133 | 0/133 |
| `cli/pkg/client` | 376/394 | 295/394 |

Both coverage status files report `complete: true`; all selected scenarios
passed without skips. The 134-input provenance maps have sorted compact JSON
SHA256 `36e27ca5d47309fa74c96a1bfc426def71453d0c92f1c741808d490047bfbf34`.
The measured checkout revision is `a8c9ea89bf6a1ec2a169bfe952261a90bb60d0ca`
plus the uncommitted review diff. The covered child binary SHA256 is
`311c1c3cf8382cf063481be3f260109695671cac394ccf197b956d760cc4b9a9`.
Raw profiles, exact uncovered statements/functions, source hashes, scenarios and
command journals remain under `cli/.coverage/{unit,integration}`. Unit has 788
uncovered production statements; terminal has 1385. Go is
`go1.26.8-X:nodwarf5`; golangci-lint is 2.13.1 and govulncheck is 1.7.0.
This checkpoint and implementation log are documentation edits after campaign
measurement; measured production and test source hashes remain current. The
caller owns commits, pushes and dev merging.

## P8 review fixes 05: brief editor seed and answerable IDs (2026-09-28)

Both findings were valid. Pending brief input now seeds Ctrl+E; a saved draft
seeds it when the input is empty. Agent output rejects question IDs with outer
whitespace, colons or control characters, matching the `ID: answer` input.
The brief prompts state that ID syntax. The P8 ledger names the new regression
tests. Fake collaborators and local processes only; no live backend was used.

| Command on review-fix source | Exit | Result |
| --- | ---: | --- |
| Focused `go test ./internal/app ./internal/agent` | 0 | Both editor paths and invalid/valid question IDs pass. |
| `make -C cli check` | 2 | Vet, race, architecture and tooling pass. Existing format failures: `cmd/mch/main.go`, `internal/app/clipboard.go`; existing package-comment lint: `internal/help/commands.go`, `internal/styles/styles.go`, `internal/ui/layout.go`. |
| `make -C cli coverage` | 2 | Complete unit campaign **4670/5455 (85.6095%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete program/startup suite passes. |
| `make -C cli integration-coverage` | 2 | Complete program and real PTY campaign **4053/5455 (74.2988%)**; strict >90% gate fails. |
| `git diff --check` | 0 | No whitespace errors. |

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cli/cmd/mch` | 0/3 | 1/3 |
| `cli/internal/agent` | 265/319 | 241/319 |
| `cli/internal/app` | 2095/2588 | 1885/2588 |
| `cli/internal/changes` | 732/869 | 666/869 |
| `cli/internal/configurations` | 170/183 | 167/183 |
| `cli/internal/documents` | 268/273 | 224/273 |
| `cli/internal/dto` | 0/0 | 0/0 |
| `cli/internal/epics` | 214/214 | 187/214 |
| `cli/internal/health` | 48/50 | 47/50 |
| `cli/internal/help` | 6/6 | 6/6 |
| `cli/internal/navigation` | 25/40 | 18/40 |
| `cli/internal/projects` | 217/227 | 196/227 |
| `cli/internal/styles` | 0/0 | 0/0 |
| `cli/internal/testcases` | 130/132 | 112/132 |
| `cli/internal/ui` | 8/8 | 8/8 |
| `cli/pkg/briefprocess` | 116/149 | 0/149 |
| `cli/pkg/client` | 376/394 | 295/394 |

Both campaign status files report `complete: true`; selected scenarios passed
without skips. Unit has 785 uncovered statements and terminal has 1402. The
measured checkout revision is `1dfe730a9030fcb1bd22dfefc07905822c256133`
plus this uncommitted review diff. The source hashes for the changed Go files
are recorded in each campaign's `provenance.json`; the covered child binary
SHA256 is `a2ded60b9f412ad03c76b952b80217b66178a306f90b328a839a190b04ae2381`.
Raw profiles, uncovered statements/functions, scenarios and journals remain
under `cli/.coverage/{unit,integration}`. Go is `go1.26.8-X:nodwarf5`;
golangci-lint is 2.13.1 and govulncheck is 1.7.0. Documentation was finalized
after measurement; production and test source hashes remain current. The caller
owns commits, pushes and dev merging.

## P8 review fixes 06: live agent progress (2026-09-28)

The process adapter now publishes bounded stdout/stderr excerpts through a
one-slot progress channel while the child runs. The controller forwards that
channel with the scoped runner request, and the shell turns it into typed
Bubble Tea messages. Only a current, busy clarification shows progress; the
footer escapes terminal controls and late messages are ignored. Cancellation
still reaps the child and leaves no blocked progress sender. Local process,
controller and shell regression tests cover these paths; no live backend or
Codex process was used.

| Command on review-fix source | Exit | Result |
| --- | ---: | --- |
| Focused `go test ./internal/agent ./internal/app ./pkg/briefprocess -count=1` | 0 | Progress, cancellation, channel wiring, UI and retained tests pass. |
| `make -C cli check` | 2 | Vet, race, architecture and tooling pass. Existing format failures remain in `cmd/mch/main.go`, `internal/app/clipboard.go`; existing package-comment lint remains in `internal/help/commands.go`, `internal/styles/styles.go`, `internal/ui/layout.go`. |
| `make -C cli coverage` | 2 | Complete unit campaign **4698/5486 (85.6362%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete program/startup suite passes. |
| `make -C cli integration-coverage` | 2 | All 31 selected program scenarios and the real PTY child pass. Complete terminal campaign **4072/5486 (74.2253%)**; strict >90% gate fails. |

The two campaign status files report `complete: true`. Unit package gaps include
`internal/agent` 268/322, `internal/app` 2115/2611 and `pkg/briefprocess`
121/154; terminal gaps include `internal/agent` 244/322, `internal/app`
1901/2611 and `pkg/briefprocess` 0/154. Exact totals for every production
package, the raw profiles, source hashes, commands and uncovered statements are
under `cli/.coverage/{unit,integration}`. The measured checkout revision is
`af2b2711b59423847428f8655175139e20ad6af0` plus this uncommitted review
diff. The covered child binary SHA256 is
`e1eb486e41642cdbaebaee8835ba77c1cfdaabd7be8b6c7121d9efe7e2e0523f`.
Documentation was updated after measurement; production and test source hashes
remain current. The caller owns commits, pushes and dev merging.

## P8 review fixes 07: structured arrays, owned runner paths and PTY cancellation (2026-09-28)

The controller now rejects nil `questions` and `unresolved` arrays even when an
injected runner returns a ready result. It allocates the operation directory and
passes the original, current input, context, question, answer and output paths to
every runner. The process adapter uses those paths and checks their ownership;
the controller rejects a runner response naming another directory. Controller
tests cover explicit empty arrays, missing arrays, wrong locations and an
unowned scratch path. The program fixture's valid ready result now includes an
explicit empty blocker array. The real PTY child scrolls the brief view, checks
editor redraw, starts a fake agent, cancels it and checks process reaping and
scratch cleanup. Tests use fake HTTP and owned local processes only.

| Command on review-fix source | Exit | Result |
| --- | ---: | --- |
| Focused agent, adapter, program and PTY tests | 0 | New regressions and retained workflow paths pass. |
| `make -C cli check` | 2 | Vet, race, architecture and tooling pass. The existing format failures in `cmd/mch/main.go` and `internal/app/clipboard.go`, and package-comment lint in `internal/help/commands.go`, `internal/styles/styles.go`, `internal/ui/layout.go`, remain. |
| `make -C cli coverage` | 2 | Complete unit campaign **4734/5528 (85.6368%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete program/startup suite passes. |
| `make -C cli integration-coverage` | 2 | All 31 selected program scenarios and the real PTY child pass. Complete terminal campaign **4193/5528 (75.8502%)**; strict >90% gate fails. |

Both campaign status files report `complete: true`. Unit package gaps include
`internal/agent` 297/357, `internal/app` 2115/2611, and `pkg/briefprocess`
128/161; terminal gaps include `internal/agent` 270/357, `internal/app`
1912/2611, and `pkg/briefprocess` 84/161. Exact package totals, source hashes,
scenario manifests, raw profiles and uncovered statements remain under
`cli/.coverage/{unit,integration}`. The measured checkout is `506f63a8f1dad731d7ddfa949252f576e4389b7b`
plus the review diff; the covered child binary SHA256 is
`1fb4f67e2a4f4e07da360e75399b014f1422f1aeda8a3b4dc452783d24fed278`.
Documentation was finalized after measurement; production and test source hashes
remain current. The caller owns further commits and dev merging.

## P8 review fixes 08: re-entry generation and normalized brief no-ops (2026-09-28)

Opening a brief carries forward and advances the previous controller generation,
so a late result from a canceled workflow cannot match a new workflow for the
same project and change. Existing human edits and agent rewrites compare the
backend-normalized brief body before deciding to append a version. Inputs for
actual writes and the user's original draft retain their exact bytes. Focused
regression tests cover same-scope re-entry and both whitespace-only no-ops.

| Command on review-fix source | Exit | Result |
| --- | ---: | --- |
| Focused agent/app regression tests | 0 | Both new tests pass. |
| `make -C cli check` | 2 | Vet, race, architecture and tooling pass. The existing format failures in `cmd/mch/main.go` and `internal/app/clipboard.go`, and package-comment lint in `internal/help/commands.go`, `internal/styles/styles.go`, `internal/ui/layout.go`, remain. |
| `make -C cli coverage` | 2 | Complete unit campaign **4748/5530 (85.8590%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete program/startup suite passes. |
| `make -C cli integration-coverage` | 2 | All 31 selected program scenarios and the real PTY child pass. Complete terminal campaign **4195/5530 (75.8590%)**; strict >90% gate fails. |

Both campaign status files report `complete: true`. Unit package gaps include
`internal/agent` 297/357, `internal/app` 2129/2613 and `pkg/briefprocess`
128/161; terminal gaps include `internal/agent` 270/357, `internal/app`
1914/2613 and `pkg/briefprocess` 84/161. Exact package totals, source hashes,
scenario manifests, raw profiles and uncovered statements are under
`cli/.coverage/{unit,integration}`. The measured checkout is
`1902939038a4edf66466ead9ec48df393df948c3`, and the commands above were
rerun on that clean committed source with the same exits and measured totals.
The covered child binary SHA256 is
`cd1d138ccbae6977cfdfa5bb0cb3a610d44694dd5002636e1118d39223b59091`.
Documentation was finalized after measurement; production and test source hashes
remain current. The caller owns commits, pushes and dev merging.
At the user's request, the subsequent review pass 02 was interrupted after it
began; no result from that pass is claimed. The strict coverage and recorded
static-check shortfalls remain open for the final CLI rebuild.

## Bottom menu visual pass (2026-09-29)

Both the ordinary prompt and the command dropdown now use `InputBackground`
`#454748`, with terminal half-block rows above and below their text row to
approximate the screenshot's half-row padding. Both use `AccentPurple` for the
leading `>`; ordinary typed text uses `Foreground` `#FFFFFF`, while open-menu
prompt text uses `AccentGreen`. The closed prompt is indented one terminal cell,
and its placeholder uses `Gray` `#AFAFAF`. Gray command names omit the leading
slash and have aligned help. The selected row uses `AccentGreen` text and `MenuBackground`
(`#47514A`, sampled from the screenshot); a DarkGray
`(selected/total)` counter appears below the eight-row window when at least ten
options match. Smaller menus show every option and omit the counter. No scroll
arrows are rendered. Selector dropdowns share the new
row and counter layout. All palette hex colors, including three gradient stops
and the two calculated fallbacks, are named CLI constants. `scripts/colors.pl`
was not modified. The retained PTY and program tests wait for visible command
help before continuing menu interaction.

| Command on this uncommitted source | Exit | Result |
| --- | ---: | --- |
| Focused app menu and selector tests | 0 | Layout, descriptions, color tokens, counter threshold, scrolling and selection pass. |
| `make -C cli check` | 2 | Vet, race, architecture and tooling pass. Existing format failures remain in `cmd/mch/main.go` and `internal/app/clipboard.go`; existing package-comment lint remains in `cmd/mch/main.go`, `internal/help/commands.go`, `internal/styles/styles.go` and `internal/ui/layout.go`. |
| `make -C cli coverage` | 2 | Complete unit campaign **4764/5547 (85.8843%)**; strict >95% gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-coverage` | 2 | All selected program and real PTY scenarios pass; complete terminal campaign **4212/5547 (75.9329%)**; strict >90% gate fails. |

Both campaign status files report `complete: true`; fresh profiles and package
totals are under `cli/.coverage/{unit,integration}`. The source is based on
`7be8c1ec821f92733e5f3b93fc454ef8f522005c` with uncommitted visual changes.
No commit, push or promotion was performed.

## Changes list/menu behavior and coverage policy (2026-09-29)

The changes list stays visible while its bottom command menu opens; its viewport
uses the remaining terminal height, and the extra blank row before the prompt
is removed. The Phase column fits `in-progress` without shifting later columns.
Nullable Ref, Slug, Epic and after-change fields render `-` while their internal
values retain their existing write semantics. Backspace/Delete can remove the
command menu's `/`, and Ctrl+C closes any active dropdown. The PTY row-count
assertion now reflects the extra visible table row.

The current coverage policy is at least 80% CLI unit and 70% terminal integration
production statements. Frontend coverage remains to be decided. Backend unit
and API integration policy is at least 95% and 90%, respectively; backend tooling
enforces the API gate only on a complete instrumented APIHydra campaign.

| Command on current uncommitted source | Exit | Result |
| --- | ---: | --- |
| Focused `go test ./internal/app ./internal/changes` | 0 | Screen and menu tests pass. |
| `make -C cli check` | 2 | Vet, race, architecture and tooling pass. Existing formatting failures in `cmd/mch/main.go` and `internal/app/clipboard.go`, and three package-comment lint findings, remain. |
| `make -C cli coverage` | 0 | Complete unit profile **4779/5555 (86.0306%)**; 80% gate passes. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-coverage` | 0 | Complete program and real PTY campaign **4223/5555 (76.0216%)**; 70% gate passes. An earlier run failed on the stale PTY row-count assertion; it was repaired and the full campaign rerun. The PTY now tests Backspace, Delete and Ctrl+C menu closure. |

Both current campaign status files report `complete: true`. Independent raw
profiles, source provenance, exact package totals and uncovered statements are
under `cli/.coverage/{unit,integration}`. No live backend or database was used.

## Literal command prompt input (2026-09-29)

The first `/` opens the command menu and remains the visible prompt prefix.
Later printable characters are shown literally: a second `/` displays `//` and
has no matching commands; one Backspace restores `/` and the command list.
Spaces are also displayed and filtered as typed. Unit tests cover the exact
prompt/filter sequence and `/q` matching `/quit`; the real PTY test covers
`//` and Backspace restoration.

On this final uncommitted source, `make -C cli coverage` exits 0 with
**4788/5563 (86.0687%)** and `make -C cli integration-coverage` exits 0 with
**4237/5563 (76.1639%)**. Both campaigns completed without skips and pass the
80%/70% gates. `make -C cli check` exits 2 only for the previously recorded
formatting and package-comment lint findings; vet, race, architecture and tooling
pass. The earlier dependency audit found no vulnerabilities; dependencies did not
change. Current raw profiles and exact package totals are under
`cli/.coverage/{unit,integration}`.

## Visible cursor in active menus (2026-09-29)

Every active dropdown now renders the ordinary prompt's white block cursor
after its visible prompt text, including the first `/`, filtered command text,
selector text and a long truncated prompt. Focused tests check the cursor's
rendered ANSI style and the prompt width at a narrow terminal size.

`make -C cli check` exits 2 only for the existing format and package-comment
lint findings; vet, race, architecture and tooling pass. `make -C cli deps-audit`
exits 0 with no vulnerabilities. The first `make -C cli coverage` run was
incomplete because the unrelated brief-process cancellation timing test failed
before producing a valid profile; the rerun exits 0 with **4788/5563 (86.0687%)**.
`make -C cli integration-coverage` exits 0 with **4237/5563 (76.1639%)**.
Both final campaigns report complete, independent profiles and pass the 80%/70%
gates. No live backend or database was used.

## 031 CLI document refactor (2026-10-03)

Implemented [031](../../agent/specs/031-cli-docs-refactor.md): current backend
DTO/routes, ordered configured Docs slots and the full Brief editor, independent
comments, captured-ID bottom delete confirmation, owner/type-scoped retained
history with captured real bat colors, same-ID activation/undelete, inactive
changes and active-only epic selection. `/documents` retains all three owners
and starts history at the explicitly selected version. Local timestamps and
committed-write/read-only retry behavior are covered by unit and program tests.
Editor callbacks additionally bind owner, type, project and revision.
The [assertion ledger](cli-contracts.md) maps every numbered requirement.

Fresh baseline at `63ce51493cf69a3d1b1531d9df9bb89d8cbd2ab1` was clean Git source:
`make -C cli check` exited 2 solely for the stale backend-route ledger; formatting,
lint (zero issues), vet, race and architecture passed. Earlier checkpoint format
and package-comment findings were already repaired in this revision.
Baseline unit **4987/5784 (86.2206%)** and terminal **4416/5784 (76.3485%)**
campaigns both exited 0. Baseline command logs are under `/tmp/031-baseline-*.log`.

| Final command | Exit | Evidence |
| --- | ---: | --- |
| `make -C cli check` | 0 | Formatting, zero lint findings, vet, uncached unit race, architecture and all 36 tooling tests pass. `/tmp/031-check-final4.log`. |
| `make -C cli coverage` | 0 | Complete independent unit campaign **5503/6363 (86.4844%)**; strict 80% gate passes. `/tmp/031-unit-final3.log`. |
| `make -C cli deps-audit` | 0 | No vulnerabilities; dependencies unchanged. `/tmp/031-deps-final2.log`. |
| `make -C cli integration-coverage` | 0 | Complete independent terminal campaign **4892/6363 (76.8820%)**; strict 70% gate passes. All 34 program manifest scenarios plus the instrumented real PTY child pass without skips, missing counters or cleanup failures. `/tmp/031-integration-final3.log`. |
| `make -C cli integration-test` | 0 | Retained whole-program, startup and Flow suites pass. `/tmp/031-integration-work4.log`; later final campaign verifies the changed program scenarios after final editor isolation fixes. |
| `make -C cli terminal-test` | 0 | PTY and harness assertions pass. `/tmp/031-pty-work5.log`; the final campaign repeats the PTY only to validate final source and collect independent counters. |
| `go test -count=1 ./internal/doc -run 'Test(CommentAndDeleteAPIContracts\|CommentUndelete\|ActiveSet)'` (backend cwd) | 0 | Fresh unchanged backend endpoint unit assertions. Backend source and APIHydra suites were already implemented/verified before this pass; no new API coverage measurement is claimed here. |

Both `cli/.coverage/{unit,integration}/status.json` report `complete: true` and
`exit: 0`. Each contains independent raw/complete profiles, command exits,
revision plus full worktree diff and file hashes, exact package totals,
`functions.log` and `uncovered.txt`. Tested base revision is the hash above;
the common aggregate production Go source SHA-256 is
`d679f859b2d3947c8285455f8a5521ef63283e600407d99f378b4e999bf0fd16`
(SHA-256 of sorted compact JSON mapping of the 77 production Go input hashes).
The terminal child binary SHA-256 is
`97001e376ab02305588e5028db20a1a4435fa835675cbff8f2845775b4ac3eb8`.
Go is `go1.26.8-X:nodwarf5`, golangci-lint 2.13.1, govulncheck 1.7.0;
exact bat/socat command versions and scenario lists are retained in campaign logs.
Only this evidence/checkpoint, assertion ledger and implementation log were edited
after the final campaigns; production and test source hashes remain unchanged.

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cli/cmd/mch` | 0/3 | 1/3 |
| `cli/internal/agent` | 297/357 | 270/357 |
| `cli/internal/app` | 2486/3029 | 2313/3029 |
| `cli/internal/changes` | 858/989 | 742/989 |
| `cli/internal/configurations` | 170/183 | 167/183 |
| `cli/internal/documents` | 484/509 | 400/509 |
| `cli/internal/dto` | 0/0 | 0/0 |
| `cli/internal/epics` | 221/221 | 191/221 |
| `cli/internal/health` | 48/50 | 47/50 |
| `cli/internal/help` | 6/6 | 6/6 |
| `cli/internal/navigation` | 25/40 | 19/40 |
| `cli/internal/projects` | 224/234 | 200/234 |
| `cli/internal/styles` | 0/0 | 0/0 |
| `cli/internal/testcases` | 130/132 | 112/132 |
| `cli/internal/ui` | 8/8 | 8/8 |
| `cli/pkg/briefprocess` | 128/161 | 84/161 |
| `cli/pkg/client` | 402/424 | 317/424 |
| `cli/pkg/documentprocess` | 16/17 | 15/17 |

The denominator includes the untested main package and structurally empty DTO
and style packages; no unit hits contribute to terminal totals. Remaining gaps
are visible in each campaign's uncovered statements/functions: main error exits,
navigation fallbacks, agent/brief error paths, transport rejection branches,
and some document/history invalid-input/cancellation/printing paths. Both final
aggregate targets pass without excluding those statements.

Intermediate focused runs exposed stale Active/testcase/menu/PTY expectations,
footer line-wrap synchronization, missing refresh feedback on first history
activation, test color-profile leakage and new lint/manifest findings. Those were
repaired; diagnostic logs remain under `/tmp/031-*-work*.log` and
`/tmp/031-check-final.log`. Incomplete/failed runs established no coverage success.
The final two complete campaigns above supersede earlier measurements.
No CLI test used a live backend/database. No backend source/schema, database
lifecycle, Git publication, deployment, stage or production promotion occurred.

Final review additionally repaired retry attribution after switching from comment
mutations to change-field or testcase mutations. Successful old comment feedback
cannot capture a subsequent reload; starting another write clears that old outcome.
`Test031RetryBelongsToTheLatestCommittedOperation` proves committed read-only
recovery without repeating either write. A fixture initially assumed a failed read
without injecting the failure; it was corrected. That incomplete unit campaign's
raw artifacts remain at `/tmp/031-unit-failed-02`, and its diagnostics at
`/tmp/031-unit-final2.log`. The final campaigns above use the corrected fixture.
The final log keeps the skill-required blank line after its implementation block;
`git diff --check` flags that trailing log line only. Production, test and other
documentation diffs have no whitespace findings.

## 031 review fixes 01 (2026-10-03)

All three supplied findings were validated against 031-11/12/14/16/18/19 and
fixed in the app shell without changing HTTP or feature contracts. A committed
comment insert/update closes the editor and clears its exact draft before any
refresh error is returned; failed writes retain the draft for Enter retry.
Returning from inactive changes schedules a fresh active-list read and restores
the original selected ID through the existing filter/selection machinery. This
also restarts an active read canceled by entering inactive mode. Returning from
owner document history makes the selected list row visible after Details resets
the offset. Esc and Ctrl+C are covered for both navigation paths.

Regression tests in `internal/app/docs_review_fixes_test.go` reproduced all three
findings on the original production source (focused run exit 1), and pass after
the fixes. They cover both comment refresh stages, exact failed-write drafts,
typed read-only recovery without duplicate writes, reordered/filtered active
rows, late canceled results, and a scrolled `/documents` record. The existing
program and PTY scenarios now verify a fresh active list containing the newly
activated row (PTY verifies the new total). The assertion ledger records reuse.

| Command | Exit | Evidence |
| --- | ---: | --- |
| Baseline `make -C cli check` | 0 | Clean baseline; `/tmp/031-review-baseline-check.log`. |
| `make -C cli format` | 0 | Only intended source/test files formatted; `/tmp/031-review-format.log`. |
| `go test -count=1 ./internal/app ./internal/documents ./internal/changes` (CLI cwd) | 0 | `/tmp/031-review-focused.log`. |
| `go test -count=1 -timeout=3m ./integration -run '^TestCLIProgram031'` (CLI cwd) | 0 | `/tmp/031-review-program.log`. |
| Final `make -C cli check` | 0 | Format, lint (zero issues), vet, unit race, architecture and tooling pass; `/tmp/031-review-check-final.log`. |
| Final `make -C cli coverage` | 0 | Complete independent unit profile **5515/6368 (86.6049%)**; 80% integer gate passes; `/tmp/031-review-unit-final2.log`. |
| `make -C cli deps-audit` | 0 | No vulnerabilities; `/tmp/031-review-deps.log`. Production and dependencies unchanged afterward. |
| Final `make -C cli integration-coverage` | 0 | Complete independent terminal profile **4897/6368 (76.9001%)**; 70% integer gate passes; `/tmp/031-review-integration-final2.log`. All 34 program scenarios and the real PTY scenario finish without skips, missing counters or cleanup failures. |

Both final campaign status files report complete and exit 0. Tested base revision
is `30832e776e40199517d48ccdf67a507937b03ae3`, plus the recorded worktree changes
and untracked regression test. Both campaigns hash the same 77 production Go
files, with aggregate SHA-256
`4e1c31e3c4857d12567ef6ff7bffce54c753a7095c63aaeeff7c7f8e6d71c23b`
(SHA-256 of sorted compact JSON of the production file hash mapping). The terminal
child binary SHA-256 is
`befd23232a2a7275bf0203d1ee877e5d4543e462148acac7841dfd8f2f33ada7`.
Go is `go1.26.8-X:nodwarf5`, golangci-lint 2.13.1, govulncheck 1.7.0.
Independent raw profiles, exact package totals, uncovered statements/functions,
scenario results, tool versions and commands/exits remain under
`cli/.coverage/{unit,integration}`. Only this evidence and the implementation log
were edited after the final campaigns; production and test source is unchanged.

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cli/cmd/mch` | 0/3 | 1/3 |
| `cli/internal/agent` | 297/357 | 270/357 |
| `cli/internal/app` | 2497/3034 | 2318/3034 |
| `cli/internal/changes` | 859/989 | 742/989 |
| `cli/internal/configurations` | 170/183 | 167/183 |
| `cli/internal/documents` | 484/509 | 400/509 |
| `cli/internal/dto` | 0/0 | 0/0 |
| `cli/internal/epics` | 221/221 | 191/221 |
| `cli/internal/health` | 48/50 | 47/50 |
| `cli/internal/help` | 6/6 | 6/6 |
| `cli/internal/navigation` | 25/40 | 19/40 |
| `cli/internal/projects` | 224/234 | 200/234 |
| `cli/internal/styles` | 0/0 | 0/0 |
| `cli/internal/testcases` | 130/132 | 112/132 |
| `cli/internal/ui` | 8/8 | 8/8 |
| `cli/pkg/briefprocess` | 128/161 | 84/161 |
| `cli/pkg/client` | 402/424 | 317/424 |
| `cli/pkg/documentprocess` | 16/17 | 15/17 |

Remaining gaps include the untested main unit boundary, navigation fallbacks,
agent/brief error paths, transport rejection branches and document/history
invalid-input, cancellation and printing failures. Exact statements and functions
are retained in each campaign's `uncovered.txt` and `functions.log`; the whole
production inventory remains in both denominators and profiles are independent.

An initial focused program run failed because its final read-count assertion
also counted a later ordinary reload; the assertion was moved to the inactive
return boundary. The first terminal campaign exited 2 on the old PTY return
status marker; it produced no valid coverage result. Its raw artifacts are kept
at `/tmp/031-review-integration-failed-01`. After updating that assertion, both
campaigns exited 2 because the assertion ledger changed while they ran; their
raw artifacts remain at `/tmp/031-review-{unit,integration}-input-changed`.
All program/PTY tests passed in those rejected campaigns, but no coverage success
is claimed for them. The final complete campaigns above supersede those runs.
The implementation log retains the skill-required final blank line; any
`git diff --check` finding for that log line is intentional. No other whitespace
finding remains. No live backend/database or Git publication was used.

## 031 review fixes 02 — inactive deletion refresh and stale detail isolation

Both findings are valid under 031-16/19 and 031-15/19 respectively. Change list
loads and post-delete refreshes now capture the same inactive-aware reader.
Opening change document history invalidates and cancels the originating change
request, preventing a delayed detail snapshot from overwriting a successful
history activation and its return refresh. HTTP contracts remain unchanged.

`Test031DeleteRefreshPreservesListModeAndReadOnlyRecovery` covers both list modes,
success and failed post-delete refreshes, retained undeleted rows and read-only
recovery without replaying the mutation. The keyboard-driven
`Test031HistoryActivationRejectsPendingDetailSnapshot` pauses comments after
capturing the old active documents, opens history, activates an older document,
returns through Esc/Ctrl+C and then delivers the old snapshot despite cancellation.
It verifies cancellation at history entry and unchanged detail/status/error on
late delivery. Both regression tests failed on the original production code;
raw expected-failure output remains in `/tmp/031-review02-repro.log`.

| Command | Exit | Evidence |
| --- | ---: | --- |
| Baseline `make -C cli check` | 0 | `/tmp/031-review02-baseline-check.log`; clean baseline. |
| Focused two new regression tests (before fixes) | 1 | Both reported regressions reproduced; `/tmp/031-review02-repro.log`. |
| `make -C cli format` | 0 | `/tmp/031-review02-format.log`; only intended edits. |
| `go test -count=1 -race ./internal/app ./internal/changes ./internal/documents` (CLI cwd) | 0 | `/tmp/031-review02-focused.log`. |
| Final `make -C cli check` | 0 | `/tmp/031-review02-check.log`; format, lint, vet, race, architecture and tooling pass, including the final strengthened cancellation assertion. |
| `make -C cli coverage` | 0 | `/tmp/031-review02-unit.log`; **5525/6370 (86.7347%)**, 80% integer gate passes. |
| `make -C cli deps-audit` | 0 | `/tmp/031-review02-deps.log`; no vulnerabilities. |
| `make -C cli integration-coverage` | 0 | `/tmp/031-review02-integration.log`; **4899/6370 (76.9074%)**, 70% integer gate passes. |

Both independent campaign status files report complete and exit 0. All 34 program
scenarios and the real PTY scenario finish without skips, missing counters,
assertion failures or cleanup failures. Base revision is
`59d4a8c5df24d47e82a83fe0dbbcec26c188e7b0`, plus the recorded worktree diff.
Both campaigns hash the same 77 production files; the sorted compact JSON
production-hash mapping has SHA-256
`ae6bde237eb17dd4d121a06ec5c48be18aa104f30fce25bca4a90f19a454af42`.
The terminal child binary SHA-256 is
`6732c1df99ee57ff3c13c36c3cef9e093578e47d0cca55b2b257d1e04efbfd59`.
Go is `go1.26.8-X:nodwarf5`, golangci-lint 2.13.1 and govulncheck 1.7.0.
Raw profiles, commands/exits, source provenance, exact package totals and uncovered
statements/functions remain in `cli/.coverage/{unit,integration}`. Only this
checkpoint and the implementation log changed after the completed campaigns;
production, tests and the assertion ledger are unchanged.

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cli/cmd/mch` | 0/3 | 1/3 |
| `cli/internal/agent` | 297/357 | 270/357 |
| `cli/internal/app` | 2504/3036 | 2320/3036 |
| `cli/internal/changes` | 862/989 | 742/989 |
| `cli/internal/configurations` | 170/183 | 167/183 |
| `cli/internal/documents` | 484/509 | 400/509 |
| `cli/internal/dto` | 0/0 | 0/0 |
| `cli/internal/epics` | 221/221 | 191/221 |
| `cli/internal/health` | 48/50 | 47/50 |
| `cli/internal/help` | 6/6 | 6/6 |
| `cli/internal/navigation` | 25/40 | 19/40 |
| `cli/internal/projects` | 224/234 | 200/234 |
| `cli/internal/styles` | 0/0 | 0/0 |
| `cli/internal/testcases` | 130/132 | 112/132 |
| `cli/internal/ui` | 8/8 | 8/8 |
| `cli/pkg/briefprocess` | 128/161 | 84/161 |
| `cli/pkg/client` | 402/424 | 317/424 |
| `cli/pkg/documentprocess` | 16/17 | 15/17 |

Remaining gaps include the main unit boundary, navigation fallbacks, agent and
brief error paths, transport rejection branches and document/history error paths;
exact locations remain in each campaign's `uncovered.txt` and `functions.log`.
No required scenario failed or was skipped in final verification. No backend,
database or Git publication was used. The skill-required final blank line in
`cli/implementation-log.md` remains intentional.

## 031 review fixes 03 — history exit reconciliation and portable timestamps

Both findings are valid under 031-14/15/19 and 031-24. Esc/Ctrl+C now cancels
the pending history mutation's HTTP/process work while retaining its revision
and busy state until the result arrives. The application then refreshes the
originating owner using reads only, retaining committed feedback and selection.
Read/print-only history operations still cancel immediately. Timestamp
assertions in the change list/detail rendering test use the active local
timezone rather than assuming Zagreb summer time. HTTP contracts are unchanged.

`Test031HistoryExitReconcilesPendingMutation` pauses activation and comment
restoration at the write, retained-history read, active-selection read and
printing boundaries, with both exit keys. It verifies committed feedback,
owner refresh, failed-refresh/read-only retry, no duplicate Space write and
rejection of duplicate results. `Test031OwnerHistoryExitDrainsQueuedMutationResult`
checks the `/documents` return path with a result queued behind the exit key.
The existing complete-program document/comment scenario now cancels a pending
history read after each successful mutation and verifies returned committed
feedback and the refreshed delete target.

| Command | Exit | Evidence |
| --- | ---: | --- |
| Baseline `make -C cli check` | 0 | `/tmp/031-review03-baseline-check.log`; clean baseline in the host timezone. |
| Baseline `TZ=UTC go test -count=1 ./internal/app -run '^TestChangesCommandLoadsAndRendersBackendRows$'` (CLI cwd) | 1 | Both reported assertions fail: rendered `10:45`, expected `12:45`. |
| Initial history regression before production fixes | 1 | `/tmp/031-review03-history-repro.log`; committed feedback is lost for both mutation kinds. Two spec/print cases also timed out because the initial fake activation failed to clear deletion metadata; that fixture was repaired before final verification. |
| `make -C cli format` | 0 | `/tmp/031-review03-format.log`; only intended source/test edits. |
| `TZ=UTC go test -count=1 -race ./internal/app ./internal/documents` (CLI cwd) | 0 | `/tmp/031-review03-utc-focused.log`; complete app/document suites, including new regressions. |
| `TZ=Europe/Zagreb go test -count=1 ./internal/app -run '^TestChangesCommandLoadsAndRendersBackendRows$'` (CLI cwd) | 0 | `/tmp/031-review03-zagreb.log`; list/detail assertions also pass in Zagreb. |
| `go test -count=1 -race -timeout=60s ./integration -run '^TestCLIProgram031DocumentCommentsAndHistory$'` (CLI cwd) | 0 | `/tmp/031-review03-program-focused.log`; keyboard-driven pending-read cancellation passes for both mutations. |
| Final `make -C cli check` | 0 | `/tmp/031-review03-check.log`; format, lint, vet, race, architecture and tooling pass. |
| `make -C cli coverage` | 0 | `/tmp/031-review03-unit.log`; **5546/6386 (86.8462%)**, 80% integer gate passes. |
| `make -C cli deps-audit` | 0 | `/tmp/031-review03-deps.log`; no vulnerabilities. |
| `make -C cli integration-coverage` | 0 | `/tmp/031-review03-integration.log`; **4917/6386 (76.9966%)**, 70% integer gate passes. |

Both independent campaigns report complete and exit 0. All 34 program scenarios,
including the covered standalone startup child, and the real PTY scenario pass
without skips, missing counters, assertion failures or cleanup failures.
The tested base revision is `51f42539c6366613d3c994027fe83248819e1bc4`, plus
the recorded worktree diff. Both campaigns hash the same 77 production files;
the sorted compact JSON source mapping has SHA-256
`901cbabee6bd1a8728eb2dce1e8112a99373b8448b25d7abf3ba90757e21e991`.
The terminal child binary SHA-256 is
`cba66a536c72f058e7a80054ee1aa1b74b9b46d9003c940dd0d625d4161d0814`.
Go is `go1.26.8-X:nodwarf5`, golangci-lint 2.13.1 and govulncheck 1.7.0.
Profiles, exact package totals, source provenance, commands/exits and uncovered
statements/functions remain under `cli/.coverage/{unit,integration}`.

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cli/cmd/mch` | 0/3 | 1/3 |
| `cli/internal/agent` | 297/357 | 270/357 |
| `cli/internal/app` | 2516/3045 | 2329/3045 |
| `cli/internal/changes` | 862/989 | 742/989 |
| `cli/internal/configurations` | 170/183 | 167/183 |
| `cli/internal/documents` | 493/516 | 408/516 |
| `cli/internal/dto` | 0/0 | 0/0 |
| `cli/internal/epics` | 221/221 | 191/221 |
| `cli/internal/health` | 48/50 | 47/50 |
| `cli/internal/help` | 6/6 | 6/6 |
| `cli/internal/navigation` | 25/40 | 19/40 |
| `cli/internal/projects` | 224/234 | 200/234 |
| `cli/internal/styles` | 0/0 | 0/0 |
| `cli/internal/testcases` | 130/132 | 112/132 |
| `cli/internal/ui` | 8/8 | 8/8 |
| `cli/pkg/briefprocess` | 128/161 | 84/161 |
| `cli/pkg/client` | 402/424 | 318/424 |
| `cli/pkg/documentprocess` | 16/17 | 15/17 |

Remaining gaps include the main unit boundary, navigation fallbacks, agent/brief
error paths, transport rejection branches and document/history failures; exact
locations remain in each campaign's `uncovered.txt` and `functions.log`.
All campaign inputs matched their recorded hashes before these final checkpoint
and implementation-log writes. Production, tests, tooling and the assertion
ledger have not changed since verification. No backend, database or Git
publication was used. The implementation log's final blank line is intentional.

## 031 review fixes 04 — owner history feedback, confirmation filtering and Docker bat

All three findings are valid under 031-09/14/18/19 and the documented Docker
verification contract. Returning from `/documents` history now carries the
committed activation outcome into the owner model. Pending, failed, cancelled
and successful refreshes retain that outcome; `/retry` performs reads only.
New owner/operation feedback replaces the previous outcome. The document delete
confirmation renders the same filtered options used for selection. Docker
`test_version` installs Python and Debian's bat package and supplies the `bat`
executable name through its `batcat` alias. HTTP contracts are unchanged.

Regressions: `Test031OwnerHistoryCommitSurvivesRefreshFailure` covers both exit
keys, visible feedback, repeated refresh failure, recovery, restored selection
and exactly one activation. `Test031OwnerHistoryCommitRefreshCancellationAndScopeReset`
covers cancellation and new-owner/insert isolation.
`Test031DocumentConfirmationFilterMatchesRenderedSelection` covers yes/no and
unmatched filtering with exact deletion effects. The isolated
`test_docker_provisions_bat_before_check` executes the real container recipe
against a Debian-style batcat installation before permitting `make init check`.

| Command | Exit | Evidence |
| --- | ---: | --- |
| Pre-fix focused app regressions | 1 | `/tmp/031-review04-repro.log`; lost commit feedback and mismatched rendered options reproduced. |
| Pre-fix `python3 -B -m unittest discover -s scripts -p makefile_test.py` (CLI cwd) | 1 | `/tmp/031-review04-tooling-repro.log`; Docker recipe installs Python without bat. |
| `make -C cli format` | 0 | `/tmp/031-review04-format.log`; only intended changes. |
| `go test -count=1 -race ./internal/app ./internal/documents ./pkg/documentprocess` (CLI cwd) | 0 | `/tmp/031-review04-focused.log`; complete focused suites pass. |
| `make -C cli tooling-test` | 0 | `/tmp/031-review04-tooling.log`; 37 Python tests plus architecture fixtures and PTY harness tests pass. |
| `make -C cli check` | 0 | `/tmp/031-review04-check.log`; formatting, lint, vet, race, architecture and tooling pass. |
| `make -C cli coverage` | 0 | `/tmp/031-review04-unit.log`; **5558/6398 (86.8709%)**, 80% integer gate passes. |
| `make -C cli deps-audit` | 0 | `/tmp/031-review04-deps.log`; no vulnerabilities. |
| `make -C cli integration-coverage` | 0 | `/tmp/031-review04-integration.log`; **4923/6398 (76.9459%)**, 70% integer gate passes. |
| `make -C cli test_version` | 2 | `/tmp/031-review04-docker.log`; bat installation succeeds, but the bind-mounted Makefile is unreadable. |
| Exact `test_version` image/shell recipe with a tar-copied checkout | 0 | `/tmp/031-review04-docker-copy.log`; stock `golang:1.26.0`, pinned tooling, full check and the real bat test pass. |
| `python3 -B -m unittest discover -s cli/scripts -p documentation_test.py` | 0 | `/tmp/031-review04-docs.log`; final command/link and route-ledger checks pass. |

The direct Docker target remains blocked by this host's bind-mount permissions:
a separate stock-image probe cannot read `/project/cli/Makefile` even as container
root (`/tmp/031-review04-docker-mount.log`). Docker enables SELinux, and the
checkout has `user_home_t` labels; this evidence points to an existing host
labeling restriction. The copied-checkout verification uses the exact target's
image and shell recipe without relabeling source or changing container security.
It includes the tracked repository files and a temporary empty Git repository
for tooling's ignore-rule check. It does not establish that the bind-mounted
target passes on this host. The disposable container was removed on exit.

Both independent host coverage campaigns report complete and exit 0. All 34
program scenarios (including covered startup) and the real PTY scenario pass
without skips, missing counters or cleanup failures. Tested base revision:
`8493d27eee2f97f9b7b0415d4b7be96b89fe6528`, plus the recorded worktree diff.
Both campaigns hash the same 77 production files; the sorted compact JSON source
mapping has SHA-256
`ddcde65705e56828f99b6b70371e4b15074cee0165315c09e45c3135a65ac390`.
Terminal child binary SHA-256:
`80a85c616f01a1330d4314a0236258b80ffdd1c5ed03925ce1d30ef38d62c065`.
Host Go is `go1.26.8-X:nodwarf5`, golangci-lint 2.13.1 and govulncheck 1.7.0.
Raw profiles, provenance, command exits and exact uncovered statements/functions
remain under `cli/.coverage/{unit,integration}`.

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cli/cmd/mch` | 0/3 | 1/3 |
| `cli/internal/agent` | 297/357 | 270/357 |
| `cli/internal/app` | 2518/3047 | 2329/3047 |
| `cli/internal/changes` | 862/989 | 742/989 |
| `cli/internal/configurations` | 170/183 | 167/183 |
| `cli/internal/documents` | 503/526 | 414/526 |
| `cli/internal/dto` | 0/0 | 0/0 |
| `cli/internal/epics` | 221/221 | 191/221 |
| `cli/internal/health` | 48/50 | 47/50 |
| `cli/internal/help` | 6/6 | 6/6 |
| `cli/internal/navigation` | 25/40 | 19/40 |
| `cli/internal/projects` | 224/234 | 200/234 |
| `cli/internal/styles` | 0/0 | 0/0 |
| `cli/internal/testcases` | 130/132 | 112/132 |
| `cli/internal/ui` | 8/8 | 8/8 |
| `cli/pkg/briefprocess` | 128/161 | 84/161 |
| `cli/pkg/client` | 402/424 | 318/424 |
| `cli/pkg/documentprocess` | 16/17 | 15/17 |

Remaining gaps include the main unit boundary, navigation fallbacks, agent/brief
error paths, transport rejection branches and document/history failures; see the
campaigns' `uncovered.txt` and `functions.log`. Campaign input hashes matched
before these checkpoint/log writes; production, tests and tooling remain the
verified sources. No backend/database use, Git commit/push or deployment occurred.
Next action: caller review/publication; Docker bind-mount labeling remains a host
verification limitation. The implementation log's final blank line is intentional.

## 031 review fixes 05 — lossless comment recovery and history detail reload

Both findings are valid under 031-12/14/19. Comment editing now saves the full
raw body in `editorDraft` before launching the external editor. Editor startup
failure leaves tabs and CRLF intact for Enter submission and Ctrl+E retry.
History records when it cancels an in-flight change/details read and starts a
fresh full detail read on return, including after history activation. Successful
reload restores editing, testcase readiness and the originating selection;
failed reload remains visible and permits read-only retry. Committed activation
feedback survives pending, failed and successful reads. Canceled results remain
stale. Navigation away clears the pending reload attribution. HTTP contracts and
feature boundaries are unchanged.

Regressions in `cli/internal/app/docs_review_fixes_test.go`:
`Test031CommentEditorFailureRetainsRawDraft` covers failed startup, exact retry
files, same-ID Enter update and independent comment preservation.
`Test031HistoryReturnRestartsCanceledDetailRead` covers Esc/Ctrl+C, successful
and failed reload, retry, selection, late results, and Enter/Delete/Space access.
`Test031HistoryActivationRejectsPendingDetailSnapshot` now also proves detail
readiness and retained activation feedback after canceling a real pending read.
`Test031HistoryDetailReloadRetainsCommittedActivation` proves the committed
outcome survives failure and retry without repeating activation.

| Command | Exit | Evidence |
| --- | ---: | --- |
| Pre-fix focused regressions (CLI cwd) | 1 | `/tmp/031-review05-repro.log`; both findings reproduced, including return after activation. |
| `make -C cli format` | 0 | `/tmp/031-review05-format.log`; only intended source changes. |
| `go test -count=1 -race ./internal/app ./internal/documents ./pkg/documentprocess` (CLI cwd) | 0 | `/tmp/031-review05-focused.log`; complete focused suites on final source. |
| `make -C cli check` | 0 | `/tmp/031-review05-check.log`; formatting, lint, vet, race, architecture and tooling pass. |
| `make -C cli coverage` | 0 | `/tmp/031-review05-unit.log`; **5581/6420 (86.9315%)**, 80% integer gate passes. |
| `make -C cli deps-audit` | 0 | `/tmp/031-review05-deps.log`; no vulnerabilities. |
| `make -C cli integration-coverage` | 0 | `/tmp/031-review05-integration.log`; **4931/6420 (76.8069%)**, 70% integer gate passes. |
| `python3 -B -m unittest discover -s cli/scripts -p documentation_test.py` | 0 | `/tmp/031-review05-docs.log`; command/link and route-ledger checks pass. |

During editing, one focused run failed compilation after a local predicate was
placed in the wrong result handler, and one targeted run failed an incorrect
expected activation-status string. Both were corrected before the final focused
race suites and required checks above. Initial passing check/unit measurements
were superseded by the final campaigns after adding committed-feedback coverage.
No failed or superseded run is used for these final coverage claims.

Both independent campaigns report complete and exit 0. All 34 selected program
scenarios (including covered standalone startup) and the real PTY scenario pass
without skips, missing counters, timeouts or cleanup failures. Tested base:
`aca8bead4feed87a84adf3752bffaa1f010ca9aa`, plus the recorded worktree diff.
Both campaigns hash the same 77 production files; the sorted compact JSON source
mapping SHA-256 is `fd271cd0bf79299f4b4afca282929cd5c765a4b35071a44c86d0f835a36b1160`.
Terminal child binary SHA-256: `6aa2c17f8d5553fdaa61aff83d45816d2f6dbbbdaccc57896069a4f588e557ba`.
Host tools: Go `go1.26.8-X:nodwarf5`, golangci-lint 2.13.1, govulncheck 1.7.0.
Raw independent profiles, command exits, scenarios, source provenance and exact
uncovered statements/functions remain in `cli/.coverage/{unit,integration}`.

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cli/cmd/mch` | 0/3 | 1/3 |
| `cli/internal/agent` | 297/357 | 270/357 |
| `cli/internal/app` | 2541/3069 | 2337/3069 |
| `cli/internal/changes` | 862/989 | 742/989 |
| `cli/internal/configurations` | 170/183 | 167/183 |
| `cli/internal/documents` | 503/526 | 414/526 |
| `cli/internal/dto` | 0/0 | 0/0 |
| `cli/internal/epics` | 221/221 | 191/221 |
| `cli/internal/health` | 48/50 | 47/50 |
| `cli/internal/help` | 6/6 | 6/6 |
| `cli/internal/navigation` | 25/40 | 19/40 |
| `cli/internal/projects` | 224/234 | 200/234 |
| `cli/internal/styles` | 0/0 | 0/0 |
| `cli/internal/testcases` | 130/132 | 112/132 |
| `cli/internal/ui` | 8/8 | 8/8 |
| `cli/pkg/briefprocess` | 128/161 | 84/161 |
| `cli/pkg/client` | 402/424 | 318/424 |
| `cli/pkg/documentprocess` | 16/17 | 15/17 |

Remaining gaps include the main unit boundary, navigation fallbacks, agent/brief
error paths, transport rejection branches and document/history failures. Exact
locations are in each campaign's `uncovered.txt` and `functions.log`.
All recorded input hashes matched before these checkpoint/log writes;
production, tests and tooling remain the verified sources. The prior Docker
bind-mount limitation remains historical evidence; Docker was not rerun because
this pass changes neither tooling nor toolchain compatibility. No backend or
database access, Git publication or deployment occurred. The implementation log's
final blank line is intentional.

## 031 review fixes 06 — safe comment previews and stable history error selection

Both findings are valid under 031-10/12/14/19. Comment previews now escape
terminal controls before preview clipping through the existing document-screen
escaping implementation, moved to shared `internal/ui`. The document helpers
retain their behavior and exported contracts; feature import boundaries stay
intact. OSC 52, CSI display/style sequences, C1 controls and format controls are
shown as escaped text. Raw comment bodies, including tabs, CRLF and content
beyond the preview, remain the exact external-editor seed and recovery draft.
History now reconciles refreshed, sorted rows by selected ID before reading and
validating the active set, retaining #8 when [9,8] becomes [10,9,8] even if that
second read fails. Committed activation/undelete feedback and read-only retry
remain intact. Backend HTTP contracts are unchanged.

Regressions: `Test031CommentPreviewEscapesTerminalControls` proves escaped rows
and final DetailsView output at narrow/wide widths, unchanged raw bodies and
three-line previews. `Test031CommentControlEscapingRetainsRawEditorSeed` proves
the shell's displayed preview is safe and the actual editor file/draft is exact.
`Test031HistoryRetainsSelectionAfterPartialReadFailure` exercises spec/comment,
refresh/activation and active-read/validation failures, retaining selected ID,
avoiding printing on failed reads and avoiding mutation replay on retry.

| Command | Exit | Evidence |
| --- | ---: | --- |
| Pre-fix `make -C cli check` | 0 | `/tmp/031-review06-baseline-check.log`; clean baseline. |
| Pre-fix targeted regressions (CLI cwd) | 1 | `/tmp/031-review06-repro.log`; controls remain raw and all eight history cases select #9 instead of #8. |
| `make -C cli format` | 0 | `/tmp/031-review06-format.log`; intended files only. |
| `go test -count=1 -race ./internal/changes ./internal/documents ./internal/app ./internal/ui` (CLI cwd) | 0 | `/tmp/031-review06-focused.log`; final focused suites pass. |
| `make -C cli check` | 0 | `/tmp/031-review06-check.log`; format, lint, vet, unit race, architecture and tooling checks pass. |
| `make -C cli coverage` | 0 | `/tmp/031-review06-coverage.log`; **5581/6420 (86.9315%)**, 80% integer gate passes. |
| `make -C cli deps-audit` | 0 | `/tmp/031-review06-deps-audit.log`; no vulnerabilities. |
| `make -C cli integration-coverage` | 0 | `/tmp/031-review06-integration-coverage.log`; **4931/6420 (76.8069%)**, 70% integer gate passes. |

An initial focused run failed compilation because the new test treated
`editorDraft` as a struct instead of a string pointer. The assertion was fixed
before the final focused suites and required checks. No failed run establishes
coverage. Final campaigns are complete, independent and exit 0; all 34 selected
program scenarios (including covered startup) and the real PTY scenario pass
without skips, missing counters, assertion failures, crashes, timeouts or cleanup
failures. Source base: `acf0c620f8c3e5ddd5b950ba872919ef1b35e64a` plus worktree
diff and the untracked shared helper, whose hash is recorded in both campaigns.
All input hashes matched each other and the final source before checkpoint/log
writes. The 78-file production mapping's sorted compact JSON SHA-256 is
`86eaa19c7ab26e9b34de5ca65bc492a499b9b8b017e94a2eb0701fb0cef301d4`.
Terminal child binary SHA-256:
`b0730fcc54b958f7ab07f5553997104f38685dec0d7588b2296e30e5b5ba4195`.
Tools: Go `go1.26.8-X:nodwarf5`, golangci-lint 2.13.1, govulncheck 1.7.0.
Raw profiles, commands/exits, scenarios, provenance, exact package counts and
uncovered statements/functions remain in `cli/.coverage/{unit,integration}`.

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cli/cmd/mch` | 0/3 | 1/3 |
| `cli/internal/agent` | 297/357 | 270/357 |
| `cli/internal/app` | 2541/3069 | 2337/3069 |
| `cli/internal/changes` | 862/989 | 742/989 |
| `cli/internal/configurations` | 170/183 | 167/183 |
| `cli/internal/documents` | 490/513 | 405/513 |
| `cli/internal/dto` | 0/0 | 0/0 |
| `cli/internal/epics` | 221/221 | 191/221 |
| `cli/internal/health` | 48/50 | 47/50 |
| `cli/internal/help` | 6/6 | 6/6 |
| `cli/internal/navigation` | 25/40 | 19/40 |
| `cli/internal/projects` | 224/234 | 200/234 |
| `cli/internal/styles` | 0/0 | 0/0 |
| `cli/internal/testcases` | 130/132 | 112/132 |
| `cli/internal/ui` | 21/21 | 17/21 |
| `cli/pkg/briefprocess` | 128/161 | 84/161 |
| `cli/pkg/client` | 402/424 | 318/424 |
| `cli/pkg/documentprocess` | 16/17 | 15/17 |

Remaining gaps include the main unit boundary, navigation fallbacks, agent/brief
error paths, HTTP rejection branches and document/history failures; exact
locations are in each campaign's `uncovered.txt` and `functions.log`. Moving
escaping to `ui` moves its 13 statements from documents into the shared package;
the production denominator and aggregate covered counts are unchanged.
No backend/database access, Git publication or deployment occurred. Docker was
not rerun because tooling and toolchain compatibility are unchanged.
Documentation command/link checks also pass:
`python3 -B -m unittest discover -s cli/scripts -p documentation_test.py`
exits 0 (`/tmp/031-review06-docs.log`). Final diff has no whitespace errors.
