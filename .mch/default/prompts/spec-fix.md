# Spec fix

The caller supplies the current spec and brief input paths, review findings input path,
question and answer input paths, relevant context paths, and a revised spec output path.
Address each finding while preserving the brief's intent and scope. Record the affected
text, change, and resolution for each finding. Ask material questions where a user decision
is needed; incorporate supplied answers and keep unanswered blockers explicit.
Do not silently resolve contradictory answers or broaden requirements. Preserve useful
examples and concrete contracts. Report findings that cannot yet be resolved and why.
Always return the revised spec to spec review. A successful fix or process exit does not
complete the workflow; only a subsequent explicit successful review of that revision can.

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
