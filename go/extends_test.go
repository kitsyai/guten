package guten

import "testing"

func TestExtends(t *testing.T) {
	e := New()
	if err := e.Register(Template{Name: "email_base", Parts: map[string]string{
		"subject": "{{ subject }}",
		"html":    "<main>{{ body }}</main>",
		"text":    "{{ body }}",
	}}); err != nil {
		t.Fatal(err)
	}
	// welcome inherits subject+text, overrides only html.
	if err := e.Register(Template{Name: "welcome", Extends: "email_base", Parts: map[string]string{
		"html": "<div>Welcome</div><main>{{ body }}</main>",
	}}); err != nil {
		t.Fatal(err)
	}
	r, err := e.Render("welcome", map[string]any{"subject": "Hi", "body": "Ready"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Parts["subject"] != "Hi" {
		t.Fatalf("subject should be inherited: %q", r.Parts["subject"])
	}
	if r.Parts["text"] != "Ready" {
		t.Fatalf("text should be inherited: %q", r.Parts["text"])
	}
	if r.Parts["html"] != "<div>Welcome</div><main>Ready</main>" {
		t.Fatalf("html should be overridden: %q", r.Parts["html"])
	}
	if err := e.Register(Template{Name: "x", Extends: "nope", Parts: map[string]string{"html": "y"}}); err == nil {
		t.Fatal("expected error extending an unknown base")
	}
}

// Slots are a pure data convention (no engine change): a base renders
// {{ slots.* | default: ... }}; a child or the CLI fills slots via data.
func TestSlotsData(t *testing.T) {
	e := New()
	if err := e.Register(Template{Name: "doc", Parts: map[string]string{
		"html": "<h>{{ slots.header | default: 'Default' }}</h>",
	}}); err != nil {
		t.Fatal(err)
	}
	out, _ := e.RenderPart("doc", "html", nil)
	if out != "<h>Default</h>" {
		t.Fatalf("default slot: %q", out)
	}
	out, _ = e.RenderPart("doc", "html", map[string]any{"slots": map[string]any{"header": "Custom"}})
	if out != "<h>Custom</h>" {
		t.Fatalf("filled slot: %q", out)
	}
}
