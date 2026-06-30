// Package guten is a multi-format templating engine. It compiles named
// templates and renders them with caller-supplied data into one or more output
// parts (for example: subject, html, text).
//
// Scope and non-scope
//
//	guten owns rendering only. It has no knowledge of channels, delivery,
//	recipients, PII, or any business domain. It does not know about email vs
//	SMS, about kitsy/heypkv/boss, or about what a template "means". A caller
//	(such as the gustav comms product, or a billing service generating invoice
//	PDFs) registers templates, passes data, and gets rendered parts back.
//
// Pluggable engines
//
//	The templating engine is a pluggable Renderer (see renderer.go). guten ships
//	a Liquid renderer today; future renderers — a layout / "Canva-like" html-css
//	designer, MJML, a WYSIWYG invoice template — implement the same Renderer
//	interface and are added with Engine.RegisterRenderer, with no change to
//	callers or to the template model. A Template names its renderer
//	(Template.Renderer); empty means the Engine's default (Liquid).
//
// Configuration
//
//	Configuration flows through cnos only — guten never reads process
//	environment variables. The cnos runtime (cnos-go) resolves the `guten.*`
//	value namespace with its own layering/superposition; code Defaults() apply
//	when a value is absent. See config.go.
//
// Security note
//
//	The Liquid renderer does not HTML-escape interpolated data by default. For
//	html parts fed with untrusted data, escape in the template with the `escape`
//	filter, e.g. {{ body | escape }}. Values you generate yourself (codes,
//	signed URLs) can be left unescaped.
package guten

import (
	"fmt"
	"sort"
	"sync"
)

// Conventional part names. They are conventions, not constraints: a template
// may define any part names. Email typically uses subject/html/text; SMS and
// WhatsApp use text; a PDF document renders from html (converted downstream).
const (
	PartSubject = "subject"
	PartHTML    = "html"
	PartText    = "text"
)

// Template is a named bundle of template sources keyed by output part. Renderer
// selects the templating engine (empty => the Engine's default). The json tags
// support templates-as-config (see config.go).
type Template struct {
	Name     string            `json:"name"`
	Renderer string            `json:"renderer,omitempty"`
	Parts    map[string]string `json:"parts"`
}

// Rendered is the result of rendering a Template with a data set.
type Rendered struct {
	Template string
	Parts    map[string]string
}

// storedTemplate is a compiled template plus the renderer that produced it.
type storedTemplate struct {
	renderer string
	parts    map[string]CompiledTemplate
}

// Engine compiles templates once at registration and renders them on demand.
// It owns a set of pluggable renderers and a set of compiled templates. An
// Engine is safe for concurrent use.
type Engine struct {
	mu              sync.RWMutex
	renderers       map[string]Renderer
	defaultRenderer string
	tmpls           map[string]storedTemplate
}

// New returns an Engine with the built-in Liquid renderer registered.
func New() *Engine {
	e := &Engine{
		renderers:       make(map[string]Renderer),
		defaultRenderer: DefaultRenderer,
		tmpls:           make(map[string]storedTemplate),
	}
	e.RegisterRenderer(NewLiquidRenderer())
	return e
}

// RegisterRenderer adds (or replaces) a pluggable templating engine. This is
// the extension point for future engines (layout/html-css, MJML, designer).
func (e *Engine) RegisterRenderer(r Renderer) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.renderers[r.Name()] = r
}

// SetDefaultRenderer chooses the renderer used by templates that don't name
// one. The renderer must already be registered.
func (e *Engine) SetDefaultRenderer(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.renderers[name]; !ok {
		return fmt.Errorf("guten: renderer %q not registered", name)
	}
	e.defaultRenderer = name
	return nil
}

// Register compiles and stores a template, replacing any existing template of
// the same name. It fails fast: empty name, no parts, an unknown renderer, or
// a part that fails to parse are all caught here, not at send time.
func (e *Engine) Register(t Template) error {
	if t.Name == "" {
		return fmt.Errorf("guten: empty template name")
	}
	if len(t.Parts) == 0 {
		return fmt.Errorf("guten: template %q has no parts", t.Name)
	}
	e.mu.RLock()
	rendererName := t.Renderer
	if rendererName == "" {
		rendererName = e.defaultRenderer
	}
	r, ok := e.renderers[rendererName]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("guten: template %q uses unknown renderer %q", t.Name, rendererName)
	}
	parts := make(map[string]CompiledTemplate, len(t.Parts))
	for part, src := range t.Parts {
		c, err := r.Compile(src)
		if err != nil {
			return fmt.Errorf("guten: parse template %q part %q (%s): %w", t.Name, part, rendererName, err)
		}
		parts[part] = c
	}
	e.mu.Lock()
	e.tmpls[t.Name] = storedTemplate{renderer: rendererName, parts: parts}
	e.mu.Unlock()
	return nil
}

// Render renders every part of the named template with data.
func (e *Engine) Render(name string, data map[string]any) (Rendered, error) {
	e.mu.RLock()
	st, ok := e.tmpls[name]
	e.mu.RUnlock()
	if !ok {
		return Rendered{}, fmt.Errorf("guten: template %q not registered", name)
	}
	out := Rendered{Template: name, Parts: make(map[string]string, len(st.parts))}
	for part, c := range st.parts {
		s, err := c.Render(data)
		if err != nil {
			return Rendered{}, fmt.Errorf("guten: render template %q part %q: %w", name, part, err)
		}
		out.Parts[part] = s
	}
	return out, nil
}

// RenderPart renders a single part of the named template. Useful when a channel
// needs only one part (e.g. SMS needs only "text").
func (e *Engine) RenderPart(name, part string, data map[string]any) (string, error) {
	e.mu.RLock()
	st, ok := e.tmpls[name]
	e.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("guten: template %q not registered", name)
	}
	c, ok := st.parts[part]
	if !ok {
		return "", fmt.Errorf("guten: template %q has no part %q", name, part)
	}
	s, err := c.Render(data)
	if err != nil {
		return "", fmt.Errorf("guten: render template %q part %q: %w", name, part, err)
	}
	return s, nil
}

// Templates lists the registered template names, sorted, for diagnostics.
func (e *Engine) Templates() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return sortedKeys(e.tmpls)
}

// Renderers lists the registered renderer names, sorted.
func (e *Engine) Renderers() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return sortedKeys(e.renderers)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// toBindings copies data into the map[string]interface{} shape renderers
// expect. We copy rather than cast so a caller's map can't be mutated.
func toBindings(data map[string]any) map[string]interface{} {
	b := make(map[string]interface{}, len(data))
	for k, v := range data {
		b[k] = v
	}
	return b
}
