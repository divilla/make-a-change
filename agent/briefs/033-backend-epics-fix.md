# Backend Epics Fix

## DB Changes

- vw_epic_list_inactive is removed from db and active col added to vw_epic_list
- vw_change_list_inactive is removed from db and active col added to vw_change_list

## Backend

- `Active` pointer field is added to `EpicListRequest`
- Now you can call api/v1/epic/list with `active` set to query active and inactive or query with active = null to list all epic
- New endpoint `update-active` is implemented

- `Active` pointer field is added to `ChangeListRequest`
- Now you can call api/v1/change/list with `active` set to query active and inactive or query with active = null to list all epic
- New endpoint `update-active` is implemented

- Check all the code changes in this branch to note what was change
- Use this changes as a reference
- Implement entire code what is missing
- Run all the tests and make them pass

## Scope and desired behavior

Complete this migration in the backend only. The unified epic view and
EpicListRequest.Active are already started on this branch; finish the
backend implementation so it works with the changed database schema.

POST /api/v1/epic/list must return:
- active: true — active epics only.
- active: false — inactive epics only.
- active: null or omitted — all epics.

POST /api/v1/change/list must return:
- active: true — active epics only.
- active: false — inactive epics only.
- active: null or omitted — all epics.

Preserve project scoping, existing response fields, and ordering.
Epic details must work for both active and inactive epics.
Remove remaining backend references to the retired inactive view and
obsolete ListInactive code. Keep /epic/list-inactive retired.

Do not change CLI code or decide its future browsing behavior in this task.
The effect of omitted active on current CLI callers is acknowledged;
CLI adaptation is a separate follow-up and does not block this backend work.

Add meaningful tests for these behaviors, update backend contract
documentation, and run the backend checks required by AGENTS.md.
Report failures and coverage shortfalls honestly.
