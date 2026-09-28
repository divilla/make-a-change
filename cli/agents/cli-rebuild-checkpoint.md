# CLI rebuild checkpoint

## Current handoff: P3 reviewed, ready for dev merge

2026-09-28. [Plan](../../agent/cli-rebuild-plan.md),
[architecture](../../docs/cli-architecture.md),
[contract and assertion ledger](cli-contracts.md),
[P3 specification](../../agent/specs/022-cli-epic-management.md).
P0 merged as `e46bf64`; P1 as `6a3d5bb`; P2 as actual dev `1658624`.
P3 implementation `254e5af` and fixes `9c6f516`, `1fcdde6`, `c8ba036` are
published on `change/022-cli-epic-management`. Native review pass04 against
pinned P2 dev found no actionable regressions and independently passed unit and
integration-test Make targets. Final production revision is `c8ba036`;
subsequent edits are documentation only. Next publish checkpoint, merge via dev
helper and verify actual squash SHA before starting P4. No stage/prod promotion.

## Established contracts

- Epic list(project_id), details(id), create(project_id,name), update(id,name),
  delete(id) use one typed POST each. Create returns a 201 ID; update/delete
  accept empty 204. Required numeric IDs, counts, completion and timestamps match
  current backend DTOs without null defaults, guessed envelopes or history.
  Server completion is preserved, even when different from a local testcase ratio.
  The P2 bounded context-aware transport retains status and underlying errors.
- Epics own forms, validation, drafts, sequencing, results and operation identity.
  Actual list/detail/create/edit/delete actions replace the placeholder, including
  help, loading, empty/error/success states, confirmation and unchanged-edit no-op.
  A committed create/update/delete survives failed refresh. Created IDs remain
  usable; `/retry` reads only; unrelated rows survive deletion refresh failure.
- Exact raw editor bytes and slash-prefixed form data survive failed saves,
  repeated unchanged editor retries and Enter retries. Cancel/discard performs
  no write. Every loaded edit name remains literal, including `/save`; unchanged Enter
  performs no write and prompt edits preserve slash-prefixed names.
- Epic lists fit the space remaining after the rendered shell, including wrapped
  errors/footer, and keep the selected row visible across movement and resizing.
  Details likewise fit the remaining height, with arrow/page scrolling and reset
  on selection or successful reads/mutations. ANSI styles and wide/combining
  graphemes survive linear-time row/detail truncation. Mutation outcomes persist only for their failed refresh/read retry; recovery,
  navigation or unrelated operations clear the retained outcome.
- Project/entity/generation-bound reads cancel or are ignored after navigation,
  a new operation or project selection. Loading lists clear cached selectable
  rows. Filter and detail selector openings now advance generation, preventing
  canceled same-project reads from replacing reopened options or errors.
  Empty find submissions restart the canceled list/detail read automatically.
  Changes consume typed epic selector values via shell presentation conversion.
- P1 atomic asynchronous config writes, save permissions, queue draining, editor
  cleanup/redraw and literal draft regressions remain intact. P2 manual config
  recovery ordering, canceled reads and deletion/persistence diagnostics still
  pass. No backend, database, dependency, cli-proto, frontend or Flow asset changed.

## Final verification

All required scenarios completed without skips, crashes, timeouts or cleanup
failures. Both coverage status files record `complete: true`, internal threshold
exit1 (Make exit2). Unit packages with no executable statements stay in inventory.
Overall unit is **2723/3201 (85.0672%)**; independent terminal is
**2028/3201 (63.3552%)**. Strict >95% and >90% gates fail honestly. Numerical
shortfalls alone may continue through the authorized intermediate dev merge;
the full rebuild remains incomplete.

| Final command | Exit | Evidence |
| --- | ---: | --- |
| Scoped `golangci-lint fmt --no-config --enable gofumpt --enable goimports` | 0 | Three touched Go files; diff inspected. |
| `cd cli && go test ./internal/app ./internal/epics -count=1` | 0 | Existing suites plus empty/whitespace find list/detail recovery and stale-response rejection. |
| `cd cli && go test ./integration -run '^TestCLIProgramEpicDelayedScopeAndShutdown$' -count=1 -timeout=90s` | 0 | Extended keyboard program verifies canceled HTTP list/detail reads restart after empty find. |
| `make -C cli check` | 2 | Only untouched format/revive baseline below; vet, unit race, architecture, 36 Python tooling tests and Go fixtures pass. |
| `make -C cli coverage` | 2 | Complete independent unit measurement; strict numerical gate fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete-program, startup and retained integration checks. |
| `make -C cli integration-coverage` | 2 | Complete manifest-selected program/startup/real PTY campaign; strict numerical gate fails. |
| `git diff --check` and documentation tests | 0 | Whitespace, local links, commands, route/manifest ledger. |

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cli/cmd/mch` | 0/3 | 1/3 |
| `cli/internal/app` | 1492/1754 | 1022/1754 |
| `cli/internal/changes` | 477/599 | 383/599 |
| `cli/internal/dto` | 0/0 | 0/0 |
| `cli/internal/epics` | 214/214 | 187/214 |
| `cli/internal/help` | 6/6 | 6/6 |
| `cli/internal/navigation` | 25/40 | 14/40 |
| `cli/internal/projects` | 211/221 | 183/221 |
| `cli/internal/styles` | 0/0 | 0/0 |
| `cli/internal/testcases` | 5/6 | 5/6 |
| `cli/internal/ui` | 8/8 | 7/8 |
| `cli/pkg/client` | 285/350 | 220/350 |

Unit leaves **478** statements uncovered; terminal leaves **1173**. Largest gaps
remain app legacy operation/error paths, changes parsing/rendering and unported
client methods; cmd is 0/3 unit. Epic terminal gaps include invalid scope/ID,
wrong-entity reads, viewport boundary cases and failed retry permutations;
unit tests cover all epic statements. Exact uncovered blocks/functions, profiles,
command journals, source manifests, versions and child hash are retained in
`cli/.coverage/{unit,integration}/`. All 11 manifest program entries and the real
PTY scenario ran. The extended epic program uses injected I/O and a fake backend;
no live backend/database was used. Adapter-only counters remain unit-only.

Raw command logs and exits: `cli/.coverage/spec022-review03/`. Before the fix,
the focused `TestP304EpicEmptyFindRestartsCanceledRead` command exited1: all four
list/detail × empty/whitespace cases failed because no restart command was returned.
The post-fix targeted unit and program commands passed. No production/test source
changed after the final campaigns; subsequent checkpoint/ledger/log/plan edits are
docs only. Go1.26.8-X:nodwarf5, golangci-lint2.13.1, govulncheck1.7.0, socat1.8.1.1.
Covered child binary SHA256:
`f625d07bed31019218c8bcff322ee683bf5a309fa8c751e52248d5faa4fae678`.
Docker and benchmarks were not rerun: no toolchain/dependency or performance contract changed.

## Baseline owners and next action

Four untouched formatting files remain: cmd/mch/main.go and app/clipboard.go
(final cleanup/P10); testcases/api.go and model.go (P5). Final revive reports
styles/styles.go, testcases/api.go and ui/layout.go package comments (P5/P10);
prior runs also sampled cmd/mch/main.go and help/commands.go. P3 repaired epics/api.go and model.go.

Next: caller reviews P3 fixes 03 and handles publication, then the authorized dev merge
and records its actual SHA. P4 migrates change operations to current typed
contracts and feature-owned behavior, retaining project/epic selectors and all
P1–P3 editor, config, partial-success and stale-response assertions. P5 testcases,
P6 documents, P7 configs/health, P8 clarification/process, P9 review/fix and P10
combined acceptance remain, followed by evidence-based cleanup. No predicted
squash hash or full-rebuild completion claim.
