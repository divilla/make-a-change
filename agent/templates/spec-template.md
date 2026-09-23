# Spec: [Feature name]

## Goal

[What should this change accomplish?]

## Scope

- In scope: [Behavior to add or change]
- Out of scope: [What this change does not cover]

## Reference contract

- Relevant `skeleton/` files: [paths]
- Existing contract items used: [types, methods, or functions]

Follow `skeleton/`. If a contract change is needed, obtain the user's agreement
and have the skeleton updated before specifying the new contract here.

## Requirements

Use a single flat list of bullets that combines requirements, acceptance criteria,
and QA test cases. Each requirement must describe a testable outcome, preferably
one that can also be verified manually. Attach one or more unit tests to each
requirement. Where practical, also attach integration or end-to-end tests; these
do not replace unit tests.

Write each bullet in clear, natural language. Use Given/When/Then, identifiers,
or checkboxes only when useful. Test references may appear alongside a requirement
or in a separate mapping, provided it is clear which tests cover each requirement.

- [Required behavior and observable result.]
- [Error or edge case behavior and observable result.]

Put constraints that cannot be tested under **Notes → Additional requirements**.

## Notes

Always include the `## Notes` heading, even when the section is empty. Its `###`
subsections are optional: include each only when needed and omit unused subsections.
If none are needed, leave only the `## Notes` heading. Remove these template
instructions from the completed specification.

### Additional requirements

[List constraints that cannot be tested, such as prescribed type names or naming
conventions. Keep contract names aligned with `skeleton/`.]

### Diagrams

[Add diagrams that clarify behavior, relationships, or data flow.]

### Code examples

[Add examples that clarify usage and follow the reference contract.]

### Terminology

[Define terms or abbreviations needed to understand this specification.]
