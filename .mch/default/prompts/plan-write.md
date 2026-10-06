# Plan Writing

- Read the supplied brief (path relative to the repository root) and follow
  `.mch/default/templates/plan-template.md` for structure and formatting.
- Check relevant repository contracts and references. Resolve planning gaps, ambiguities,
  and contradictions through focused questions, one at a time, waiting for each answer.
  Do not guess or repeat answered questions. Resolve all planning questions before saving.
- Preserve every brief detail and clarification answer without expanding scope unless
  explicitly requested. Map each to a specification item or shared constraint.
- Use concise, simple language. Give each specification a cohesive, testable outcome,
  stable identifier, and clear scope. Assign each behavior one owner and explain dependencies
  without cycles, listing prerequisites first. Leave detailed requirements and testcases
  to specification writing; omit Q&A transcripts.
- Verify complete coverage, consistent scope, and valid dependencies. Save the complete
  plan as `plan.md` beside the brief, overwriting any existing plan. Preserve other files.
- Read the saved plan back and verify completeness and template compliance. On success,
  output exactly `Done.` without printing the plan or adding commentary.
- If clarification is needed, ask and wait; do not save an incomplete plan. For missing or
  unreadable inputs, or save or verification failures, output only `Error: <reason>`.
  Never output `Done.` on failure.
- Apply these rules to revisions. Write only the plan; do not write specifications,
  implement code, or perform Git publication.

[brief-file-path.md]
