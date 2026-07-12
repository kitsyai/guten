package library

import (
	"os"
	"path/filepath"
	"testing"
)

// isolateHome points HOME/USERPROFILE at a fresh temp dir so these tests
// never read or write the real ~/.kitsy/guten on the machine running them.
func isolateHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("HOME", tmp)
	return tmp
}

func TestNewUserTemplateFromBuiltin(t *testing.T) {
	isolateHome(t)

	dir, err := NewUserTemplate("my-invoice", "invoice", "")
	if err != nil {
		t.Fatalf("NewUserTemplate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "template.json")); err != nil {
		t.Fatalf("template.json missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "html.liquid")); err != nil {
		t.Fatalf("html.liquid missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "sample.json")); err != nil {
		t.Fatalf("sample.json missing: %v", err)
	}

	// The new template must round-trip through LoadBundle from the user tier.
	b, err := LoadBundle("my-invoice", "")
	if err != nil {
		t.Fatalf("LoadBundle(my-invoice): %v", err)
	}
	if b.Template.Parts["html"] == "" {
		t.Fatalf("cloned html part is empty")
	}
	if len(b.Sample) == 0 {
		t.Fatal("cloned sample is empty")
	}

	// Scaffolding on top of an existing name refuses to clobber it.
	if _, err := NewUserTemplate("my-invoice", "invoice", ""); err == nil {
		t.Fatal("expected error re-creating an existing user template")
	}
}

func TestNewUserTemplateWithoutFrom(t *testing.T) {
	isolateHome(t)

	dir, err := NewUserTemplate("blank", "", "")
	if err != nil {
		t.Fatalf("NewUserTemplate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "html.liquid")); err != nil {
		t.Fatalf("html.liquid missing: %v", err)
	}
	b, err := LoadBundle("blank", "")
	if err != nil {
		t.Fatalf("LoadBundle(blank): %v", err)
	}
	if b.Template.Parts["html"] == "" {
		t.Fatal("starter html part should not be empty")
	}
}

func TestNewUserTemplateRejectsBadNames(t *testing.T) {
	isolateHome(t)
	for _, name := range []string{"", "a/b", "a\\b", ".", ".."} {
		if _, err := NewUserTemplate(name, "", ""); err == nil {
			t.Fatalf("expected error for invalid name %q", name)
		}
	}
}

func TestAddUserTemplateCopiesDirAndRefusesDuplicate(t *testing.T) {
	isolateHome(t)

	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "template.json"),
		[]byte(`{"name":"acme-note","parts":{"html":"<p>{{ x }}</p>"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sample.json"), []byte(`{"x":"hi"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	dst, err := AddUserTemplate(src)
	if err != nil {
		t.Fatalf("AddUserTemplate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "template.json")); err != nil {
		t.Fatalf("copied template.json missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "sample.json")); err != nil {
		t.Fatalf("copied sample.json missing: %v", err)
	}

	b, err := LoadBundle("acme-note", "")
	if err != nil {
		t.Fatalf("LoadBundle(acme-note): %v", err)
	}
	if b.Template.Parts["html"] != "<p>{{ x }}</p>" {
		t.Fatalf("unexpected html part: %q", b.Template.Parts["html"])
	}

	if _, err := AddUserTemplate(src); err == nil {
		t.Fatal("expected error adding a duplicate name")
	}
}

func TestAddUserTemplateRequiresManifest(t *testing.T) {
	isolateHome(t)
	src := t.TempDir() // no template.json
	if _, err := AddUserTemplate(src); err == nil {
		t.Fatal("expected error for a dir without template.json")
	}
}

func TestRemoveUserTemplate(t *testing.T) {
	isolateHome(t)

	if _, err := NewUserTemplate("throwaway", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := RemoveUserTemplate("throwaway"); err != nil {
		t.Fatalf("RemoveUserTemplate: %v", err)
	}
	if _, err := LoadBundle("throwaway", ""); err == nil {
		t.Fatal("expected removed template to no longer resolve")
	}
}

func TestRemoveUserTemplateRefusesBuiltinsAndUnknownNames(t *testing.T) {
	isolateHome(t)

	if err := RemoveUserTemplate("invoice"); err == nil {
		t.Fatal("expected error removing a builtin")
	}
	// A builtin must still resolve after the refused removal attempt.
	if _, err := LoadBundle("invoice", ""); err != nil {
		t.Fatalf("builtin invoice should still resolve: %v", err)
	}

	if err := RemoveUserTemplate("does-not-exist-anywhere"); err == nil {
		t.Fatal("expected error removing an unknown template")
	}
}

func TestIsBuiltin(t *testing.T) {
	if !IsBuiltin("invoice") {
		t.Fatal("invoice should be a builtin")
	}
	if IsBuiltin("definitely-not-a-template") {
		t.Fatal("unknown name should not report as builtin")
	}
}
