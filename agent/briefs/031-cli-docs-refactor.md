# CLI Docs Refactor

- Align CLI fully with all changed and added endpoints in backend
- All relevant backend changes are in previous commit `Implement change 030-backend-docs-refactor`

## Docs

Docs must work like this:
- when you load project it will refer to config where docs are specified like:
  - project_docs: [docs...]
  - epic_docs: [docs...]
  - change_docs: [docs...]
- projects can have 0 or 1 prd attached, 0 or 1 plan, etc
- `comment` is the only doc that can be attached 0..n times to each project, epic or change

## All Docs except comments
- Brief is always displayed as it is currently implemented: Brief: up to 16 lines truncated - <return> enters edit mode
- All but brief must be displayed:
- Docs: <list-all-doc-types-except-brief-and-comments>
  - `spec [<green-check-if-exists>] 2016-10-03 00:52` or `spec [ ]` if document doesn't exist
  - <return> opens edit mode
  - <del> deletes document if exists - if not <del> does nothing
    
## Comments
- Comments are printed in max 3 lines without line wrap
- If comment is longer <return> opens edit mode where it can be red in its entirety
- Fourth comment line shows date: `2016-10-03 00:52`
- Comments label on ChangeDetailsScreen is `Comments`
- Menu item is added on first position /new-comment
- There can be multiple comments on single Change
- <return> edits comment
- <del> deletes comment
