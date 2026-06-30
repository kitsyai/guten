# guten

A multi-format, multi-runtime **templating engine**. guten compiles named
templates written in [Liquid](https://shopify.github.io/liquid/) and renders
them with caller-supplied data into one or more output **parts** (subject, html,
text — and, on the roadmap, PDF).

guten owns **rendering only**. It knows nothing about channels, delivery,
recipients, PII, or any business domain. Callers (such as the **gustav** comms
product, or a billing service generating invoice PDFs) register templates, pass
data, and get rendered parts back.

## Why Liquid

Liquid has mature, independent implementations in both Go
(`github.com/osteele/liquid`) and JavaScript (`liquidjs`), so the same template
renders the same way in every guten runtime. It is also designed to safely
render user-authored templates — which matters because consumers can register
their own templates, not just the batteries-included ones.

## Runtimes

| Runtime | Status | Path |
| --- | --- | --- |
| Go | **v0 — working** | [`go/`](go) (module `github.com/kitsyai/guten/go`) |
| Node + browser | planned | `js/` |

A shared [`spec/`](spec) defines the template + data manifest, and a parity test
corpus (planned) guarantees Go and JS produce identical output for the same
template + data.

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

## Security

Liquid does not HTML-escape interpolated data by default. In `html` parts that
carry **untrusted data**, escape with the `escape` filter — e.g.
`{{ body | escape }}`. Values you generate yourself (codes, signed URLs) can be
left unescaped. See the builtins in [`go/builtins.go`](go/builtins.go) for the
pattern.
