# guten — agent router

**Status: PARKED (2026-07-13).** Active feature development is paused while the
ecosystem focus is on `hey` distribution. Pause, not abandonment — resume from
the coop `workbench` track.

## What guten is

Cross-runtime (Go + JS) templating engine: render Liquid templates to
HTML/text/PDF. CLI + embedded web UI (`guten ui`) + npm package. Distributed via
`hey guten`. Full spec: [docs/feature-set-v1.md](docs/feature-set-v1.md).

## Shipped (v0.4.0)

- Engine (Go/JS parity), template library (builtin/gutenkit/user tiers), CLI
  (render/export/lib/ui/batch/new).
- `guten batch` (JSONL/CSV → many docs), user-library management (`guten new`,
  `lib add/rm`, UI duplicate-and-edit), `guten ui` (embedded Vite+React).

## Pending (coop `workbench` track — parked)

- `TASK-UI-V2-POLI-PART-THEM-1` — UI v2 (part switcher, theme panel, resizable
  panes).
- `TASK-DOCU-ENGI-DEPE-CONT-1` — document-engine dependency contract (how djin /
  heypkv apps vendor+pin bundles and embed the Go engine).

## To resume

`coop list tasks` here, `coop show <id>`. Rebuild UI with `cd cli/ui && npm run
build` (dist is committed). Release with `scripts/release.sh minor|patch`.
