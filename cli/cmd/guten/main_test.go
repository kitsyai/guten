package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	pdfconv "github.com/kitsyai/guten/cli/internal/pdf"
)

func TestRunRenderLiquid(t *testing.T) {
	out, err := runRender(opts{template: "Hi {{ name }}", renderer: "liquid", data: `{"name":"Ada"}`, part: "html"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hi Ada" {
		t.Fatalf("got %q", out)
	}
}

func TestRunRenderLayout(t *testing.T) {
	src := `{"blocks":[{"type":"heading","text":"{{ title }}"}]}`
	out, err := runRender(opts{template: src, renderer: "layout", data: `{"title":"Sale"}`, part: "html"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `<h1 class="guten-heading"`) || !strings.Contains(out, ">Sale<") {
		t.Fatalf("got %q", out)
	}
}

func TestRunExportHTMLAndText(t *testing.T) {
	dir := t.TempDir()
	htmlOut := filepath.Join(dir, "out.html")
	txtOut := filepath.Join(dir, "out.txt")
	mf := filepath.Join(dir, "tpl.json")
	if err := os.WriteFile(mf, []byte(`{"name":"t","parts":{"html":"<p>{{ x }}</p>","text":"{{ x }}"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	written, err := runExport(opts{manifest: "@" + mf, data: `{"x":"hi"}`, outs: []string{htmlOut, txtOut}, part: "html"})
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 2 {
		t.Fatalf("written %v", written)
	}
	if h, _ := os.ReadFile(htmlOut); string(h) != "<p>hi</p>" {
		t.Fatalf("html %q", h)
	}
	if tx, _ := os.ReadFile(txtOut); string(tx) != "hi" {
		t.Fatalf("txt %q", tx)
	}
}

func TestRunExportPDF(t *testing.T) {
	if pdfconv.DetectBrowser() == "" {
		t.Skip("no Chrome/Edge/Chromium available for PDF test")
	}
	dir := t.TempDir()
	pdfOut := filepath.Join(dir, "out.pdf")
	if _, err := runExport(opts{template: "<h1>Hello {{ n }}</h1>", data: `{"n":"PDF"}`, outs: []string{pdfOut}, part: "html", renderer: "liquid"}); err != nil {
		t.Fatalf("export pdf: %v", err)
	}
	b, _ := os.ReadFile(pdfOut)
	if len(b) < 5 || string(b[:5]) != "%PDF-" {
		t.Fatalf("not a pdf (%d bytes)", len(b))
	}
}

func TestRenderDataThemeAndSet(t *testing.T) {
	o := opts{
		data:  `{"theme":{"accent_color":"#111"},"name":"x"}`,
		theme: `{"font_family":"Georgia"}`,
		sets:  []string{"theme.accent_color=#0ea5e9", "footer.platform=acme.com"},
	}
	d, err := renderData(o)
	if err != nil {
		t.Fatal(err)
	}
	th := d["theme"].(map[string]any)
	if th["font_family"] != "Georgia" {
		t.Fatalf("theme merge failed: %v", th)
	}
	if th["accent_color"] != "#0ea5e9" {
		t.Fatalf("--set override failed: %v", th)
	}
	if f := d["footer"].(map[string]any); f["platform"] != "acme.com" {
		t.Fatalf("nested --set failed: %v", f)
	}
}

func TestInjectCSS(t *testing.T) {
	out, err := injectCSS("<html><head><title>x</title></head><body>y</body></html>", []string{"body{color:red}"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "body{color:red}") {
		t.Fatalf("css missing: %s", out)
	}
	if strings.Index(out, "body{color:red}") > strings.Index(out, "</head>") {
		t.Fatal("css must be injected before </head>")
	}
	if same, _ := injectCSS("<p>x</p>", nil); same != "<p>x</p>" {
		t.Fatal("no css should be a no-op")
	}
}
