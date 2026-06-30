package guten

import "github.com/osteele/liquid"

// Renderer is a pluggable templating engine. guten ships the Liquid renderer
// below; future engines — a layout / "Canva-like" html-css designer where users
// build beautiful invoices/emails with images and colours, an MJML compiler, a
// WYSIWYG template builder — implement this same interface and are registered
// with Engine.RegisterRenderer. Nothing about the template model, the Engine
// API, or callers changes when a new engine is added.
type Renderer interface {
	// Name is the renderer's stable id, referenced by Template.Renderer.
	Name() string
	// Compile parses a template source into a reusable CompiledTemplate, or
	// returns a parse error (which Engine surfaces at registration time).
	Compile(source string) (CompiledTemplate, error)
}

// CompiledTemplate is a parsed template ready to render with data.
type CompiledTemplate interface {
	Render(data map[string]any) (string, error)
}

// Built-in renderer ids.
const (
	// RendererLiquid is the built-in Liquid renderer.
	RendererLiquid = "liquid"
	// DefaultRenderer is used for templates that don't name one.
	DefaultRenderer = RendererLiquid
)

// liquidRenderer is the built-in Renderer backed by github.com/osteele/liquid,
// chosen for cross-runtime parity (a JS liquidjs runtime renders identically)
// and safe handling of user-authored templates.
type liquidRenderer struct{ lq *liquid.Engine }

// NewLiquidRenderer returns the built-in Liquid renderer.
func NewLiquidRenderer() Renderer { return &liquidRenderer{lq: liquid.NewEngine()} }

func (r *liquidRenderer) Name() string { return RendererLiquid }

func (r *liquidRenderer) Compile(source string) (CompiledTemplate, error) {
	t, err := r.lq.ParseString(source)
	if err != nil {
		return nil, err
	}
	return liquidCompiled{t: t}, nil
}

type liquidCompiled struct{ t *liquid.Template }

func (c liquidCompiled) Render(data map[string]any) (string, error) {
	b, err := c.t.Render(toBindings(data))
	if err != nil {
		return "", err
	}
	return string(b), nil
}
