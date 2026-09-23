Read the software change definition from `/stg-tmp-dir/input.md`.

Rewrite the definition for clarity, structure, grammar, and readability without changing its intent, scope, or product
decisions.

Write the rewritten definition to `/stg-tmp-dir/output.md`.

Rules:

- Preserve all meaningful user intent.
- Preserve the original level of detail.
- Keep the document a definition draft.
- Fix grammar, wording, ordering, and readability.
- Improve wording, organization, headings, and bullet structure where useful.
- Wrap non-code Markdown lines at 100 characters or fewer.
- Remove repetition when it does not remove meaning.
- Clarify vague wording only when the intended meaning is obvious.
- Preserve fenced code blocks exactly, including language tags and contents.
- Preserve concrete labels, names, paths, commands, API shapes, examples, and quoted text unless the input clearly
  contains a typo.
- Do not invent product decisions, implementation details, requirements, verification steps, or
  new scope.
- Do not add Change spec sections or turn the definition into a requirements document, implementation plan, or Change spec.
- Do not implement code.
- Do not edit any files except `/stg-tmp-dir/output.md`.

After writing the file, print exactly:

`Done.`
