# guten feature set v1 — from renderer to document workbench

guten v0.3 ships the engine (Go/JS parity), the template library, the CLI,
and `guten ui` (embedded web UI: pick template → edit data → preview →
PDF/HTML). v1 turns the UI into a small document workbench and makes guten
the document engine other ecosystem apps (djin's invoice register, heypkv
business apps) build on. Ecosystem context: hey repo `docs/north-star.md`.

## 1. Batch rendering (`guten batch` + UI)

Render one template against many records: CSV/JSONL in → n PDFs (or HTMLs)
out, with a filename template (`--name "{{ invoice.number }}.pdf"`).
- CLI: `guten batch --lib invoice -d @rows.jsonl --name "..." -o out/`
- UI: upload/paste rows, preview any row, download all as zip.
- This is the "generate the quarter's invoices in one shot" story — the
  exact workflow we did by hand for PC27I.

## 2. User template library management

Today `~/.kitsy/guten/user` works but is managed by hand.
- CLI: `guten lib add <dir|@manifest>`, `guten lib rm <name>`,
  `guten new <name> [--from <builtin>]` (scaffold template.json +
  html.liquid + sample.json from an existing bundle).
- UI: "duplicate & edit" a builtin — edit the liquid parts and sample,
  save to the user library, immediately renderable. Guardrail: builtins are
  read-only; user lib is where edits land.

## 3. UI v2 polish

- Part switcher (html/text/subject) for email templates.
- Theme panel: edit `theme.*` keys (accent, fonts, page size) with live
  re-render; persists into the data JSON.
- Layout: resizable panes, render-on-idle (debounced) toggle.

## 4. Library-as-a-dependency contract

djin and heypkv apps embed the Go engine + selected bundles. Document (in
spec/) the supported way to: vendor specific bundles, pin their versions,
and validate data against a bundle's sample shape. Keep the corpus parity
guarantee (Go ↔ JS identical output) as the compatibility contract.

## Non-goals for v1

WYSIWYG template design, cloud rendering, template marketplace — later or
never; the engine stays lean and deterministic.
