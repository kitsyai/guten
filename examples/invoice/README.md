# Invoice example

A worked example: a GST **tax invoice** rendered by guten (a Liquid HTML
template) to PDF via the CLI, reproducing the layout kitsy/domains draws
programmatically with gofpdf — header, order summary, Supplier/Bill-to/Ship-to
blocks, the line-items table, totals, the dashed payment chit, and the Terms
footer.

## Render

```
guten export -t @invoice.html.liquid -d @sample.json -o invoice.html -o invoice.pdf
```

PDF output needs a headless Chrome/Edge/Chromium (see [`../../cli`](../../cli));
HTML output needs no browser. Page size/margins come from the template's
`@page` CSS (A4).

## Files

| File | What |
|---|---|
| `invoice.html.liquid` | the template (single `html` part, Liquid renderer) |
| `sample.json` | sample data — the shape the template expects |

`invoice.html` / `invoice.pdf` are generated outputs (git-ignored).

## Theming & overrides

Every visual choice is a `theme.*` variable with a default, so you rebrand
without editing the template:

```
guten export -t @invoice.html.liquid -d @sample.json --theme @brand.json -o out.pdf
guten export -t @invoice.html.liquid -d @sample.json --set theme.accent_color=#0ea5e9 -o out.pdf
guten export -t @invoice.html.liquid -d @sample.json --css "body{font-family:Georgia,serif}" -o out.pdf
```

Theme keys: `font_family`, `font_size`, `text_color`, `muted_color`,
`rule_color`, `accent_color`, `chit_border_color`, `page_size`, `page_margin`.
`--css` (injected before `</head>`) wins the cascade — use it for `@font-face`,
layout tweaks, or hiding blocks.

For exact-font parity (e.g. Noto Sans), point an `@font-face` at a local `.woff2`
via `--css`, or set `theme.font_family` to an installed family.
