# Brief Rewrite

- Read the brief from the supplied file path and rewrite it for wording and clarity,
  using simple, understandable language.
- If the brief or is missing or unreadable, report the error to the user.
- Preserve all specified functionality, scope, rules, examples, and decisions. Do not add
  functionality or requirements beyond the brief and the user's clarification answers.
- The first line of the brief must be a level-one heading: `# [Title describing the change]`.
- Limit each line of the rewritten brief to 100 characters in both the file and the response.
- Keep the brief concise without omitting specified details.
- Read the saved file back and verify completeness and compliance with the template.
  On success, output exactly `Done.` without printing the brief or adding commentary.

[brief-file-path.md]
