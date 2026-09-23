# Spec File Structure

Use this structure exactly for every Spec file. Replace all placeholders and instructional text with concrete Spec content. Keep every section concise, specific, and testable.

A Spec expresses the desired future state and gives strict, implementation-ready instructions for
what must change and how the Change must be conducted. The Definition and initial code originate the
Change: the Definition supplies direction, while code supplies current behavior and constraints. Code
remains the single source of truth for current behavior throughout the Change lifecycle.

Include every relevant branch change already applied manually, by an agent, or by another process.
The PR summarizes what the final code changed, why it changed, verification evidence, and any Spec
instructions intentionally deferred; it does not override code. Do not add implementation-progress
statuses or a change diary; Git and the PR preserve that history.

Do not add, remove, rename, or reorder top-level `##` sections unless the user explicitly changes the Spec workflow. Optional `###` and `####` headings may organize details within a required section.

Use nested headings when a flat list would obscure screens, flows, states, commands, keybindings, configuration, persistence, error handling, glossary terms, naming conventions, relationships, navigation, or data movement.

Use a Markdown-native diagram when a non-trivial workflow, screen flow, state transition, relationship, or data flow would be clearer visually. Prefer Mermaid fenced blocks such as `flowchart`, `sequenceDiagram`, or `stateDiagram-v2` when supported; otherwise use a concise Markdown table or ASCII diagram. A diagram must clarify, not replace, testable prose. If a non-trivial flow has no diagram, explain why in `Design Notes`.

The generated Spec must begin with one H1 title. An optional `Types:` metadata line may appear
immediately after the title. When it is present, place exactly one blank line between the title
and metadata and exactly one blank line between the metadata and Spec body. When it is omitted,
place exactly one blank line between the title and Spec body. Do not place blank lines or other
content before the H1 title.

Format populated metadata as `Types: <type-slugs>`. Join multiple slugs with `|` and no spaces,
as in `Types: feature|fix|test`. `Types:` with an empty value is also structurally valid. Structural
validation checks only the line's placement and format; it does not restrict which slugs may
appear.

Use these subsystem tags where instructed:

- `DOC` - documentation
- `DB` - database
- `BE` - backend
- `FE` - frontend
- `CLI` - CLI
- `SP` - scripts and prompts
- `OTH` - work that does not fit another tag

Format a tagged item as `- <TAG> - <statement>`. When one item spans multiple subsystems, join its tags with `|` and no spaces, as in `- BE|CLI - <statement>`.

Do not wrap the generated Spec in a code block.

# <Spec Title>

Types: feature|fix|test

## Goal

Describe the observable end state or end states this Spec must define. Use multiple goal statements when the Spec has distinct required outcomes. Do not write a list of implementation tasks.

## Scope

- List the behavior, architecture, implementation, and explicitly requested documentation areas
  included in the Change.
- Do not add documentation work as a routine Change stage. Include it only when the user or Definition
  explicitly requests a README, ADR, or operational runbook update.
- Keep every item directly tied to the Goal.
- Account for relevant existing branch work and exclude adjacent work outside the requested Change.

## Requirements

- State testable obligations the implementation must satisfy. Every delivered obligation must be
  supported by code and tests or identified as intentionally deferred in the PR summary.
- Include expected behavior, important boundaries, validation, and failure handling where relevant.
- Prefix each subsystem-specific requirement with all applicable subsystem tags. Leave an item untagged only when it is genuinely cross-cutting.
- Write behavioral requirements, not task summaries. For example:
  - `- CLI - The Change details view displays the Ref UUID label and value when a value is present.`
  - `- SP - The session restore script resumes the selected session using the configured default and temporary directories.`

## Non-Goals

- List related work that is intentionally outside the Spec.
- Include decisions that prevent accidental scope expansion.
- Move useful but non-essential work here or to `Follow-Ups` instead of expanding Scope.
- Use `- None.` when there are no non-goals.

## Design Notes

- Record implementation decisions already evidenced by code, plus constraints, data model
  assumptions, UX details, and workflow rules needed to guide the Change.
- Use nested headings for a glossary, naming conventions, state or flow models, screen design, data shapes, compatibility, or other structured detail when useful.
- Link to a durable decision or operational runbook only when the Change explicitly depends on it;
  code remains authoritative for current behavior.
- State assumptions that reviewers and future agents must preserve.

## Verification

- List the repository-supported commands needed to verify every affected area. Make result claims
  only from available verification evidence.
- Include relevant lint, unit test, race test, API-test, typecheck, and build commands.
- Do not invent commands or include commands unrelated to the Spec.

## QA Test Cases

- List manual or product-level scenarios, not implementation tasks or automated Verification commands.
- Cover applicable happy paths, validation failures, backend or external-command failures, cancellation or no-op behavior, persistence, and boundary cases.
- Prefix every QA test case with one or more subsystem tags. Use `OTH` when no more specific tag applies. For example:
  - `- CLI - Open a Change that has a Ref UUID and verify the details view shows the expected label and value.`
  - `- SP - Restore a saved session and verify the script uses the configured directories and resumes the selected session.`

## Review Focus

- Identify the riskiest or most subtle areas reviewers should inspect first.
- Highlight changed contracts, data flow, persistence, migrations, concurrency, security, generated artifacts, or workflow automation when relevant.

## Follow-Ups

- List useful future work that is outside the Spec.
- Use `- None.` when there are no known follow-ups.
