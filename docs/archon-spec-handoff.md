# Start Archon from an mch specification

Run this from the repository root after writing the accepted specification to
`/tmp/mch/spec/ddd-slug.md`, where `ddd` is the change ID:

```bash
archon workflow run archon-ship \
  --input spec_path=/tmp/mch/spec/123-slug.md \
  --branch change/123-slug --detach
```

Replace `123-slug` in both arguments with the change's ID and slug.

The workflow's first step copies the input to `$ARTIFACTS_DIR/spec.md`, then
deletes the source only if `cp` succeeds. A failed copy stops the workflow and
retains the source. Triage receives the artifact path. No spec is added to Git.

The Archon handoff is implemented; automatic export and launch from mch are
still to be implemented. Workflow details live in
[the Archon README](../.archon/README.md#temporary-specification-handoff).
