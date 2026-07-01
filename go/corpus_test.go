package guten

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// corpusCase is one shared parity case: a template + data. The rendered output
// (spec/corpus/expected.json) is the golden the Go and JS runtimes must both
// match, which proves cross-runtime parity.
type corpusCase struct {
	Name     string         `json:"name"`
	Template Template       `json:"template"`
	Data     map[string]any `json:"data"`
}

func loadCorpus(t *testing.T) []corpusCase {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "spec", "corpus", "cases.json"))
	if err != nil {
		t.Fatalf("read cases: %v", err)
	}
	var cases []corpusCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("parse cases: %v", err)
	}
	return cases
}

func renderCorpus(t *testing.T, cases []corpusCase) map[string]map[string]string {
	t.Helper()
	out := make(map[string]map[string]string, len(cases))
	for _, c := range cases {
		e := New()
		e.RegisterRenderer(NewLayoutRenderer())
		if err := e.Register(c.Template); err != nil {
			t.Fatalf("register %s: %v", c.Name, err)
		}
		r, err := e.Render(c.Template.Name, c.Data)
		if err != nil {
			t.Fatalf("render %s: %v", c.Name, err)
		}
		out[c.Name] = r.Parts
	}
	return out
}

// TestCorpusParity renders the shared corpus and compares against the golden
// spec/corpus/expected.json. Run with GUTEN_CORPUS_WRITE=1 to (re)generate the
// golden from the Go reference; the JS runtime asserts against the same file.
func TestCorpusParity(t *testing.T) {
	cases := loadCorpus(t)
	rendered := renderCorpus(t, cases)
	expectedPath := filepath.Join("..", "spec", "corpus", "expected.json")

	if os.Getenv("GUTEN_CORPUS_WRITE") == "1" {
		raw, err := json.MarshalIndent(rendered, "", "  ")
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if err := os.WriteFile(expectedPath, append(raw, '\n'), 0o644); err != nil {
			t.Fatalf("write expected: %v", err)
		}
		t.Logf("wrote %s (%d cases)", expectedPath, len(cases))
		return
	}

	raw, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read expected (run with GUTEN_CORPUS_WRITE=1 to generate): %v", err)
	}
	var expected map[string]map[string]string
	if err := json.Unmarshal(raw, &expected); err != nil {
		t.Fatalf("parse expected: %v", err)
	}
	for name, parts := range rendered {
		for part, got := range parts {
			if want := expected[name][part]; got != want {
				t.Fatalf("case %s part %s: Go output != golden\n got:  %q\n want: %q", name, part, got, want)
			}
		}
	}
}
