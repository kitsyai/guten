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
