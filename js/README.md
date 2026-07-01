# @kitsyai/guten (JS runtime)

The **Node + browser** runtime of [guten](https://github.com/kitsyai/guten) — a
multi-format, multi-runtime templating engine (Liquid). It implements the same
[`spec/template-manifest.md`](../spec/template-manifest.md) contract as the Go
engine, and a **shared parity corpus proves the two render identically**.

## Install

```
npm i @kitsyai/guten     # or: pnpm add @kitsyai/guten
```

## Usage

```ts
import { Engine, newWithBuiltins } from "@kitsyai/guten";

const e = newWithBuiltins();
const r = e.render("basic_notification", {
  title: "Welcome", name: "Asha", body: "Your account is ready.", brand_name: "Acme",
});
// r.parts.subject / r.parts.html / r.parts.text

// register your own template (any Liquid):
e.register({
  name: "order_shipped",
  parts: { subject: "Your order {{ order_no }} shipped", text: "Hi {{ name | default: 'there' }}" },
});
```

## Browser / offline-first

Pure ESM, no runtime network calls; bundles cleanly (liquidjs runs in the
browser). Ship it in a web app for offline-first rendering.

## Parity (Go ≡ JS)

`pnpm test` runs the shared corpus at [`../spec/corpus/`](../spec/corpus) and
asserts the JS output equals the **Go golden** (`expected.json`). Regenerate the
golden from the Go reference with:

```
GUTEN_CORPUS_WRITE=1 go -C ../go test -run TestCorpusParity
```

## Renderers

- **Liquid** (default) — via liquidjs.
- **Layout** — `LayoutRenderer`: a JSON block layout (heading / paragraph /
  button / image) → HTML, the seam for a "Canva-like" designer. Byte-identical
  to the Go layout renderer (covered by the parity corpus).

## PDF

`renderToPdf(engine, name, data, converter)` renders the `html` part and hands
it to an injectable `PdfConverter`. HTML → PDF needs a browser engine
(puppeteer / headless Chromium), wired by the consumer — keeping guten pure and
offline-first.

## Dev

```
pnpm install          # if prompted, allow esbuild's build once: pnpm approve-builds
pnpm typecheck        # tsc --noEmit
pnpm test             # vitest (unit + parity corpus)
pnpm build            # tsup -> dist (ESM + CJS + d.ts)
```
