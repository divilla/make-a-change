Do not use any skills for this request.

Write an implementation-ready Change Spec from the Definition, initial code, and relevant branch work.

## Files

- Input definition: `/stg-tmp-dir/input.md`
- Output spec: `/stg-tmp-dir/output.md`
- Change type options: `/def-dir/prompts/change-types.md`

## Instructions

Read, in this order:

1. `/stg-tmp-dir/input.md`
2. `/def-dir/prompts/spec-file-structure.md`
3. `/def-dir/prompts/change-types.md`
4. `AGENTS.md`, when present

### Change Type Options

Load the complete list of allowed Change type slugs from
`/def-dir/prompts/change-types.md`. Require the file to begin with `# Change Types`, followed by
one or more Markdown list items formatted as `- <slug>`. Treat those list-item values, in file
order, as the complete set of allowed Change type options.

Select one or more listed slugs that best describe the intended Change and format them according
to `/def-dir/prompts/spec-file-structure.md`. Always include a populated `Types:` line in the new
Spec. Use only slugs from the loaded file; do not use a hardcoded fallback or invent a slug.

If the file is missing, unreadable, malformed, or contains no slugs, use the Clarification Gate
instead of writing the Spec.

### Sources and Authority

The Definition supplies the requested direction, and the code that existed when the Change began supplies
the starting behavior and constraints. The Spec must express the desired future state and give
strict instructions for what must change and how the Change must be conducted.

Inspect code as the source of truth for current behavior. Inspect the complete branch state,
including:

- the committed diff against the workflow base branch
- staged and unstaged diffs
- relevant untracked files
- tests and verification evidence relevant to the Change
- published PR metadata, when a PR exists

Include relevant behavior already applied manually, by an agent, or by another process. Do not
assume implementation has not started. If a PR exists, use it as a summary and evidence source, but
resolve descriptions of current behavior from code and tests.

Use targeted search to inspect relevant code and tests. Documentation is outside the default Change
Flow: do not inspect it by default or add documentation work to the Spec unless the Definition or user
explicitly requests that work. Read a retained ADR or operational runbook only when it is explicitly
cited by the Definition or user; it never overrides code.

### Writing Contract

- Use `/def-dir/prompts/spec-file-structure.md` exactly. Do not add, remove, rename, or reorder top-level sections.
- Account for every meaningful Definition item. Incorporate each item as a Goal, Scope item, Requirement, Non-Goal, Design Note, QA Test Case, Review Focus item, or Follow-Up. Ask for clarification before writing when an item cannot be classified confidently.
- Account for every relevant change already present in the branch, including work applied before
  the Spec was written, without treating applied work as correct merely because it exists.
- Describe intended final behavior in Requirements without adding implementation statuses, completion markers, or a change diary.
- Use existing implementation to make Requirements and Design Notes concrete, but do not assume that existing code is correct merely because it exists.
- Preserve fenced code blocks from the Definition when they remain part of the intended Change. Reformat them only as needed to meet the 100-character line limit without changing their meaning or behavior. Ask for clarification when that is not possible.
- Keep every line in the generated Spec within 100 characters.
- Do not invent product decisions, behavior, API contracts, database behavior, verification results, QA expectations, or new scope.
- Do not claim verification passed unless evidence shows it ran successfully.
- When the Definition conflicts with existing branch work on the intended final state, ask which behavior
  the Change must deliver.

### Clarification Gate

Before writing the Spec, stop and output only concise clarifying questions if:

- the scope or intended observable behavior has materially different interpretations
- existing branch work cannot be classified as part of or unrelated to the prospective PR
- the Definition and existing implementation conflict on a product decision
- API, persistence, CLI, validation, or failure behavior required for implementation is unclear
- valid Change type slugs cannot be determined
- any Definition statement is unclear, contradictory, or of uncertain intent

Use this format:

```text
Questions:
1. First question?
2. Second question?
```

Number every question and keep each question line within 100 characters.

### Quality Bar

The final Spec must satisfy:

- `Goal`: one or more observable end states intended for the Change.
- `Scope`: the intended Change plus relevant branch work already applied.
- `Requirements`: testable final behavior, visible contracts, persistence expectations, and boundaries.
- `Non-Goals`: intentionally excluded adjacent work, or `- None.`.
- `Design Notes`: established implementation decisions, constraints, and assumptions.
- `Verification`: realistic repository-supported commands with claims limited to available evidence.
- `QA Test Cases`: behavior-focused scenarios covering important paths and boundaries.
- `Review Focus`: the riskiest intended behavior and contracts.
- `Follow-Ups`: useful work outside the prospective PR, or `- None.`.

Write the completed Spec to `/stg-tmp-dir/output.md`, intentionally overwriting the file if it already exists. Do not modify any other file.

## Final Response

After successfully saving `/stg-tmp-dir/output.md`, output exactly:

Done.
