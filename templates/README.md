# templates/

`templates/internal` is the shared built-in template source of truth for the guten
runtimes:

- CLI offline snapshot
- Go builtin templates (`NewWithBuiltins`)
- JS built-in template bootstrap and template resolution

Each entry is a manifest directory with `template.json` plus optional
`theme.json` / `sample.json` / part source files.

The templates are **brand-neutral** and fully parameterised by data â€” guten
does not carry business, brand, or channel knowledge.
