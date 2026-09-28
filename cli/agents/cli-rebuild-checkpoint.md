# CLI rebuild checkpoint

## Current handoff: P1 reviewed, P2 next

2026-09-28. The full sequential CLI factory is authorized by the user; no stage
or production promotion. [Plan](../../agent/cli-rebuild-plan.md),
[architecture](../../docs/cli-architecture.md), [contract and test-reuse ledger](cli-contracts.md).

P0/spec019 merged to dev as `e46bf64`. P1/[spec020](../../agent/specs/020-cli-startup-flow-removal.md)
is implemented through `307340d`. Native review pass08 against pinned dev
`e46bf64` exited0 with no findings; it independently passed unit, program and
terminal tests. Review log `/tmp/mch-cli-020-review.log`; implementation log
`/tmp/mch-cli-020-code.log`. Next: publish this compact documentation handoff,
merge020 through the helper, verify actual dev SHA, then branch/write P2 projects
and typed transport. No future squash SHA is asserted.

## Contracts now established

- Startup requires local `.mch/config.yaml`, independently of Flow resources,
  agent executable, workspace creation or branch state. Injected roots bypass Git
  lookup. Only the five approved brief/spec prompts remain under
  `.mch/default/prompts/`; obsolete Flow YAML, scripts, Makefile and prompts are
  removed. `.mch/config.yaml` and existing `.mch/tmp/` data are untouched.
- Old Flow controller, session/workspace restoration, branch automation,
  `/reference` and assign-flow adapter are gone. New fixed workflow belongs to
  P8/P9. Ordinary API/editor behavior remains; unimplemented fake actions do not
  claim saves. Backend HTTP migration still belongs to P2–P7.
- Project selection changes memory immediately and serializes asynchronous atomic
  config saves. Prior file contents survive failure; existing permissions survive
  replacement. Orderly quit drains active/queued saves; failure cancels exit and
  remains visible. User can retry or explicitly exit afterward.
- Successful change creation retains its ID after a later type-update failure.
  Existing document editors require loaded details, preserve exact bytes/tabs/
  long text, and make unchanged edits no-ops. Raw unsaved drafts survive failed
  writes and editor reopen. Clipboard updates synchronize representable drafts;
  lossy drafts remain editable through Ctrl+E. Editor output/retries are literal
  data; ordinary forms dispatch only exact recognized commands, preserving other
  slash-prefixed input. All these review regressions have meaningful tests.
- Existing useful program/PTY drivers, configuration/navigation/rendering/editor
  assertions were reused. Removed Flow assertions are mapped to their removed
  behavior in the ledger; future process/progress tests are assigned to P8/P9.
  The removed Flow-only process adapter's pipe race is recorded there, not hidden
  as a passing retained implementation.

## Final P1 verification and exact coverage

Final production code is `307340d`; subsequent checkpoint edits are documentation.
All required scenarios pass without skips. Raw counts below are independent:
unit **2220/2727 (81.4081%)**, terminal **1427/2727 (52.3286%)**. Strict >95% and
>90% gates fail honestly. These are final-rebuild targets; numerical shortfalls
alone do not block authorized intermediate dev merges. Overall rebuild remains
incomplete, including the static baseline listed below.

| Command actually run on final implementation/review fixes | Exit | Result |
| --- | ---: | --- |
| Scoped formatter on touched Go files | 0 | Formatting applied and diff inspected. |
| `cd cli && go test ./internal/app -count=1` | 0 | Unit regressions pass. |
| `cd cli && go test ./integration -run '^TestCLIProgramEditorSaveAndFailure$' -count=1` | 0 | Exact form/editor data and retry regression passes. |
| `make -C cli check` | 2 | Baseline format/lint only; vet, unit race, architecture, 36 Python tooling tests and Go fixtures pass. |
| `make -C cli coverage` | 2 | Complete measurement; strict unit threshold fails. |
| `make -C cli deps-audit` | 0 | No vulnerabilities found. |
| `make -C cli integration-test` | 0 | Complete-program and startup suites pass. |
| `make -C cli integration-coverage` | 2 | Complete program/startup/real PTY campaign; strict terminal threshold fails. |
| Documentation tests and working-diff whitespace checks | 0 | Links and documentation checks pass. |
| Native review pass08 against pinned dev | 0 | No findings; unit/program/terminal tests independently pass. |

| Production package | Unit covered/total | Terminal covered/total |
| --- | ---: | ---: |
| `cli/cmd/mch` | 0/3 | 1/3 |
| `cli/internal/app` | 1366/1635 | 781/1635 |
| `cli/internal/changes` | 479/601 | 382/601 |
| `cli/internal/dto` | 0/0 | 0/0 |
| `cli/internal/epics` | 4/5 | 4/5 |
| `cli/internal/help` | 6/6 | 6/6 |
| `cli/internal/navigation` | 23/40 | 6/40 |
| `cli/internal/projects` | 105/116 | 69/116 |
| `cli/internal/styles` | 0/0 | 0/0 |
| `cli/internal/testcases` | 5/6 | 5/6 |
| `cli/internal/ui` | 12/13 | 8/13 |
| `cli/pkg/client` | 220/302 | 165/302 |

Unit leaves507 statements uncovered, terminal1300. Largest gaps are app
forms/effects/errors, changes parsing/rendering, and client decoding/error paths;
startup remains0/3 unit. Every production package, including zero-statement
packages, stays inventoried. Detailed blocks/functions, source hashes, command
journals and separate profiles are in `cli/.coverage/{unit,integration}/`.
Both status files report completed measurements with failing numerical gates.
Final outer logs are `cli/.coverage/review07/`; earlier failed runs and review
reproductions remain in their pass directories. The initial P1 stale-frame test
race was repaired and repeated under -race; no failed run established coverage.
No live backend, database or real-agent campaign was run; collaborators are
aligned test fakes. Go1.26.8-X:nodwarf5, golangci-lint2.13.1, govulncheck1.7.0 and
socat1.8.1.1 were used. Docker was not rerun in P1 because toolchain/dependencies
are unchanged; P0's mounted-file Permission denied remains an environment blocker.

## Remaining baseline and queue

Fourteen untouched baseline formatting files: cmd/mch/main.go;
app/clipboard.go and project_save.go; changes/model.go, view.go, view_test.go;
epics/api.go and model.go; projects/api.go, model.go, navigation.go, view.go;
testcases/api.go and model.go. Revive reports baseline package-comment findings
(the sampled package list varies; final run includes dto, styles and ui).
Repair these in owning feature/adapter passes and final cleanup. P1 resolved the
forbidden agent→changes dependency and obsolete Flow missing-file test failures.
New regressions must be repaired; no assertions or coverage gates may be weakened.

P2 projects/typed HTTP transport is next, then P3 epics, P4 changes, P5 testcases,
P6 documents, P7 configs/health, P8 brief clarification, P9 spec review/fix,
P10 combined acceptance, followed by R1–R6 evidence-based cleanup categories.
The root remains a shell; feature behavior belongs in its feature. Continue the
factory through every bounded spec/review/dev merge, skipping empty cleanup passes.
Preserve the P1 editor/config contracts when migrating backend DTOs and ownership.

P0 history is retained in Git at dev `e46bf64` and `/tmp/mch-cli-019-*.log`:
tooling36 plus architecture/harness, vet/race/deps pass; unit3060/4021 and
terminal1796/4021 completed below target, baseline static/Flow failures recorded.
Its review repaired Docker input mounts, streaming diagnostics, Linux-owned
subprocess-tree cleanup, and SIGINT/SIGTERM/repeated-cancellation handling. Keep
those safeguards and independent all-production measurement throughout.
