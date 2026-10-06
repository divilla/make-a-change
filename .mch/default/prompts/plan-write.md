# Plan Writing

- Brief and Plan are the files below; resolve their paths relative to the repository root.
  Read Brief and follow `.mch/default/templates/plan-template.md` for structure and formatting.
- Check Brief and relevant repository contracts for five categories: gaps (missing information),
  ambiguities (unclear meaning), contradictions (conflicting requirements), decisions (choices
  requiring user input), and suggestions (optional improvements).
- Present the first four categories under Questions and improvements under Suggestions in one
  response using the format below. Number each section independently and omit empty sections.
  Request one reply answering all questions and accepting or rejecting each suggestion by number.
- Wait for that reply. Repeat only missing or unclear items, explaining what remains unresolved,
  plus any newly discovered items, in the same format. Request one combined reply each round.
  Retain resolved answers; do not guess, decide for the user, or include unaccepted suggestions.
- Write Plan only when no items remain unresolved. Preserve Brief details, incorporating answers
  and accepted suggestions. Map them to specification items or shared constraints; expand scope
  only with explicit user approval.
- Use simple, concise language. Give each specification a stable identifier, clear scope, and
  cohesive, testable outcome. Assign each behavior one owner; explain dependencies without cycles
  and list prerequisites first. Leave detailed requirements and testcases to specification writing.
  Omit Q&A transcripts.
- Verify coverage of Brief and answers, consistent scope, and valid dependencies. Save to Plan,
  overwriting it if present. Preserve other files. Read Plan back and verify template compliance
  and completeness, then output exactly `Done.` without printing Plan or adding commentary.
- For missing or unreadable inputs, or save or verification failures, output only `Error: <reason>`.
  Do not save an incomplete Plan or output `Done.` on failure.
- Apply these rules to revisions. Write only Plan; do not write specifications, implement code,
  or perform Git publication.

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
