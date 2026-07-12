---
id: TASK-USER-LIBR-MANA-LIB-1
short_id: 2326f2c4071b
title: "User library management: lib add/rm + guten new"
type: feature
status: in_review
created: 2026-07-12
updated: 2026-07-12
aliases: []
priority: p2
track: workbench
acceptance:
  - guten new <name> --from invoice scaffolds
    template.json/html.liquid/sample.json into ~/.kitsy/guten/user
  - guten lib add <dir>, guten lib rm <name> manage the user tier; builtins
    untouchable
  - "UI duplicate-and-edit flow: edit liquid + sample, save to user lib, render
    immediately"
tests_required: []
origin:
  authority_refs:
    - docs/feature-set-v1.md
  derived_refs: []
---
