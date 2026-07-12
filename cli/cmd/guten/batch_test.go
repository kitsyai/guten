package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeRowsFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRunBatchJSONLInvoiceWritesOneFilePerRow(t *testing.T) {
	dir := t.TempDir()
	rows := strings.Join([]string{
		`{"invoice":{"number":"INV-0001"}}`,
		`{"invoice":{"number":"INV-0002"}}`,
		`{"invoice":{"number":"INV-0003"}}`,
	}, "\n")
	rowsPath := writeRowsFile(t, dir, "rows.jsonl", rows)
	outDir := filepath.Join(dir, "out")

	res, err := runBatch(context.Background(), batchOpts{
		lib:    "invoice",
		data:   "@" + rowsPath,
		name:   "{{ invoice.number }}.html",
		outDir: outDir,
	})
	if err != nil {
		t.Fatalf("runBatch: %v (errors: %v)", err, res.Errors)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("expected no row errors, got %v", res.Errors)
	}
	if len(res.Written) != 3 {
		t.Fatalf("expected 3 files written, got %d: %v", len(res.Written), res.Written)
	}
	for _, want := range []string{"INV-0001.html", "INV-0002.html", "INV-0003.html"} {
		p := filepath.Join(outDir, want)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", p, err)
		}
		if !strings.Contains(strings.ToLower(string(b)), "<!doctype html") {
			t.Fatalf("%s does not look like rendered invoice html", p)
		}
	}
}

func TestRunBatchContinuesPastBadRow(t *testing.T) {
	dir := t.TempDir()
	rows := strings.Join([]string{
		`{"invoice":{"number":"INV-GOOD-1"}}`,
		`not json`,
		`{"invoice":{"number":"INV-GOOD-2"}}`,
	}, "\n")
	rowsPath := writeRowsFile(t, dir, "rows.jsonl", rows)
	outDir := filepath.Join(dir, "out")

	res, err := runBatch(context.Background(), batchOpts{
		lib:    "invoice",
		data:   "@" + rowsPath,
		name:   "{{ invoice.number }}.html",
		outDir: outDir,
	})
	if err == nil {
		t.Fatal("expected a non-nil error since a row failed")
	}
	if len(res.Errors) != 1 {
		t.Fatalf("expected exactly 1 row error, got %v", res.Errors)
	}
	if res.Errors[0].Row != 2 {
		t.Fatalf("expected row 2 (the malformed JSON line) to fail, got row %d: %v", res.Errors[0].Row, res.Errors[0])
	}
	if len(res.Written) != 2 {
		t.Fatalf("expected the other 2 rows to still succeed, got %v", res.Written)
	}
	for _, want := range []string{"INV-GOOD-1.html", "INV-GOOD-2.html"} {
		if _, err := os.Stat(filepath.Join(outDir, want)); err != nil {
			t.Fatalf("expected %s to exist: %v", want, err)
		}
	}
}

func TestRunBatchContinuesPastRowRenderFailure(t *testing.T) {
	dir := t.TempDir()
	// All three rows are valid JSON so loadRows succeeds; row 2's "n": 0
	// makes the template's own "divided_by" filter fail at render time,
	// exercising the row-continuation path for failures that surface deeper
	// than JSON parsing (e.g. bad/missing fields for the chosen template).
	rows := strings.Join([]string{
		`{"i":1,"n":2}`,
		`{"i":2,"n":0}`,
		`{"i":3,"n":5}`,
	}, "\n")
	rowsPath := writeRowsFile(t, dir, "rows.jsonl", rows)
	outDir := filepath.Join(dir, "out")

	res, err := runBatch(context.Background(), batchOpts{
		template: "{{ 10 | divided_by: n }}",
		renderer: "liquid",
		data:     "@" + rowsPath,
		name:     "row-{{ i }}.html",
		outDir:   outDir,
	})
	if err == nil {
		t.Fatal("expected error since a row failed")
	}
	if len(res.Errors) != 1 {
		t.Fatalf("expected exactly 1 row error, got %v", res.Errors)
	}
	if res.Errors[0].Row != 2 {
		t.Fatalf("expected row 2 to fail, got row %d: %v", res.Errors[0].Row, res.Errors[0])
	}
	if len(res.Written) != 2 {
		t.Fatalf("expected the other 2 rows to still succeed, got %v", res.Written)
	}
	for _, want := range []string{"row-1.html", "row-3.html"} {
		if _, err := os.Stat(filepath.Join(outDir, want)); err != nil {
			t.Fatalf("expected %s to exist: %v", want, err)
		}
	}
}

func TestLoadRowsBadJSONLReportsRowNumber(t *testing.T) {
	dir := t.TempDir()
	rowsPath := writeRowsFile(t, dir, "rows.jsonl", "{\"a\":1}\nnot json\n")
	rows, err := loadRows("@" + rowsPath)
	if err != nil {
		t.Fatalf("loadRows should not fail outright: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].err != nil {
		t.Fatalf("row 1 should have parsed cleanly: %v", rows[0].err)
	}
	if rows[1].n != 2 || rows[1].err == nil {
		t.Fatalf("row 2 should carry a parse error: %+v", rows[1])
	}
}

func TestLoadRowsCSV(t *testing.T) {
	dir := t.TempDir()
	rowsPath := writeRowsFile(t, dir, "rows.csv", "name,amount\nAda,10\nGrace,20\n")
	rows, err := loadRows("@" + rowsPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].data["name"] != "Ada" || rows[0].data["amount"] != "10" {
		t.Fatalf("row 0 = %v", rows[0].data)
	}
	if rows[1].data["name"] != "Grace" {
		t.Fatalf("row 1 = %v", rows[1].data)
	}
}

func TestRunBatchRequiresNameAndOutDir(t *testing.T) {
	dir := t.TempDir()
	rowsPath := writeRowsFile(t, dir, "rows.jsonl", `{"a":1}`)
	if _, err := runBatch(context.Background(), batchOpts{lib: "invoice", data: "@" + rowsPath, outDir: dir}); err == nil {
		t.Fatal("expected error for missing --name")
	}
	if _, err := runBatch(context.Background(), batchOpts{lib: "invoice", data: "@" + rowsPath, name: "{{ x }}.html"}); err == nil {
		t.Fatal("expected error for missing -o")
	}
}

func TestParseBatchOpts(t *testing.T) {
	o, err := parseBatchOpts([]string{"--lib", "invoice", "-d", "@rows.jsonl", "--name", "{{ x }}.pdf", "-o", "out/"})
	if err != nil {
		t.Fatal(err)
	}
	if o.lib != "invoice" || o.data != "@rows.jsonl" || o.name != "{{ x }}.pdf" || o.outDir != "out/" {
		t.Fatalf("parsed opts = %+v", o)
	}
	if _, err := parseBatchOpts([]string{"--nope"}); err == nil {
		t.Fatal("expected error for unknown flag")
	}
}
