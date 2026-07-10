# @kitsy/guten (JS runtime)

The **Node + browser** runtime of [guten](https://github.com/kitsyai/guten) — a
multi-format, multi-runtime templating engine (Liquid). It implements the same
[`spec/template-manifest.md`](../spec/template-manifest.md) contract as the Go
engine, and a **shared parity corpus proves the two render identically**.

## Install

```
npm i @kitsy/guten     # or: pnpm add @kitsy/guten
```

## Usage

```ts
import { Engine, newWithBuiltins } from "@kitsy/guten";

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

### Script tag (UMD)

```html
<script src="https://cdn.jsdelivr.net/npm/@kitsy/guten@<version>/dist/index.umd.js"></script>
<script>
  const e = Guten.newWithBuiltins();
  const rendered = e.render("basic_notification", {
    title: "Welcome",
    name: "Asha",
    body: "Your account is ready.",
    brand_name: "Acme",
  });
  console.log(rendered.parts.html);
</script>
```

## React surface (`@kitsy/guten/react`)

For React users, import from the optional subpath and only include this package
when used:

```ts
import { GutenPDF, GutenView, useGuten } from "@kitsy/guten/react";
```

You must provide `react` and `react-dom` in your app:

```bash
npm i react react-dom
```

Use the helper for direct bootstrap:

```ts
import { GutenReact } from "@kitsy/guten/react";

GutenReact.render("#mount", "pdf", "invoice_bold", {
  data: { name: "Asha", invoice_number: "1001" },
});
```

UMD entry is also available:

```html
<script src="https://cdn.jsdelivr.net/npm/@kitsy/guten@<version>/dist/react.umd.js"></script>
<script>
  window.GutenReact.render("#mount", "html", "basic_notification", {
    data: { title: "Welcome", name: "Asha" },
  });
</script>
```

## Template assets on jsDelivr

Template registries are consumed from the dedicated template package so Node consumers can
list or cache them directly from CDN:

- `@kitsy/gutenkit` templates: `https://cdn.jsdelivr.net/npm/@kitsy/gutenkit@<version>/templates/index.json`
- fallback path (if npm package not yet published): `https://cdn.jsdelivr.net/gh/kitsyai/gutenkit@v<version>/templates/index.json`

The entries point to `.json` manifests and `.liquid` parts in the same tree:

```
.../templates/index.json
.../templates/<name>/template.json
.../templates/<name>/<part>.liquid
```

You can also use the helper functions:

```ts
import {
  templateRegistryFallbackUrl,
  templateAssetFallbackUrl,
  templateRegistryUrl,
  fetchTemplateRegistry,
  templateAssetUrl,
} from "@kitsy/guten";

const registry = await fetchTemplateRegistry("gutenkit", { version: "latest" });
for (const t of registry.templates) {
  console.log(t.name, templateAssetUrl("gutenkit", `${t.path}/template.json`));
}

templateRegistryFallbackUrl("gutenkit", { version: "0.2.4" });
templateAssetFallbackUrl("gutenkit", "invoice/template.json", { version: "0.2.4" });
```

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
pnpm build            # tsup -> dist (ESM + CJS + UMD + d.ts)
```
