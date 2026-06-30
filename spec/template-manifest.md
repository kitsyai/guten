# guten template & data manifest (v0)

This is the runtime-neutral contract every guten runtime (Go now, JS next)
implements. The Go types in `go/guten.go` are the reference; `js/` must match.

## Template

A **template** is a named bundle of **parts**. Each part is a Liquid source
string that renders to one output (subject line, HTML body, plain text, …).

```
template:
  name:  string            # unique id, referenced at render time
  parts: map<string,string> # part name -> Liquid source
```

Part names are conventions, not constraints:

| Part | Used by |
| --- | --- |
| `subject` | email subject line |
| `html` | email HTML body, or the source for PDF rendering |
| `text` | plain-text email body, SMS, WhatsApp, feeds |

A template may define any additional parts a caller needs.

## Data

Render input is a flat-ish map of named fields the template interpolates:

```
template_data[<template_name>].fields = { ... }
```

Example for `basic_notification`:

```json
{
  "subject": "Welcome",          // optional; falls back to title
  "title": "Welcome",
  "name": "Asha",                // optional; falls back to "there"
  "body": "Your account is ready.",
  "brand_name": "Acme",          // optional
  "accent": "#10b981",           // optional hex; default #10b981
  "action_url": "https://...",   // optional; renders a button when present
  "action_label": "Open"         // optional; default "Open"
}
```

## Render result

```
rendered:
  template: string
  parts:    map<string,string>   # part name -> rendered output
```

## Rules

1. **Compile at registration.** A template that fails to parse is rejected at
   registration, never at send time.
2. **Same template + same data => identical output across runtimes.** This is
   enforced by the parity corpus (planned) under `spec/corpus/`.
3. **No business knowledge in guten.** Brand, copy, and channel choices are all
   data or caller decisions. Batteries-included templates are brand-neutral and
   fully parameterised.
4. **Escaping is explicit.** `html` parts must `| escape` untrusted fields.
