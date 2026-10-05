# CLI Change Creation, Agent Flow, and Testcase Synchronization

Start this flow after the user successfully saves a new change's initial brief or an edited
brief for an existing change. Agent-authored document saves do not start another flow.

## New-change editor flow

1. On `/new-change`, generate `ref_uuid` in the CLI and create an empty file at
   `<system-temp-dir>/mch/<ref_uuid>/brief.md`.
2. Open that file with the `editor` configured in `.mch/config.yaml`. If no editor is
   configured, use the default system editor.
3. After the user exits successfully, read the file. Reject an empty brief or an invalid
   title and show an error on `ChangesListScreen`; do not create a change or start the agent.
4. Require a nonempty level-one Markdown title (`# Title`) on the first nonblank line.
   Body text is optional: `# Add DB migrations tool to the project` is a valid brief.
5. Create the change through the API using the selected project, generated UUID, title from
   the heading, and full brief. After successful creation, start brief rewriting immediately
   using the same file. Do not stop on a creation screen or ask for another confirmation.

Delete `ChangeCreateScreen` and all code related to that screen.

## Brief rewriting

1. For an existing change, write the saved brief to
   `<system-temp-dir>/mch/<ref_uuid>/brief.md`. For a new change, reuse the editor's file.
   Record its modification time before interactive Codex starts. `<ref_uuid>` is the
   change's reference UUID, supplied by the CLI during new-change creation.
2. Load `.mch/default/prompts/brief-rewrite.md` from the repository into memory. Replace
   `[brief-file-path.md]` with the temporary brief's path in memory.
3. Run interactive Codex: `codex -C <repo-root> '<parsed-prompt>'`.
   `<parsed-prompt>` is the resulting prompt text, not a file path.
4. When the user exits successfully, compare `brief.md`'s modification time with the recorded
   value. If it has not changed, skip both the new brief version and spec writing.
5. If the modification time changed, read `brief.md` and save a new brief version through the
   backend API. Start spec writing only after this save succeeds.

Use the file's modification time to detect rewriting; no explicit completion response is
required from interactive brief rewriting.

## Spec writing

1. Load `.mch/default/prompts/spec-write.md` into memory. Replace `[brief-file-path.md]` with
   the same temporary brief path in memory.
2. Run non-interactive Codex: `codex exec -C <repo-root> '<parsed-prompt>'`.
3. Show `AgentExecScreen` during execution. Stream command output in color as it arrives so
   the user can monitor progress.
4. After a successful process exit, read the session's final output:
   - If it is `Done.`, read `<system-temp-dir>/mch/<ref_uuid>/spec.md`, save a spec version
     through the backend API, synchronize testcases, and return to `ChangeDetailsScreen`.
   - Otherwise, resume that Codex session interactively with
     `codex resume <codex-session-id>` so the user can inspect what went wrong.

Resuming requires the session ID from the non-interactive run. If it is unavailable, stop
and report the error rather than attempting to resume.

## After the resumed session

Record whether `spec.md` exists and its modification time before the resumed session starts.
When the user exits the resumed session successfully:

- If `spec.md` was created or its modification time changed during that session, read it,
  save a spec version through the backend API, synchronize testcases, and return to
  `ChangeDetailsScreen`.
  An explicit `Done.` response is no longer required.
- If `spec.md` was not created or modified during that session, return to
  `ChangeDetailsScreen` without saving a spec. Show the exact message
  ``Error generating `spec` `` on one line immediately above the footer.

## Testcase synchronization

After every successful CLI spec save, including manual edits and either agent save path,
extract testcases from the saved spec's `## Testcases` section. Use the template's
single-line bullets, each containing an action and expected result. Each bullet's complete
scenario text, without the list marker, corresponds to one database testcase.

Validate the entire section before changing testcases. If the section is missing, empty,
or malformed, show a testcase error on `ChangeDetailsScreen` and leave all existing
testcases unchanged. The spec remains saved.

For a valid section, read the change's existing testcases through the API and synchronize:

- Match complete scenario text exactly; do not use fuzzy matching.
- Keep matching database records unchanged, including IDs and checked state.
- Delete records absent from the spec and excess duplicate records.
- Add unmatched spec entries through the API; every new testcase must be unchecked.
- Match duplicate occurrences individually so database scenario text and counts match the
  spec completely. When choosing between duplicate matches, keep checked records first.
  For example, if the spec lists a scenario once and the database has checked and unchecked
  copies, keep the checked copy and delete the unchecked one.

Use only the current change's testcases. Automatically return to `ChangeDetailsScreen`
after synchronization finishes; show synchronization errors there.

## Failures

If a required file or prompt cannot be read or written, an editor or Codex process fails
or is cancelled, or an API operation fails, stop, report the error, retain useful local
drafts, and skip later operations. Preserve writes that already succeeded and make partial
persistence visible, including partial testcase synchronization. A non-interactive process
error stops the flow even if a session ID is available; it does not trigger resumption.

## Scope and existing contracts

- Remove all other agent flow code still present in the CLI, including the existing
  structured brief clarification flow. Preserve ordinary document and change actions,
  except for the creation-screen removal and save behavior explicitly changed here.
- Spec review and fix behavior is outside this change and will be addressed in later specs.
- Follow `docs/cli-architecture.md` for CLI boundaries, repository-root resolution, document
  persistence, and safe scratch-file ownership. Its broader review/fix sequence is deferred;
  this brief replaces its structured clarification sequence for this change.
- Use the current backend contracts in `backend/internal/domain/change.go`,
  `backend/internal/domain/doc.go`, and `backend/internal/doc/{api,service}.go`, with the
  CLI adapters in `cli/pkg/client/{changes,documents}.go`.
- `POST /api/v1/change/create` stores the initial user brief. Read active documents through
  `POST /api/v1/doc/list-active`; append rewritten briefs and generated specs through
  `POST /api/v1/doc/insert` with the change ID, `ref_table: change`, `doc_type: brief` or
  `spec`. Set `agent_edit: true` for agent-authored documents and retain human provenance
  for manual edits. Keep the original user-authored brief in document history.
- The selected project configuration must support the required document types. The backend
  rejects blank document bodies and trims outer whitespace. Do not change these contracts.
- Read `.mch/default/prompts/{brief-rewrite,spec-write}.md` and
  `.mch/default/templates/spec-template.md` for the agent's file and output contracts.
  Spec writing saves `spec.md` beside `brief.md` and outputs `Done.` on success.
- See `cli/internal/app/{config,editor}.go` for local configuration and editor handling,
  and `cli/internal/changes/model.go` for existing title parsing. Its body requirement is
  replaced by the title-only rule above for new-change briefs.
- Testcase contracts are in `backend/internal/domain/test_case.go`,
  `backend/internal/testcase/{api,service,repo}.go`, and
  `cli/pkg/client/{http,documents}.go`. Use `POST /api/v1/test-case/list` with `change_id`,
  `/delete` with `id`, and `/create` with `change_id` and `scenario`. These routes share the
  `/api/v1/test-case` prefix. Creation defaults to `done: false`; scenario text is trimmed
  by the backend. See `db/init.sql` and `db/tests/test_case.sql` for these database rules.
- `cli/internal/agent/`, `cli/pkg/briefprocess/`, `cli/internal/app/brief.go`, and the P8
  section of `cli/agents/cli-contracts.md` describe the existing clarification flow being
  replaced; its JSON response and question/answer protocol do not govern this new flow.
