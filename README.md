# guten

A multi-format, multi-runtime **templating engine**. guten compiles named
templates written in [Liquid](https://shopify.github.io/liquid/) and renders
them with caller-supplied data into one or more output **parts** (subject, html,
text — and, on the roadmap, PDF).

guten owns **rendering only**. It knows nothing about channels, delivery,
recipients, PII, or any business domain. Callers (such as the **gustav** comms
product, or a billing service generating invoice PDFs) register templates, pass
data, and get rendered parts back.

## Pluggable engines

The templating engine is a pluggable **`Renderer`** (`go/renderer.go`). guten
ships a **Liquid** renderer today; future engines — a layout / "Canva-like"
html-css designer where users build beautiful invoices/emails with images and
colours, an MJML compiler, a WYSIWYG builder — implement the same `Renderer`
interface and register with `Engine.RegisterRenderer`. Nothing about the
template model, the `Engine` API, or callers changes when a new engine is added.
A `Template` names its renderer (`Template.Renderer`); empty means the engine's
default (Liquid).

### Why Liquid is the first engine

Liquid has mature, independent implementations in Go
(`github.com/osteele/liquid`) and JavaScript (`liquidjs`), so the same template
renders the same way in every guten runtime, and it safely renders
user-authored templates — which matters because consumers register their own
templates, not just the batteries-included ones.

## Runtimes

| Runtime | Status | Path |
| --- | --- | --- |
| Go | **v0.1 — working** | [`go/`](go) (module `github.com/kitsyai/guten/go`) |
| Node + browser | **v0.1 — working** | [`js/`](js) (npm `@kitsyai/guten`) |

A shared [`spec/`](spec) defines the template + data manifest, and a **parity
corpus** ([`spec/corpus/`](spec/corpus)) guarantees Go and JS produce identical
output for the same template + data — the Go reference generates the golden, and
both runtimes assert against it.

## Output formats

- **v0:** `html`, `text` (and any caller-defined part — `subject`, `sms`, …).
- **Roadmap:** `pdf` (via an HTML → PDF step over the rendered `html` part).

## Layout

| Path | Purpose |
| --- | --- |
| `go/` | Go module: the engine + batteries-included templates. |
| `js/` | Node/browser runtime (planned). |
| `templates/` | Batteries-included template sources. |
| `spec/` | Template + `template_data` manifest schema. |
| `docs/` | Usage and design notes. |

## Go usage

```go
import guten "github.com/kitsyai/guten/go"

e, _ := guten.NewWithBuiltins()
out, err := e.Render("basic_notification", map[string]any{
    "brand_name": "Acme",
    "title":      "Welcome",
    "name":       "Asha",
    "body":       "Your account is ready.",
    "action_url": "https://example.test/start",
})
// out.Parts["subject"], out.Parts["html"], out.Parts["text"]
```

Register your own template (replaces a builtin of the same name):

```go
_ = e.Register(guten.Template{
    Name: "order_shipped",
    Parts: map[string]string{
        "subject": "Your order {{ order_no }} has shipped",
        "text":    "Hi {{ name | default: \"there\" }}, it's on the way.",
    },
})
```

## Configuration (cnos)

guten reads configuration **only through cnos** — never from process
environment variables. The cnos runtime (cnos-go) resolves the `guten.*` value
namespace with its own layering/superposition; code `Defaults()` apply when a
value is absent. A consuming service overrides any of these in its own cnos
workspace.

| cnos value | meaning |
| --- | --- |
| `guten.default_renderer` | renderer for templates that don't name one (default `liquid`) |
| `guten.templates` | templates supplied as config ("templates-as-config"), a JSON string for now |

```go
e, _ := guten.NewFromCnos()          // cnos.Load() + builtins + guten.templates
// or, if your service already holds a *cnos.Runtime:
e, _ := guten.NewFromRuntime(rt)
// or guten.NewWithBuiltins() for zero-config, batteries-included.
```

`guten.templates` carries whole templates as a string for now; file-shaped
template configs land when cnos supports larger text configs.

## Security

Liquid does not HTML-escape interpolated data by default. In `html` parts that
carry **untrusted data**, escape with the `escape` filter — e.g.
`{{ body | escape }}`. Values you generate yourself (codes, signed URLs) can be
left unescaped. See the builtins in [`go/builtins.go`](go/builtins.go) for the
pattern.
