# Changelog Entries

This repository uses `go.opentelemetry.io/build-tools/chloggen` to collect
release notes before cutting receiver module tags.

Create an entry for user-visible changes:

```bash
make chlog-new NAME=<short-entry-name>
```

Then edit the generated `.chloggen/<short-entry-name>.yaml` file.

Use one of these change types:

- `breaking`
- `deprecation`
- `new_component`
- `enhancement`
- `bug_fix`

Run validation before opening or merging a pull request:

```bash
make chlog-validate
```

Preview the generated changelog without modifying files:

```bash
make chlog-preview
```

Small internal-only maintenance changes may omit a changelog entry.
