# CLI Epics Fix

## Goal

Make epic browsing, epic selection, and change activation work with the current backend contracts.
Align the epic table and make inactive changes accessible through a persistent list filter.

## Scope

- In scope: Epic list layout, colors, deletion and activation; active-only epic assignment;
  change list filtering, deactivation and restoration; confirmed deactivation from change details.
- Out of scope: Backend or database changes, unrelated CLI behavior, and permanent change deletion.

## Requirements

- EpicsListScreen lists both active and inactive epics for the selected project.
- Render active epic rows in `Foreground` and inactive rows in `AccentRed`, including selected
  rows. Retain a visible selection highlight.
- Use the ChangesListScreen table layout for epics, with vertically aligned headers and values.
  Columns appear in this order: `ID`, `Name`, `DoneTC`, `Compl`, `Chngs`, `Active`.
  Use the existing table conventions for padding, display-width clipping, and terminal bounds.
- The epic `ID` column is 4 characters wide, with its header and values right-aligned.
- The epic `Name` column is 30 characters wide, with its header and values left-aligned.
- The epic `DoneTC` column is 6 characters wide and right-aligned. Display completed/total
  testcase counts, such as `2/8`, using the server's `done_tc` and `total_tc` values.
- The epic `Compl` column is 5 characters wide and right-aligned. Append `%` to the server's
  `completed` value, producing values such as `100%` and `33%`.
- The epic `Chngs` column is 5 characters wide and right-aligned, showing `change_count`.
- The epic `Active` column is left-aligned. Its value is empty for active epics and exactly
  `inactive` for inactive epics.
- On EpicsListScreen, Delete acts on the selected epic immediately, without confirmation.
  An epic attached to no changes is deleted; an epic attached to any change is set inactive.
  Refresh the list so a deleted epic disappears and a retained epic displays its inactive state.
- On EpicsListScreen, Space toggles the selected epic between active and inactive and refreshes
  its displayed state. Both states remain listed.
- The epic selector on ChangeDetailsScreen offers only active epics from the selected project.
  Preserve the existing option to clear the epic association.
- ChangesListScreen shows only active changes by default. `/inactive-filter` is the single
  command name for switching between active-only and inactive-only lists.
- Insert `/del-change` immediately below `/new-change` in the active changes list menu.
  In the inactive list, replace that item with `/undel-change` in the same position.
- Insert `/inactive-filter` immediately below `/find-filter` in the changes list menu, and
  display its label to the right of `/find-filter` in the filter summary at the top of the screen.
- Selecting `/inactive-filter` enables it and shows only inactive changes. Render its summary
  label in `AccentRed` while enabled. Selecting it again disables it, shows only active changes,
  and restores the same gray used by the other filter labels.
- Treat `/inactive-filter` like the existing persistent filters: retain it through opening and
  leaving details, returning to Main, reopening Changes, and switching projects. Apply the other
  list filters within the chosen active or inactive set; toggling this filter preserves them.
- `/clear-filters` clears the existing filters and disables `/inactive-filter`, returning to
  active changes. Navigation alone must not reset the inactive filter.
- On ChangesListScreen, Delete or `/del-change` for the selected change opens the bottom
  `Are you sure?` prompt with Yes/No choices. Yes sets the change inactive; No or cancellation
  leaves it unchanged. Refresh the current list after a successful deactivation.
- On ChangeDetailsScreen, `/delete` uses the same confirmation and sets the change inactive
  instead of permanently deleting it. No CLI change-deletion action may permanently delete
  a change or call `/api/v1/change/delete`.
- In the inactive changes list, Space or `/undel-change` sets the selected change active and
  refreshes the inactive list, removing the restored row. Space retains its existing restoration
  behavior; it is not a new deactivation shortcut for active changes.
- Use the current project-scoped list APIs: `/api/v1/epic/list` with `active` omitted or null
  for all epics, and `/api/v1/change/list` with explicit `active: true` or `active: false` for
  the selected change list. Do not use the retired `list-inactive` endpoints.
- Use `/api/v1/epic/delete` for epic deletion and its backend-owned deactivation fallback.
  Use the respective epic or change `update-active` endpoint with an explicit boolean to
  change active state. Preserve existing error reporting and committed-write refresh recovery.

## Testcases

- Open Epics with active A and inactive B in the project → both appear; other projects do not.
- View and select active A → its row uses `Foreground` and selection remains visible.
- View and select inactive B → its row uses `AccentRed` and selection remains visible.
- View epics with different name lengths → headers and values align in the Changes table layout.
- View epic IDs 7 and 1234 → `ID` and both values align right within a 4-character column.
- View epic names Alpha and Beta → `Name` and values align left within a 30-character column.
- View an epic with 2 done of 8 testcases → `DoneTC` shows `2/8`, right-aligned in 6 characters.
- View epics with completion 100 and 33 → `Compl` shows `100%` and `33%` in 5-character cells.
- View epics with 1 and 12 changes → `Chngs` and values align right within 5 characters.
- View active A and inactive B → `Active` is blank for A and left-aligned `inactive` for B.
- View an epic name longer than 30 characters → clipping preserves the following columns.
- Narrow the terminal on Epics → the table stays within the existing terminal layout bounds.
- Press Delete on an unreferenced epic → no prompt appears and the epic disappears.
- Press Delete on an epic attached to an active change → no prompt; the epic remains inactive.
- Press Delete on an epic attached only to an inactive change → the epic remains inactive.
- Press Space on an active epic → it stays listed, becomes red, and displays `inactive`.
- Press Space on an inactive epic → it stays listed in `Foreground` with an empty Active cell.
- Open the change epic selector with active A and inactive B → only A and the clear option appear.
- Start Changes with active A and inactive B → only A appears, using the active list request.
- Open the active Changes menu → `/del-change` appears immediately below `/new-change`.
- Inspect the menu and top summary → `/inactive-filter` follows `/find-filter` in both places.
- Select `/inactive-filter` → only B appears; the label is red and menu says `/undel-change`.
- Select `/inactive-filter` again → only A appears; its label is gray and menu says `/del-change`.
- Set a Find filter, then toggle inactive filtering → the Find query still filters the new list.
- Enable inactive filtering, open details, and return → inactive filtering remains enabled.
- Enable inactive filtering, visit Main, and reopen Changes → inactive filtering remains enabled.
- Switch projects with inactive filtering enabled → only the new project's inactive changes appear.
- Run `/clear-filters` in inactive mode → all filters clear; active changes appear.
- Press Delete on active change A → the bottom prompt shows `Are you sure?` and Yes/No choices.
- Choose `/del-change` on A → the same bottom confirmation appears.
- Confirm deactivation of A with Yes → A leaves the active list and appears in the inactive list.
- Choose No or cancel A's deactivation → A remains active and unchanged.
- Run `/delete` from A's details and choose Yes → A becomes inactive and remains stored.
- Run `/delete` from A's details and choose No → A remains unchanged.
- Press Space on inactive B → B leaves the inactive list and appears in the active list.
- Choose `/undel-change` on inactive B → B is restored and leaves the inactive list.
- Reject an activation request at the API → the CLI reports failure without claiming success.
- Fail the list refresh after a committed update → show the saved outcome and refresh error.
- Retry that failed refresh → reload the list without repeating the committed mutation.

## Notes

### Reference contract

- Brief: [Brief](../briefs/034-cli-epics-fix.md). `ChangeListScreen` in the brief refers to
  the existing `ChangesListScreen`.
- [CLI architecture](../../docs/cli-architecture.md) defines feature ownership, asynchronous
  operations, persistent filters, and refresh recovery. This change adds the fifth persistent
  filter and overrides the old inactive-list navigation that reset activity mode on return.
- [Backend epic requests](../../backend/internal/domain/epic.go) define `EpicListRequest`,
  `EpicUpdateActiveRequest`, and the completion and activity fields in `Epic`.
- [Backend epic API](../../backend/internal/epic/api.go) exposes POST `list`, `delete`, and
  `update-active`. List returns HTTP 200; successful mutations return HTTP 204 without a body.
- [Epic service](../../backend/internal/epic/service.go) already deactivates an epic when deletion
  fails because changes reference it. The CLI must use that contract, not infer deletion from
  `change_count` or a successful empty mutation response.
- [Backend change requests](../../backend/internal/domain/change.go) define `ChangeListRequest`
  and `ChangeUpdateActiveRequest`. For both list APIs, `active: true` selects active records,
  `active: false` selects inactive records, and omitted/null selects both states.
- [Backend change API](../../backend/internal/change/api.go) already exposes POST `list` and
  `update-active`. Updates require `id` and a non-null `active` boolean and return HTTP 204.
  Change list responses do not include an `active` field; use the requested list state.
- [Current change client](../../cli/pkg/client/changes.go) omits activity filtering on ordinary
  lists and calls retired `change/list-inactive` for inactive lists. Both need adaptation.
- [Current epic client](../../cli/pkg/client/epics.go) already loads all epics when activity is
  omitted. [Selectors](../../cli/internal/app/selectors.go) already exclude inactive epic options.
  Keep that existing assignment behavior working while adding epic activation.
- [Change table](../../cli/internal/changes/view.go) supplies the existing table layout and
  display-width clipping conventions. Epic metric values remain backend-owned.
- [Contract ledger](../../cli/agents/cli-contracts.md) contains historical `list-inactive` and
  permanent change-deletion references. Current backend handlers and these requirements govern
  the changed behavior. Backend deletion remains available outside this CLI scope.
- Epic-details `/delete` confirmation is existing behavior; the no-confirmation decision here
  applies to Delete on EpicsListScreen.

### Additional requirements

- Cover every requirement with meaningful unit tests and follow the CLI validation and completion
  policy in [AGENTS.md](../../AGENTS.md). Keep unit and terminal coverage evidence separate.
