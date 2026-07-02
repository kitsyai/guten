# guten CLI

A single Go binary for the [guten](https://github.com/kitsyai/guten) templating
engine — render templates to stdout, or export them to HTML / text / PDF. It
runs the canonical Go engine, so it's also handy for validating templates.

## Install

```
go install github.com/kitsyai/guten/cli/cmd/guten@latest
```
…or grab a release binary (win/mac/linux).

## Usage

```
guten render -t 'Hi {{ name }}' -d '{"name":"Ada"}'
guten render -t @welcome.liquid -d @data.json --part html
guten export -t @invoice.html -d @invoice.json -o invoice.html -o invoice.pdf
guten export --manifest @template.json -d @data.json -o out.html -o out.txt
guten builtins
guten version
```

Flags: `-t/--template` (source or `@file`), `--manifest` (full Template JSON),
`-r/--renderer` (`liquid` default | `layout`), `-d/--data` (JSON or `@file`),
`--part` (stdout part, default `html`), `-o/--out` (repeatable; the file
extension picks the part), `--chrome`.

**Theming / overrides** (no template edit needed):

```
guten export -t @invoice.html -d @data.json --theme @brand.json -o out.pdf
guten export -t @invoice.html -d @data.json --set theme.accent=#0ea5e9 -o out.pdf
guten export -t @invoice.html -d @data.json --css @brand.css -o out.pdf
```

- `--theme @file|JSON` — merged into the data under `theme` (template reads
  `theme.*` for fonts/colors/spacing).
- `--set key=value` — a single dotted-path override (repeatable).
- `--css @file|"…"` — extra CSS injected before `</head>` so it wins the cascade
  (repeatable). Great for `@font-face`, layout tweaks, or hiding blocks.

## Template library

The CLI resolves `--lib <name>` across a Maven/Gradle-style chain (highest
first): `--lib-dir <dir>` → `~/.kitsy/guten/user/templates/` →
`~/.kitsy/guten/gutenkit/templates/` (synced by `lib pull`) → an **embedded
snapshot** baked into the binary (works offline).

```
guten lib list                 # what's available (+ source)
guten lib show invoice         # manifest + sample data
guten lib pull                 # sync latest from github.com/kitsyai/gutenkit
guten export --lib invoice -d @data.json -o invoice.pdf
guten export --lib welcome -d @data.json --set theme.accent_color=#0ea5e9 -o out.html
```

A bundle's `theme.json` is the lowest theme layer, overridable by
`data.theme` → `--theme` → `--set`. Create your own under
`~/.kitsy/guten/user/templates/<name>/` and use `"extends": "<base>"` to inherit.

## PDF

`-o file.pdf` renders the `html` part and converts it with a **headless
Chrome / Edge / Chromium**, auto-detected (override with `--chrome` or the
`GUTEN_CHROME` env var). HTML and text output need no browser. Control page size
and margins from the template with CSS `@page { size: A4; margin: … }`.
