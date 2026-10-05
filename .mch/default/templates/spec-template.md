# [Feature name]

Format the specification so that every line is at most 100 characters long.
Remove this formatting instruction from the completed specification.

## Goal

[What should this change accomplish?]

## Scope

- In scope: [Behavior to add or change]
- Out of scope: [What this change does not cover]

## Requirements

Keep requirements concise. Use a single flat list of bullets describing what to
build and the rules it follows. Put application scenarios under **Testcases**.
Each requirement must be testable and covered by meaningful unit tests.

- [Functionality to add or change, including applicable rules and limits.]

Put constraints that cannot be tested under **Notes → Additional requirements**.

## Testcases

Focus testcases on scenarios that provide useful verification. Testcases are
human-reproducible application scenarios that confirm particular functionality
works. Choose relevant happy paths, edge cases, and error paths.
Write each testcase as a single-line bullet: action → expected observable result.

- [Action with concrete input] → [Expected observable result].

## Notes

Always include the `## Notes` heading, even when the section is empty. Its `###`
subsections are optional: include each only when needed and omit unused subsections.
If none are needed, leave only the `## Notes` heading. Remove these template
instructions from the completed specification.

### Reference contract

- Existing contract items used: [types, methods, or functions]

### Additional requirements

[List constraints that cannot be tested, such as prescribed type names or naming
conventions.]

### Diagrams

[Add diagrams that clarify behavior, relationships, or data flow.]

### Code examples

[Add examples that clarify usage and follow the reference contract.]

### Terminology

[Define terms or abbreviations needed to understand this specification.]
