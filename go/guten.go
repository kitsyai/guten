// Package guten is a multi-format templating engine. It compiles named
// templates written in Liquid and renders them with caller-supplied data into
// one or more output parts (for example: subject, html, text).
//
// Scope and non-scope
//
//	guten owns rendering only. It has no knowledge of channels, delivery,
//	recipients, PII, or any business domain. It does not know about email vs
//	SMS, about kitsy/heypkv/boss, or about what a template "means". A caller
//	(such as the gustav comms product) registers templates, passes data, and
//	gets rendered parts back; everything downstream — choosing a channel,
//	delivering, tracking status — is the caller's concern.
//
// Liquid
//
//	Templates are Liquid (https://shopify.github.io/liquid). Liquid is chosen
//	because it has mature, independent implementations in both Go
//	(github.com/osteele/liquid) and JavaScript (liquidjs), so the same template
//	renders the same way in guten's Go and Node runtimes, and because it is
//	designed to safely render user-authored templates.
//
// Security note
//
//	Liquid does not HTML-escape interpolated data by default. For HTML parts
//	fed with untrusted data, escape in the template with the `escape` filter,
//	e.g. {{ body | escape }}. Codes/URLs you generate yourself can be left
//	unescaped. A future guten option may enforce auto-escaping per part.
package guten

import (
	"fmt"
	"sort"
	"sync"

	"github.com/osteele/liquid"
)

// Conventional part names. They are conventions, not constraints: a template
// may define any part names. Email typically uses subject/html/text; SMS and
// WhatsApp use text; a PDF document renders from html (converted downstream).
const (
	PartSubject = "subject"
	PartHTML    = "html"
	PartText    = "text"
)

// Template is a named bundle of Liquid sources keyed by output part.
type Template struct {
	Name  string
	Parts map[string]string
}

// Rendered is the result of rendering a Template with a data set.
type Rendered struct {
	Template string
	Parts    map[string]string
}

// Engine compiles templates once at registration and renders them on demand.
// An Engine is safe for concurrent Render/RenderPart calls; Register takes a
// write lock, so registration and rendering may also run concurrently.
type Engine struct {
	lq    *liquid.Engine
	mu    sync.RWMutex
	tmpls map[string]map[string]*liquid.Template
}

// New returns an empty Engine with the standard Liquid filter/tag set.
func New() *Engine {
	return &Engine{
		lq:    liquid.NewEngine(),
		tmpls: make(map[string]map[string]*liquid.Template),
	}
}

// Register compiles and stores a template, replacing any existing template of
// the same name. It fails fast if the name is empty, no parts are supplied, or
// any part fails to parse — so a bad template is caught at registration time,
// not at send time.
func (e *Engine) Register(t Template) error {
	if t.Name == "" {
		return fmt.Errorf("guten: empty template name")
	}
	if len(t.Parts) == 0 {
		return fmt.Errorf("guten: template %q has no parts", t.Name)
	}
	compiled := make(map[string]*liquid.Template, len(t.Parts))
	for part, src := range t.Parts {
		tpl, err := e.lq.ParseString(src)
		if err != nil {
			return fmt.Errorf("guten: parse template %q part %q: %w", t.Name, part, err)
		}
		compiled[part] = tpl
	}
	e.mu.Lock()
	e.tmpls[t.Name] = compiled
	e.mu.Unlock()
	return nil
}

// Render renders every part of the named template with data.
func (e *Engine) Render(name string, data map[string]any) (Rendered, error) {
	e.mu.RLock()
	parts, ok := e.tmpls[name]
	e.mu.RUnlock()
	if !ok {
		return Rendered{}, fmt.Errorf("guten: template %q not registered", name)
	}
	bindings := toBindings(data)
	out := Rendered{Template: name, Parts: make(map[string]string, len(parts))}
	for part, tpl := range parts {
		b, err := tpl.Render(bindings)
		if err != nil {
			return Rendered{}, fmt.Errorf("guten: render template %q part %q: %w", name, part, err)
		}
		out.Parts[part] = string(b)
	}
	return out, nil
}

// RenderPart renders a single part of the named template. Useful when a channel
// needs only one part (e.g. SMS needs only "text").
func (e *Engine) RenderPart(name, part string, data map[string]any) (string, error) {
	e.mu.RLock()
	parts, ok := e.tmpls[name]
	e.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("guten: template %q not registered", name)
	}
	tpl, ok := parts[part]
	if !ok {
		return "", fmt.Errorf("guten: template %q has no part %q", name, part)
	}
	b, err := tpl.Render(toBindings(data))
	if err != nil {
		return "", fmt.Errorf("guten: render template %q part %q: %w", name, part, err)
	}
	return string(b), nil
}

// Templates lists the registered template names, sorted, for diagnostics.
func (e *Engine) Templates() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	names := make([]string, 0, len(e.tmpls))
	for n := range e.tmpls {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// toBindings copies data into the map[string]interface{} shape osteele/liquid
// expects. We copy rather than cast so a caller's map can't be mutated.
func toBindings(data map[string]any) map[string]interface{} {
	b := make(map[string]interface{}, len(data))
	for k, v := range data {
		b[k] = v
	}
	return b
}
