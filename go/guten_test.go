package guten

import (
	"strings"
	"testing"
)

func TestRenderBuiltinNotification(t *testing.T) {
	e, err := NewWithBuiltins()
	if err != nil {
		t.Fatalf("NewWithBuiltins: %v", err)
	}
	r, err := e.Render("basic_notification", map[string]any{
		"brand_name":   "Acme",
		"title":        "Welcome",
		"name":         "Asha",
		"body":         "Your account is ready.",
		"action_url":   "https://example.test/start",
		"action_label": "Get started",
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if got := r.Parts[PartSubject]; got != "Welcome" {
		t.Fatalf("subject = %q, want Welcome", got)
	}
	if got := r.Parts[PartText]; !strings.Contains(got, "Hi Asha,") || !strings.Contains(got, "Your account is ready.") {
		t.Fatalf("text part missing content:\n%s", got)
	}
	for _, want := range []string{"Acme", "Welcome", "https://example.test/start", "Get started"} {
		if !strings.Contains(r.Parts[PartHTML], want) {
			t.Fatalf("html part missing %q:\n%s", want, r.Parts[PartHTML])
		}
	}
}

func TestSubjectFallsBackToTitle(t *testing.T) {
	e, _ := NewWithBuiltins()
	r, err := e.Render("basic_notification", map[string]any{
		"title": "Order shipped",
		"body":  "It's on the way.",
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if r.Parts[PartSubject] != "Order shipped" {
		t.Fatalf("subject = %q, want Order shipped", r.Parts[PartSubject])
	}
}

func TestLiquidDefaultsAndConditionals(t *testing.T) {
	e := New()
	if err := e.Register(Template{
		Name:  "greet",
		Parts: map[string]string{PartText: `{{ name | default: "there" }}{% if vip %} (VIP){% endif %}`},
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	vip, err := e.RenderPart("greet", PartText, map[string]any{"vip": true})
	if err != nil {
		t.Fatalf("RenderPart: %v", err)
	}
	if vip != "there (VIP)" {
		t.Fatalf("vip render = %q, want %q", vip, "there (VIP)")
	}

	named, err := e.RenderPart("greet", PartText, map[string]any{"name": "Sam"})
	if err != nil {
		t.Fatalf("RenderPart: %v", err)
	}
	if named != "Sam" {
		t.Fatalf("named render = %q, want Sam", named)
	}
}

func TestHTMLEscapesUntrustedData(t *testing.T) {
	e := New()
	if err := e.Register(Template{
		Name:  "card",
		Parts: map[string]string{PartHTML: `<p>{{ body | escape }}</p>`},
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	out, err := e.RenderPart("card", PartHTML, map[string]any{"body": `<script>alert(1)</script>`})
	if err != nil {
		t.Fatalf("RenderPart: %v", err)
	}
	if strings.Contains(out, "<script>") {
		t.Fatalf("expected escaped output, got: %s", out)
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Fatalf("expected HTML entities, got: %s", out)
	}
}

func TestRegisterRejectsBadTemplate(t *testing.T) {
	e := New()
	if err := e.Register(Template{Name: "bad", Parts: map[string]string{PartText: "{% if %}"}}); err == nil {
		t.Fatal("expected parse error for malformed Liquid, got nil")
	}
	if err := e.Register(Template{Name: "", Parts: map[string]string{PartText: "x"}}); err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if err := e.Register(Template{Name: "noparts"}); err == nil {
		t.Fatal("expected error for no parts, got nil")
	}
	if err := e.Register(Template{Name: "badengine", Renderer: "nope", Parts: map[string]string{PartText: "x"}}); err == nil {
		t.Fatal("expected error for unknown renderer, got nil")
	}
}

func TestRenderUnknownTemplate(t *testing.T) {
	e := New()
	if _, err := e.Render("nope", nil); err == nil {
		t.Fatal("expected error rendering unregistered template")
	}
}

// echoRenderer is a trivial pluggable renderer that returns the source verbatim,
// ignoring data. It proves a non-Liquid engine plugs in without touching Engine
// or the template model.
type echoRenderer struct{}

func (echoRenderer) Name() string { return "echo" }
func (echoRenderer) Compile(source string) (CompiledTemplate, error) {
	return echoCompiled(source), nil
}

type echoCompiled string

func (c echoCompiled) Render(map[string]any) (string, error) { return string(c), nil }

func TestPluggableRenderer(t *testing.T) {
	e := New()
	e.RegisterRenderer(echoRenderer{})

	if got := e.Renderers(); len(got) != 2 || got[0] != "echo" || got[1] != RendererLiquid {
		t.Fatalf("renderers = %v, want [echo liquid]", got)
	}

	if err := e.Register(Template{
		Name:     "raw",
		Renderer: "echo",
		Parts:    map[string]string{PartText: "RAW {{ x }}"},
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	// echo ignores Liquid syntax and data — proving the echo engine ran, not Liquid.
	out, err := e.RenderPart("raw", PartText, map[string]any{"x": "ignored"})
	if err != nil {
		t.Fatalf("RenderPart: %v", err)
	}
	if out != "RAW {{ x }}" {
		t.Fatalf("echo render = %q, want literal source", out)
	}
}

// fakeRuntime is the cnos read seam for tests — no env, no real cnos project.
type fakeRuntime map[string]any

func (f fakeRuntime) Value(path string) (any, bool, error) {
	v, ok := f[path]
	return v, ok, nil
}

func TestLoadFromCnosAndNewFromConfig(t *testing.T) {
	rt := fakeRuntime{
		"guten.default_renderer": "liquid",
		"guten.templates":        `[{"name":"welcome","parts":{"text":"Hi {{ name }}"}}]`,
	}
	cfg, err := LoadFrom(rt)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.DefaultRenderer != "liquid" {
		t.Fatalf("DefaultRenderer = %q, want liquid", cfg.DefaultRenderer)
	}
	if len(cfg.Templates) != 1 || cfg.Templates[0].Name != "welcome" {
		t.Fatalf("Templates = %+v, want one named welcome", cfg.Templates)
	}

	e, err := NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}
	out, err := e.RenderPart("welcome", PartText, map[string]any{"name": "Ada"})
	if err != nil {
		t.Fatalf("RenderPart: %v", err)
	}
	if out != "Hi Ada" {
		t.Fatalf("config template render = %q, want %q", out, "Hi Ada")
	}
}

// TestNewFromRuntimeStructuredTemplates exercises the non-string (structured)
// cnos value path for guten.templates.
func TestNewFromRuntimeStructuredTemplates(t *testing.T) {
	rt := fakeRuntime{
		"guten.templates": []any{
			map[string]any{"name": "x", "parts": map[string]any{"text": "v {{ a }}"}},
		},
	}
	e, err := NewFromRuntime(rt)
	if err != nil {
		t.Fatalf("NewFromRuntime: %v", err)
	}
	out, err := e.RenderPart("x", PartText, map[string]any{"a": "1"})
	if err != nil {
		t.Fatalf("RenderPart: %v", err)
	}
	if out != "v 1" {
		t.Fatalf("render = %q, want %q", out, "v 1")
	}
}

func TestLoadFromDefaultsWhenUnset(t *testing.T) {
	cfg, err := LoadFrom(fakeRuntime{})
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.DefaultRenderer != DefaultRenderer {
		t.Fatalf("DefaultRenderer = %q, want %q", cfg.DefaultRenderer, DefaultRenderer)
	}
	if len(cfg.Templates) != 0 {
		t.Fatalf("Templates = %+v, want empty", cfg.Templates)
	}
}
