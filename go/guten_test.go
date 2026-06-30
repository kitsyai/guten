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
}

func TestRenderUnknownTemplate(t *testing.T) {
	e := New()
	if _, err := e.Render("nope", nil); err == nil {
		t.Fatal("expected error rendering unregistered template")
	}
}
