package main

import (
	"os"
	"path/filepath"
	"testing"
)

func isolateHome(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("HOME", tmp)
}

func TestCmdNewScaffoldsFromBuiltin(t *testing.T) {
	isolateHome(t)
	if err := cmdNew([]string{"my-otp", "--from", "otp"}); err != nil {
		t.Fatalf("cmdNew: %v", err)
	}
	out, err := runRender(opts{lib: "my-otp", part: "html"})
	if err != nil {
		t.Fatalf("render scaffolded template: %v", err)
	}
	if out == "" {
		t.Fatal("expected non-empty rendered output for the cloned template")
	}
}

func TestCmdNewRequiresName(t *testing.T) {
	isolateHome(t)
	if err := cmdNew(nil); err == nil {
		t.Fatal("expected error with no name")
	}
	if err := cmdNew([]string{"--from", "otp"}); err == nil {
		t.Fatal("expected error when first arg is a flag, not a name")
	}
}

func TestCmdLibAddAndRm(t *testing.T) {
	isolateHome(t)

	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "template.json"),
		[]byte(`{"name":"note","parts":{"html":"<p>{{ x }}</p>"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := cmdLib([]string{"add", src}); err != nil {
		t.Fatalf("lib add: %v", err)
	}
	out, err := runRender(opts{lib: "note", part: "html", data: `{"x":"hi"}`})
	if err != nil {
		t.Fatalf("render added template: %v", err)
	}
	if out != "<p>hi</p>" {
		t.Fatalf("got %q", out)
	}

	if err := cmdLib([]string{"rm", "note"}); err != nil {
		t.Fatalf("lib rm: %v", err)
	}
	if _, err := runRender(opts{lib: "note", part: "html"}); err == nil {
		t.Fatal("expected removed template to no longer resolve")
	}
}

func TestCmdLibRmRefusesBuiltin(t *testing.T) {
	isolateHome(t)
	if err := cmdLib([]string{"rm", "invoice"}); err == nil {
		t.Fatal("expected error removing a builtin via CLI")
	}
}
