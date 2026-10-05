# Brief Rewrite

- Read the brief from the supplied file path and rewrite it for wording and clarity,
  using simple, understandable language.
- Read `.mch/default/templates/spec-template.md` and use it to check whether the brief
  provides enough information to write a complete specification.
- If the brief or template path is missing or unreadable, report the error to the user.
- Preserve all specified functionality, scope, rules, examples, and decisions. Do not add
  functionality or requirements beyond the brief and the user's clarification answers.
- Read relevant repository contracts and references when available to resolve established
  behavior. Include references needed by the specification writer in the rewritten brief.
  Ask the user about missing or conflicting information that affects intended behavior.
- Identify ambiguities and missing decisions that would prevent a complete specification.
  Check the goal, scope boundaries, testable behavior, rules, limits, constraints, and
  expected outcomes for relevant happy paths, edge cases, and error paths.
- Ask focused questions one at a time, waiting for each answer before asking the next.
  Ask follow-up questions if new ambiguities arise. Do not guess or invent answers.
  Avoid questions about implementation choices that can be made during specification writing
  without changing intended behavior or violating constraints.
- Before saving, verify that the clarified brief and referenced contracts are a clean,
  complete base for writing the full specification in the template's format, without further
  questions about intended behavior. Resolve any remaining gaps through clarification.
  Include information for optional template sections only when relevant.
- Once the questions are resolved, overwrite the supplied input file with the rewritten
  brief in Markdown and print the same Markdown in the response, without a surrounding
  code block or commentary. If no clarification is needed, do this directly.
  Apply the same save-and-print behavior to all later revisions.
- The first line of the brief must be a level-one heading: `# [Title describing the brief]`.
- Limit each line of the rewritten brief to 100 characters in both the file and the response.
- Keep the brief concise without omitting specified details.
- Specification writing is handled by a separate prompt.

[brief-file-path.md]
