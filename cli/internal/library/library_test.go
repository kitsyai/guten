package library

import "testing"

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
		if e.Source != "embedded" {
			t.Fatalf("expected embedded source for %q, got %q", e.Name, e.Source)
		}
	}
	for _, want := range []string{"invoice", "otp", "welcome"} {
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
