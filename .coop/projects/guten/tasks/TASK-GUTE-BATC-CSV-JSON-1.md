---
id: TASK-GUTE-BATC-CSV-JSON-1
short_id: fba4753b9189
title: "guten batch: CSV/JSONL to many documents"
type: feature
status: done
created: 2026-07-12
updated: 2026-07-12
aliases: []
priority: p2
track: workbench
acceptance:
  - guten batch --lib invoice -d @rows.jsonl --name '{{ invoice.number }}.pdf'
    -o out/ renders one file per row (html + pdf)
  - row failures reported with row number, run continues; exit non-zero if any
    failed
  - "UI: paste/upload rows, preview a row, download zip"
tests_required: []
origin:
  authority_refs:
    - docs/feature-set-v1.md
  derived_refs: []
---
