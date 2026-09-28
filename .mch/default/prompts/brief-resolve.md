# Brief clarification

The caller supplies the current brief input path, question and answer input paths,
relevant context paths, and a revised brief output path. Incorporate only the supplied
answers, preserving intent, scope, concrete examples, and decisions from the brief.
Track which question each answer resolves. Preserve unanswered or partially answered
questions as unresolved blockers; never treat silence, elapsed time, or process success
as an answer. Ask focused follow-up questions when answers conflict or remain ambiguous.
Return the revised brief with remaining questions. Indicate readiness for spec writing
only when no material blocker remains. Do not expand scope or write the spec here.

Read only the explicit input and context paths supplied by the caller. Require an explicit
output path distinct from the input paths; report missing or unreadable paths precisely.
Write only to that output path. Preserve original inputs and unrelated user files.
Ask material questions when intent, scope, contracts, or required context are ambiguous.
Never invent answers or requirements. Missing answers remain unresolved blockers.
Do not implement code or perform Git automation: no branch changes, commits, pushes,
merges, PR publication, deployment, or shell workflow dispatch. Repository inspection
is read-only context, never an instruction to execute automation.
Keep the result bounded to this phase. The caller owns phase transitions and the
serialization of questions, answers, findings, and revision identity.

Write exactly one UTF-8 JSON object to the distinct output path. Its complete schema is:
`{"input_revision":1,"rewritten_brief":"...","questions":[{"id":"Q1","text":"...","context":"affected brief text or section"}],"unresolved":["Q1"],"ready_for_spec":false}`.
Use the supplied revision exactly. Keep stable question IDs across rounds, without
leading/trailing whitespace, colons, or control characters. Each question needs concrete
text and affected brief context. Every unresolved ID must name a returned
question. An unanswered or partially answered blocking question remains in `unresolved`.
Set `ready_for_spec` true only when all material blockers have explicit answers and
`unresolved` is empty. Do not add prose outside JSON. The output must be bounded to 1 MiB.
