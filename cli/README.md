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

## PDF

`-o file.pdf` renders the `html` part and converts it with a **headless
Chrome / Edge / Chromium**, auto-detected (override with `--chrome` or the
`GUTEN_CHROME` env var). HTML and text output need no browser. Control page size
and margins from the template with CSS `@page { size: A4; margin: … }`.
