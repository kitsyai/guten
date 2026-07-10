package library

import "testing"
import "path/filepath"
import "os"

func TestEmbeddedListAndLoad(t *testing.T) {
	// Isolate HOME so only the embedded snapshot is found — a real machine may
	// have ~/.kitsy/guten/{gutenkit,user} that would otherwise shadow it.
	tmp := t.TempDir()
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("HOME", tmp)

	entries := List("")
	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name] = true
		if e.Source != "builtin" {
			t.Fatalf("expected embedded source for %q, got %q", e.Name, e.Source)
		}
	}
	for _, want := range []string{"invoice", "invoice_bold", "otp", "welcome"} {
		if !names[want] {
			t.Fatalf("missing embedded template %q (have %v)", want, names)
		}
	}

	b, err := LoadBundle("otp", "")
	if err != nil {
		t.Fatal(err)
	}
	if b.Template.Name != "otp" {
		t.Fatalf("name = %q", b.Template.Name)
	}
	if b.Template.Parts["html"] == "" || b.Template.Parts["subject"] == "" {
		t.Fatalf("parts not resolved: %v", b.Template.Parts)
	}
	if b.Theme["accent_color"] != "#4f46e5" {
		t.Fatalf("theme.json not loaded: %v", b.Theme)
	}
	if len(b.Sample) == 0 {
		t.Fatal("sample.json not loaded")
	}
	if _, err := LoadBundle("does-not-exist", ""); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestLoadBundleResolvesBuiltInInvoiceBold(t *testing.T) {
	b, err := LoadBundle("invoice_bold", "")
	if err != nil {
		t.Fatalf("LoadBundle(invoice_bold): %v", err)
	}
	if b.Template.Name != "invoice_bold" {
		t.Fatalf("unexpected template name: %q", b.Template.Name)
	}
	if len(b.Template.Parts) == 0 || b.Template.Parts["html"] == "" {
		t.Fatalf("expected invoice_bold html part, got %v", b.Template.Parts)
	}
}

func TestParseTemplateRefSupportsBuiltinAndGutenkitPrefixes(t *testing.T) {
	ref, err := parseTemplateRef("invoice")
	if err != nil {
		t.Fatalf("parseTemplateRef(invoice): %v", err)
	}
	if ref.name != "invoice" || ref.gutenkit {
		t.Fatalf("unexpected parsed builtin ref: %+v", ref)
	}

	ref, err = parseTemplateRef("@gutenkit/invoice")
	if err != nil {
		t.Fatalf("parseTemplateRef(@gutenkit/invoice): %v", err)
	}
	if !ref.gutenkit || ref.name != "invoice" {
		t.Fatalf("unexpected parsed gutenkit ref: %+v", ref)
	}

	if _, err := parseTemplateRef("@other/invoice"); err == nil {
		t.Fatal("expected unsupported prefix to error")
	}
}

func TestLoadBundleResolvesGutenkitPrefixFromLibDir(t *testing.T) {
	tmp := t.TempDir()
	libDir := filepath.Join(tmp, "lib")
	templateDir := filepath.Join(libDir, "templates", "invoice")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	manifest := "{\"name\":\"invoice\",\"kind\":\"document\",\"parts\":{\"text\":\"Hi {{ to }}\\n\"}}"
	if err := os.WriteFile(filepath.Join(templateDir, "template.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write fixture template.json: %v", err)
	}

	_, err := LoadBundle("@gutenkit/invoice", libDir)
	if err != nil {
		t.Fatalf("LoadBundle(@gutenkit/invoice): %v", err)
	}
}
