package guten

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestLayoutRenderer(t *testing.T) {
	e := New()
	e.RegisterRenderer(NewLayoutRenderer())
	src := `{"style":{"accent":"#123456"},"blocks":[{"type":"heading","text":"{{ title }}"},{"type":"paragraph","text":"Hi {{ name }}"},{"type":"button","text":"Open","url":"{{ url }}"}]}`
	if err := e.Register(Template{Name: "promo", Renderer: RendererLayout, Parts: map[string]string{PartHTML: src}}); err != nil {
		t.Fatalf("register: %v", err)
	}
	out, err := e.RenderPart("promo", PartHTML, map[string]any{"title": "Sale", "name": "Ada", "url": "https://x.test"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{
		`<h1 class="guten-heading" style="color:#123456">Sale</h1>`,
		`<p class="guten-paragraph">Hi Ada</p>`,
		`<a class="guten-button" href="https://x.test" style="background:#123456">Open</a>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

// fakePDF stands in for an injected PDF converter (the seam consumers wire).
type fakePDF struct {
	err error
	got []byte
}

func (f *fakePDF) ToPDF(_ context.Context, html []byte) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.got = html
	return append([]byte("%PDF-1.4\n"), html...), nil
}

func TestRenderToPDF(t *testing.T) {
	e, _ := NewWithBuiltins()
	conv := &fakePDF{}
	pdf, err := e.RenderToPDF(context.Background(), "basic_notification", map[string]any{"title": "Hi", "body": "x"}, conv)
	if err != nil {
		t.Fatalf("RenderToPDF: %v", err)
	}
	if !strings.HasPrefix(string(pdf), "%PDF") {
		t.Fatalf("not pdf: %q", string(pdf))
	}
	if !strings.Contains(string(conv.got), "<h1") {
		t.Fatal("converter did not receive rendered html")
	}
	if _, err := e.RenderToPDF(context.Background(), "basic_notification", nil, nil); err == nil {
		t.Fatal("expected error for nil converter")
	}
	if _, err := e.RenderToPDF(context.Background(), "basic_notification", nil, &fakePDF{err: errors.New("boom")}); err == nil {
		t.Fatal("expected converter error")
	}
}
