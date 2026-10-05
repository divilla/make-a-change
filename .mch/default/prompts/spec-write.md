# Specification Writing

- Read the supplied brief and `.mch/default/templates/spec-template.md`. Use the template
  as the source of all specification structure, content, and formatting rules.
- Preserve the brief and clarification answers: functionality, scope, rules, constraints,
  examples, and decisions. Do not add requirements or expand scope.
- Check relevant repository contracts and references. Distinguish existing behavior from
  requested changes; do not invent contracts or decisions.
- Resolve missing information, ambiguities, and contradictions through focused questions,
  one at a time, waiting for each answer before asking the next. Do not guess answers.
- Write concisely in simple language. Before saving, verify that all brief items and answers
  are covered consistently and no unresolved decision prevents implementation or testing.
- Save the complete Markdown specification as `spec.md` beside the brief, overwriting any
  existing `spec.md`. Preserve the brief and unrelated files; do not save an incomplete draft.
- Read the saved file back and verify completeness and compliance with the template.
  On success, output exactly `Done.` without printing the specification or adding commentary.
- If required inputs are missing or unreadable, or completion or saving is blocked, explain
  the reason and/or ask the next clarification question. Do not output `Done.` on failure.
- Apply these rules to later revisions. Write the specification only; do not implement code
  or perform Git publication.

[brief-file-path.md]
