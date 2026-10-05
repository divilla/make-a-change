# CLI Change Creation, Agent Flow, and Testcase Synchronization

## Goal

Create changes from editor-written briefs, run interactive brief rewriting followed by spec writing,
and synchronize saved spec testcases while preserving document history and checked testcase state.

## Scope

- In scope: New-change editing, removal of ChangeCreateScreen and other CLI agent flows, brief
  rewriting, spec writing and interactive resumption, and testcase synchronization after every CLI
  spec save.
- Out of scope: Spec review and fix, backend contract changes, and unrelated changes to ordinary
  document or change actions. The workflow does not implement code or perform Git publication.

## Requirements

- Start this flow after a successful user save of a new change's initial brief or an edited brief
  for an existing change. Agent-authored document saves must not start another flow.
- On `/new-change`, generate `ref_uuid` in the CLI and create an empty file at
  `<system-temp-dir>/mch/<ref_uuid>/brief.md`. Use that UUID for change creation.
- Open that file using `editor` configured in repository-local `.mch/config.yaml`. If no editor is
  configured, use the existing system-editor fallback. Read the file after the editor exits
  successfully.
- Require a nonempty level-one Markdown title (`# Title`) on the first nonblank line. Body text is
  optional: `# Add DB migrations tool to the project` alone is valid. Reject an empty brief or
  invalid title on ChangesListScreen without creating a change or starting an agent.
- Create the change through the API with the selected project, generated UUID, heading title, and
  full brief. After creation succeeds, immediately start rewriting with the same file, without a
  creation screen or another confirmation. Delete ChangeCreateScreen and all code related to it.
- For an existing change, write its saved brief to `<system-temp-dir>/mch/<ref_uuid>/brief.md`,
  using the change's reference UUID. For a new change, reuse the editor file. Record the file's
  modification time before starting interactive Codex.
- Read repository-relative `.mch/default/prompts/brief-rewrite.md` into memory and replace
  `[brief-file-path.md]` with the temporary brief path in memory. Run `codex -C <repo-root>
  '<parsed-prompt>'`; the prompt argument is the resulting text, not a file path.
- After interactive rewriting exits successfully, compare the brief file's modification time with
  its recorded value. If unchanged, skip both the new brief version and spec writing. Use
  modification time to detect rewriting; no explicit completion response is required.
- If the brief modification time changed, read the file and append an agent-authored brief version
  through the API. Start spec writing only after that save succeeds.
- Read repository-relative `.mch/default/prompts/spec-write.md` into memory and replace
  `[brief-file-path.md]` with the same temporary brief path. Run `codex exec -C <repo-root>
  '<parsed-prompt>'` with the resulting prompt text.
- Show AgentExecScreen during non-interactive spec writing and stream command output in color as it
  arrives so the user can monitor progress.
- After a successful non-interactive exit, inspect the session's final output. If it is `Done.`,
  read `<system-temp-dir>/mch/<ref_uuid>/spec.md`, compare it with the current active spec through
  the API, save only differing content, synchronize testcases, and return to ChangeDetailsScreen.
- If the successful non-interactive run's final output is not `Done.`, resume that session
  interactively using `codex resume <codex-session-id>`. Use the ID from that run; if unavailable,
  stop and report the error without attempting resumption.
- After a successful interactive exit, read `spec.md` from the current operation's temporary
  directory and fetch the change's active documents through the API. Compare its content with the
  active spec, trimming outer whitespace as the backend does. If no active spec exists or content
  differs, append an agent-authored spec version. If equal, retain the active version without an
  insertion. Synchronize testcases and return to ChangeDetailsScreen in either case. This path
  requires neither an explicit `Done.` response nor a change to the file's modification time.
- If `spec.md` is missing after spec generation, return to ChangeDetailsScreen without saving a
  spec. Show the exact text "Error generating `spec`" on one line immediately above the footer.
- A failed active-spec read stops persistence and testcase synchronization and retains local drafts.
- After every successful CLI spec save, including manual edits and both agent paths, extract the
  saved spec's `## Testcases` section. Use the template's single-line bullets containing an action
  and expected result. Each bullet's complete scenario text, without its list marker, corresponds to
  one database testcase.
- Validate the entire Testcases section before changing testcases. If missing, empty, or malformed,
  leave all existing testcases unchanged, retain the saved spec, and show a testcase error on
  ChangeDetailsScreen.
- For a valid section, read only the current change's existing testcases through the API. Match
  complete scenario text exactly, without fuzzy matching. Preserve matching records unchanged,
  including IDs and checked state.
- Delete testcase records absent from the spec and excess duplicate records through the API. Create
  every unmatched spec occurrence through the API with unchecked state, so scenario text and
  occurrence counts match the spec completely.
- Match duplicate occurrences individually, keeping checked records before unchecked ones. If the
  spec lists a scenario once and the database has checked and unchecked copies, keep the checked
  copy and delete the unchecked copy.
- Automatically return to ChangeDetailsScreen when synchronization finishes and show synchronization
  errors there. Preserve and visibly report partial persistence if synchronization fails partway.
- If a required file or prompt cannot be read or written, an editor or Codex process fails or is
  cancelled, or an API operation fails, stop, report the error, retain useful local drafts, and skip
  later operations. Preserve writes that succeeded and make partial persistence visible, including
  partial testcase synchronization.
- A non-interactive Codex process error stops the flow even when a session ID is available; it must
  not trigger interactive resumption.
- Keep initial brief creation on `POST /api/v1/change/create`. Read active documents with `POST
  /api/v1/doc/list-active`; append rewritten briefs and generated specs with `POST
  /api/v1/doc/insert`, the change ID as `ref_id`, `ref_table: change`, and `doc_type: brief` or
  `spec`. Use `agent_edit: true` for agent-authored versions and human provenance for manual edits.
  Keep the original user-authored brief in document history.
- Require the selected project's configuration to support the required document types. Report
  unsupported types without inventing fallback types or silently changing configuration. Preserve
  backend rejection of blank document bodies and trimming of outer whitespace.
- Synchronize through `POST /api/v1/test-case/list` with `change_id`, `/api/v1/test-case/delete`
  with `id`, and `/api/v1/test-case/create` with `change_id` and `scenario`. Preserve backend
  scenario trimming and the creation default `done: false`; do not change these contracts.
- Follow the CLI architecture's boundaries, repository-root resolution, document persistence, and
  safe scratch-file ownership. Cleanup must validate ownership and remove only files created by the
  operation. Refused directory reuse must leave existing files intact; do not replace an unrelated
  regular file at an expected directory path.
- Remove all other agent flow code remaining in the CLI, including structured brief clarification
  and its JSON/question-answer protocol. Preserve ordinary document and change actions except for
  the creation-screen removal and save behavior specified here. Defer the architecture's broader
  spec review/fix sequence.

## Testcases

- Run `/new-change` → the editor opens an empty temporary `mch/<ref_uuid>/brief.md` file.
- Set `editor` in `.mch/config.yaml` and run `/new-change` → that editor opens the brief.
- Omit `editor` and run `/new-change` → the existing system-editor fallback opens the brief.
- Save `# Add DB migrations tool to the project` alone → a change is created and rewriting starts.
- Save blank lines followed by `# Valid title` → the heading supplies the created change's title.
- Save an empty brief → ChangesListScreen shows an error; no change or agent run is created.
- Save `## Wrong level`, `# `, or text before `# Title` → title error; no change or agent run.
- Save `# Title` with body text → the selected project, generated UUID, title and full brief save.
- Complete new-change creation → rewriting uses the editor file immediately, without confirmation.
- Save an existing change's edited brief → rewriting uses its saved brief and reference UUID path.
- Save an agent-authored brief → no additional brief-to-spec flow starts from that save.
- Start rewriting → interactive Codex receives the repository root and substituted prompt text.
- Exit rewriting with unchanged modification time → no brief version or spec-writing run is added.
- Rewrite the brief and exit successfully → an agent brief version saves before spec writing starts.
- Touch the brief without changing its text and exit successfully → a brief version is saved.
- Start spec writing → non-interactive Codex receives the substituted prompt and repository root.
- Emit colored spec-writing output before exit → AgentExecScreen displays it as it arrives.
- Finish with `Done.` and a spec file → the spec saves, cases synchronize, and details open.
- Finish with other final text and a session ID → that session resumes interactively.
- Finish without `Done.` or a session ID → a missing-ID error stops the flow.
- Create `spec.md` while resumed and exit successfully → the spec saves and cases synchronize.
- Modify an existing spec while resumed without `Done.` → the spec saves and cases synchronize.
- Exit a resume without creating a spec → details show "Error generating `spec`" above the footer.
- Exit a resume with unchanged spec time and differing active content → the spec saves and cases synchronize.
- Exit with a spec equal to the active version, including outer whitespace differences → no new version is inserted; cases synchronize.
- Change the active spec while the agent runs → comparison uses the active version fetched after exit.
- Fail the active-spec read after generation → no spec inserts or cases synchronize; drafts remain.
- Manually save a spec with valid testcase bullets → cases synchronize and details open.
- Save a spec without Testcases → testcase error; the spec saves and cases stay unchanged.
- Save a spec with an empty Testcases section → error; the saved spec and existing cases remain.
- Save a section with a valid bullet then a malformed entry → error; no cases are mutated.
- Save Q with checked Q in the database → its ID, text, timestamps and checked state remain.
- Save Q where the database contains Q and R → Q remains and R is deleted.
- Save Q where no testcase matches → one unchecked Q testcase is created.
- Save R where the database contains similar Q → Q is deleted and unchecked R is created.
- Save Q once with checked and unchecked copies → checked Q remains; unchecked Q is deleted.
- Save Q twice with one checked database copy → checked Q remains; an unchecked Q copy is created.
- Save Q once with three database copies → one Q remains; checked copies take priority.
- Save change A's spec while change B has cases → only A's cases are synchronized.
- Fail testcase listing after spec save → details show the error; the spec and all cases remain.
- Fail sync after one mutation succeeds → details show partial persistence; later work stops.
- Fail change creation after editor save → error; draft remains and Codex does not start.
- Fail rewritten brief insertion → the original brief remains; spec writing does not start.
- Fail spec insertion → the draft remains; testcase synchronization does not start.
- Make a required prompt or scratch file unreadable or unwritable → error; later operations stop.
- Cancel or fail the editor or interactive Codex → error; useful drafts and prior saves remain.
- Fail non-interactive Codex with a session ID → error; no resume or spec save occurs.
- Fail resumed Codex after editing the spec → error; local draft remains without a spec save.
- Use a project without required brief/spec support → an error appears without a fallback.
- Inspect history after user save and agent rewrite → human and agent versions retain provenance.
- Put an unrelated file at the scratch directory path → an error occurs; that file remains intact.
- Refuse reuse of an unowned scratch directory → its existing files remain intact.
- Use ordinary document and change actions → they work with the specified save behavior changes.
- Inspect navigation → ChangeCreateScreen and structured clarification are absent.
- Finish spec generation and sync → the flow ends at details without spec review or fix.

## Notes

### Reference contract

- Repository paths below are relative to the resolved repository root, not the temporary brief
  folder.
- `docs/cli-architecture.md` defines CLI boundaries, root resolution, API persistence, and scratch
  ownership. Its structured clarification sequence is replaced here; its broader review/fix sequence
  is deferred.
- `.mch/default/templates/spec-template.md` defines Testcases as single-line `- action → expected
  result` bullets. Generated `spec.md` is beside the temporary `brief.md`.
- `.mch/default/prompts/brief-rewrite.md` rewrites the supplied file; no completion token is
  required. `.mch/default/prompts/spec-write.md` saves a complete spec and outputs `Done.` on
  success.
- `backend/internal/domain/change.go` defines ChangeCreateRequest. Current change creation stores
  the initial brief, returns HTTP 201 with the ID, and requires configured `backlog` and `brief`
  support. The CLI supplies the UUID rather than relying on backend generation.
- `backend/internal/domain/doc.go` defines Doc, DocListRequest, and DocInsertRequest.
  `backend/internal/doc/api.go` and `backend/internal/doc/service.go` govern active reads, append
  operations, configured types, required provenance, nonblank bodies, and outer-whitespace trimming.
- `cli/pkg/client/changes.go` provides CreateChange; `cli/pkg/client/documents.go` provides
  ActiveDocuments and InsertDocument. They return committed IDs without an implicit refresh.
- `backend/internal/domain/test_case.go` defines testcase IDs, scenario text, ownership, and done
  state. `backend/internal/testcase/api.go`, `backend/internal/testcase/service.go`, and
  `backend/internal/testcase/repo.go` implement list/create/delete. Listing returns HTTP 200,
  creation HTTP 201 with an ID, and deletion HTTP 204.
- `cli/pkg/client/documents.go` provides ListTestCases; `cli/pkg/client/http.go` provides
  CreateTestCase and DeleteTestCase. ListTestCases reads only the requested change's records.
- `db/init.sql` and `db/tests/test_case.sql` retain testcase state/count rules and default unchecked
  creation. The testcase service trims scenario outer whitespace.
- `cli/internal/app/config.go` resolves the Git repository root and loads `.mch/config.yaml`. It
  currently handles backend/project settings; configured editor support is a requested change.
- `cli/internal/app/editor.go` currently uses `$EDITOR`, falling back to `nano` when empty. Retain
  this fallback when the new local `editor` setting is absent.
- `cli/internal/changes/model.go` currently requires both a heading and body in ParseBriefStructure.
  This change replaces its body requirement for new-change briefs with the title-only rule.
- `cli/internal/agent/`, `cli/pkg/briefprocess/`, `cli/internal/app/brief.go`, and the P8 section of
  `cli/agents/cli-contracts.md` describe the structured clarification flow being replaced. Their
  JSON response and question/answer protocol do not govern this interactive flow.

### Terminology

- `<system-temp-dir>` is the operating system's temporary directory; `<repo-root>` is the resolved
  repository root. `<ref_uuid>` is the change reference UUID generated by the CLI for new changes.
- `<parsed-prompt>` is prompt text after in-memory path substitution. `<codex-session-id>`
  identifies the successful non-interactive spec-writing session to resume.
- Checked means `done: true`; unchecked means `done: false`.
- Q is the complete scenario text `Open item → Item is shown.`; R is `Open item → Item is hidden.`
  In testcase examples, each occurrence is one template-format bullet or one stored database record.
