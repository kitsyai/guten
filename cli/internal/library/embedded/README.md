# Embedded snapshot

Canonical built-in templates now live at `templates/internal` and are generated
into this package as a bundled snapshot at build time.

Update process:

- `templates/internal` is the source of truth.
- `go test`/`go run` will use the generated snapshot first.
- `guten lib pull` fetches the latest templates into `~/.kitsy/guten/gutenkit`
  (which currently wins after local/user overrides).
