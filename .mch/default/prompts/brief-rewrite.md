# Brief Rewrite

- Read the brief from the supplied file path. Improve its wording and organization using
  simple, understandable language without changing its meaning.
- Preserve all specified functionality, scope, rules, constraints, examples, and decisions.
  Do not add requirements, expand scope, or fill gaps with assumptions.
- Do not ask clarification questions. Leave missing information, ambiguities, contradictions,
  and open questions unresolved for the specification-writing step. Preserve the relevant
  wording when rewriting it would require choosing an interpretation. These unresolved
  issues do not block rewriting and saving the brief.
- The first line of the brief must be a level-one heading: `# [Title describing the change]`.
- Limit each line of the rewritten brief to 100 characters.
- Keep the brief concise without omitting specified details.
- Before saving, check that no specified details were lost or changed and no assumptions
  were introduced. Save the rewritten brief to the supplied file path, replacing its contents.
- Read the saved file back and verify that it preserves the original details and meaning
  and follows these formatting rules.
  On success, output exactly `Done.` without printing the brief or adding commentary.
- If the path or brief is missing or unreadable, or saving or verification fails, output
  only `Error: <reason>` with a concise explanation. Do not output `Done.` on failure.
- Apply these rules to later revisions. Edit the brief only; do not write a specification,
  implement code, or perform Git publication.

[brief-file-path.md]
