# Plan Writing

- Brief and Plan refer to the files listed below, with paths relative to the repository root.
  Read Brief and follow `.mch/default/templates/plan-template.md` for structure and formatting.
- Check Brief and relevant repository contracts for five categories: gaps (missing information),
  ambiguities (unclear meaning), contradictions (conflicting requirements), decisions (choices
  requiring user input), and suggestions (optional improvements).
- Put the first four categories under Questions and optional improvements under Suggestions.
  Present all current items in one response using the format below, numbering each section
  independently. Omit sections with no items; if neither has items, proceed with writing Plan.
- Ask the user to answer all questions and accept or reject each suggestion in a single reply,
  referring to the section and item numbers, then wait.
- Do not guess, do not make decisions instead of the user, or add unaccepted suggestions to scope.
  Resolve all items before saving Plan. Repeat questions and suggestions output if necessary.
- Keep clear answers and suggestion choices resolved. If an answer or choice is missing or
  unclear, repeat only unresolved items in the same Questions/Suggestions format, clarifying
  what is still needed. Omit empty sections and request one combined reply again.
- Include any newly discovered unresolved items in the next round. 
- Preserve every Brief detail and clarification answer without expanding scope unless
  explicitly requested. Map each to a specification item or shared constraint.
- Use concise, simple language. Give each specification a cohesive, testable outcome,
  stable identifier, and clear scope. Assign each behavior one owner and explain dependencies
  without cycles, listing prerequisites first. Leave detailed requirements and testcases
  to specification writing; omit Q&A transcripts.
- Verify complete coverage, consistent scope, and valid dependencies. Save the complete
  plan to Plan, overwriting it if it exists. Preserve other files.
- Read Plan back and verify completeness and template compliance. On success,
  output exactly `Done.` without printing Plan or adding commentary.
- If clarification is needed, ask and wait; do not save an incomplete Plan. For missing or
  unreadable inputs, or save or verification failures, output only `Error: <reason>`.
  Never output `Done.` on failure.
- Apply these rules to revisions. Write only Plan; do not write specifications,
  implement code, or perform Git publication.

Clarification response format (omit empty sections):

```text
Questions:
1. [Question]
2. [Question]

Suggestions:
1. [Suggestion]
2. [Suggestion]
```

Brief: `<dir>/brief.md`
Plan: `<dir>/plan.md`
