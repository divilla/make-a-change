# 011 — Backend validation error causes

## Authorized bounded repair

Implement on `change/011-backend-validation-causes` from freshly fetched
origin/dev after R1. This is an explicit follow-up error-contract repair found
by the post-P4 architecture audit, not the R2 duplicate cleanup category. R2 may
be recorded as no actionable findings independently. The user authorized the
whole sequential backend implementation/review/dev-merge sequence. Read current
AGENTS.md, docs/backend-architecture.md, this spec, backend contract ledger and
compact checkpoint; user excludes PRD, skeleton, CLI/frontend and unrelated docs.

All implementation/test/artifact/log writes stay under backend/. This committed
specification and top status of agent/backend-refactor-plan.md are authorized
documentation exceptions. No root scripts, AGENTS, database/architecture files,
application dependencies, public routes, SQL behavior, transactions or deployment
changes. Write the change-code implementation block to backend/implementation-log.md
and exclude it from counts. Factory owns commit/push, so implementation subprocess
must not commit/push/merge or start another pass. No concurrent checkout writers.

Coverage >95% unit and >=90% APIHydra statement targets remain final-result goals,
not intermediate blockers. Preserve strict gates, whole-production denominator,
separate unit/API/legacy profiles and actual evidence. Both gates passed after
P4; repair new regressions and report actual counts. No artificial coverage,
weakened tests or omitted packages. No stage/production promotion.

## Evidence and behavior to preserve

Project, epic and testcase API handlers run validate.Struct(req), but on failure
return HTTP(ErrModuleInvalidInput), discarding v.Errors. Change handlers retain
v.Errors through InvalidPayload but their tag-validation path does not preserve
ErrChangeInvalidInput identity. The safe400 messages are already correct.

change/service.go invalidPRURL calls url.Parse, reduces its returned *url.Error
to a bool and UpdatePRUrl returns ErrChangeInvalidInput. The safe public400 is
correct, but direct callers/logging cannot inspect the external parser cause.

Target architecture requires external causes to survive central classification.
Preserve all public statuses/messages, validation order, accepted input, exact
JSON envelopes, repository side effects and existing errors.Is semantic identities.
The repair must retain real validation/parser errors rather than invent dummy
causes or use test-only wrappers. Keep URL business rules in service and
wrapping/classification in internal/error. Services must never import Echo,
validate or pgx. Do not turn this into generic validation middleware/framework.

## Acceptance criteria

VC-01 Central semantic/cause handling: introduce or reuse one small public
internal/error function that retains a semantic invalid-input error plus its
external cause, both inspectable through errors.Is/As. It returns an ordinary
error, not an Echo HTTP error, so services remain transport-independent. Use
existing central wrapping conventions. Define and test straightforward nil
cause/semantic behavior, direct/wrapped causes and cause identity. No current
caller requires duplicate-wrapper detection; do not add an already-classified
optimization or prospective idempotence branches to the new helper. Nil cause should preserve the
semantic error; no semantic should preserve the original cause. Both nil must
remain nil. No broad generic error type registry or unused prospective functions.

VC-02 API validator causes: migrate every tag-validation failure in project,
epic, testcase and change APIs to the consistent central semantic-plus-cause
path, retaining the actual v.Errors. Use each module's existing invalid-input
sentinel and existing generic message (invalid project/epic/test case/change
payload) through central HTTP interpretation. Retain errors.Is to module sentinel
for existing caller contracts while adding errors.As to real validation errors.
Change's existing validation cause must not be lost while adding semantic
identity. Do not modify operation-specific c.Bind failure messages or their
existing cause wrapping. Pure business-validation failures with no external
error keep the sentinel; successful request/control flow remains unchanged.

VC-03 URL parser cause: change the private URL-validation interface only as
needed to return an ordinary semantic+cause error for a real url.Parse failure.
UpdatePRUrl must retain ErrChangeInvalidInput and expose *url.Error through
errors.As for malformed escapes/URL syntax. Wrong/non-HTTP(S) scheme or missing
host where Parse succeeds still returns the same invalid-input semantic error;
valid HTTP(S) schemes remain case-insensitive, with existing trimming and host
rules. Positive-ID/nonblank validation happens before parser/repository work as
before. Do not tighten URL syntax to match a different validator, reject existing
accepted userinfo/fragment/query forms, or change emptyURL behavior. One valid
repository call only; invalid input calls none. No Echo wrapping in service.

VC-04 Meaningful tests: each behavior above needs real-library evidence. Execute
actual validate.Struct through real API handler requests with failed ID/required
field tags, capture returned error before HTTP writing, and prove module sentinel
errors.Is plus errors.As to the real validator error type, preserving field/rule
information. Exercise all migrated handlers or a parameterized route matrix,
not a fake error passed directly into the helper alone. Prove unchanged exact
HTTP400 generic envelope, unchanged bind-error operation messages, unknown
error masking, and no collaborator calls on validation failure. For change,
preserve prior real validation-error cause proof and strengthen sentinel identity.

Direct service tests use malformed URL input that actually produces *url.Error
(e.g. malformed percent escape), then assert errors.Is ErrChangeInvalidInput,
errors.As *url.Error and underlying EscapeError where present. Test valid
uppercase scheme, trimmed valid URL, missing host, unsupported scheme, blank URL
and invalid ID, repository failure cause, no extra calls. Table tests central
helper direct/wrapped/nil behavior. Avoid mirrored AST-only tests or
artificial source branches solely for statement percentages.

VC-05 Verification/contract handoff: run focused unit tests, all four required
commands (`make -C backend check`, `coverage`, `deps-audit`, `api-test`) and the
complete retained `make -C backend legacy-api-test` to prove HTTP/SQL parity.
Use make-k check if baseline lint prevents other components from running; retain
known R5 lint debt honestly. No new integration fault scenario or outage harness
belongs in this repair. Existing actual malformed/invalid integration requests
should continue to assert unchanged400 bodies; add one malformed-URL apih request
only if the retained suite lacks real coverage of it. APIHydra cannot inspect
Go error causes, so direct unit evidence is the correct layer for cause identity.
Keep retries:-1, installed apih manual revision and backend suite location.

Record exact command exits, unit/API covered/total counts, package gaps and
named VC criterion-to-test mapping in the compact checkpoint. Update error
contract ledger and top plan status, labeling this branch as validation-cause
repair rather than claiming a duplicate-code cleanup. Preserve successful34
operation inventory and all useful SQL/legacy assertions. Append implementation
log per skill. Factory then native review --base origin/dev and supervisor
merge-to-dev; no promotion or autonomous next-category changes in subprocess.

## Implementation notes

A helper such as Validation(cause,semantic) returning the semantic/cause chain
is sufficient; choose the clearest existing naming convention after inspection.
Do not return fmt.Errorf or errors.Join from a module. API can call
HTTP(Validation(v.Errors,ErrModuleInvalidInput)); interpreter already knows
safe module400 messages, and translated Echo error must preserve full chain.
For syntax-valid but business-invalid URL, the service may return its existing
central sentinel directly. Binding failures already use InvalidPayload and
remain untouched. Preserve established behavior in existing
central helpers; do not broadly rewrite Database/ChangeCreate for this narrow fix.

Review full diff for accepted-input drift, altered short-circuit/evaluation order,
new dependencies, leaked validator/parser details in public response, semantic
identity loss, service transport imports and validation that reaches persistence.
R6 compact/API-if candidates are a later pass; keep this repair independently
reviewable and revertible.

Base confirmed at assignment: merged R1 origin/dev16bbf0b305cc263da1bf4f9b0fd0bda04d3fca4d. R2 is documented no-action; this numbered follow-up is an error-contract repair. Next implementation category after review/merge is R3, not another R2 branch.
