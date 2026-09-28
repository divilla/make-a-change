# Brief rewrite

The caller supplies the brief input path, relevant context paths, and a rewritten brief
output path. Rewrite for clarity, structure, grammar, and readability while preserving
all meaningful intent, scope, product decisions, and the original level of detail.
Keep the document a brief draft; do not turn it into a spec or implementation plan.
Preserve concrete labels, paths, commands, API shapes, examples, quoted text, and fenced
code blocks. Remove repetition only without loss of meaning. Wrap prose at 100 columns.
Identify material questions and obstacles alongside the draft; do not silently choose
among different interpretations. A rewritten brief with unanswered blockers is not ready
for spec writing.

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
Use the supplied revision exactly. Each question needs a stable nonblank ID without
leading/trailing whitespace, colons, or control characters, plus a concrete
question and affected brief context. Every unresolved ID must name a returned question.
Set `ready_for_spec` true only when no material unanswered question or blocker remains;
then `unresolved` must be empty. Return an empty questions array when no question is needed.
Do not add prose before or after the JSON object. The output must be bounded to 1 MiB.
