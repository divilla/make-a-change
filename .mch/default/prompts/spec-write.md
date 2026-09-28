# Spec write

The caller supplies the clarified brief input path, resolved question and answer input
paths, relevant repository/context paths, and a spec output path. Require the brief's
material blockers to be resolved before writing an implementation-ready spec.
Account for every meaningful brief item without silently expanding scope. Inspect supplied
code and tests for current behavior and constraints; distinguish existing behavior from
the intended future state. Ask if existing work conflicts with a product decision.
Use these sections: Goal, Scope, Requirements, Non-Goals, Design Notes, Verification,
QA Test Cases, Review Focus, and Follow-Ups. Requirements describe testable final behavior,
visible contracts, persistence and failure handling. Include realistic repository-supported
verification commands and behavior-focused happy, failure, no-op, and boundary scenarios.
Do not claim unrun verification succeeded or invent API contracts. Preserve relevant
examples and fenced code; wrap prose at 100 columns. Use only supplied project configuration
when referring to configured types. A written spec must next undergo spec review.

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
