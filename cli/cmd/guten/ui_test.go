package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func uiTestMux(t *testing.T) *http.ServeMux {
	t.Helper()
	o := uiOpts{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/templates", o.handleTemplates)
	mux.HandleFunc("GET /api/templates/{name}", o.handleTemplate)
	mux.HandleFunc("POST /api/templates", o.handleSaveTemplate)
	mux.HandleFunc("POST /api/render", o.handleRender)
	mux.HandleFunc("POST /api/batch", o.handleBatch)
	return mux
}

func TestUITemplatesList(t *testing.T) {
	rec := httptest.NewRecorder()
	uiTestMux(t).ServeHTTP(rec, httptest.NewRequest("GET", "/api/templates", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"invoice"`) {
		t.Errorf("template list should include invoice: %s", rec.Body.String())
	}
}

func TestUITemplateBundle(t *testing.T) {
	rec := httptest.NewRecorder()
	uiTestMux(t).ServeHTTP(rec, httptest.NewRequest("GET", "/api/templates/invoice", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"sample"`) || !strings.Contains(body, `"html"`) {
		t.Errorf("bundle should expose sample and html part: %s", body)
	}
}

func TestUIRenderWithSampleFallback(t *testing.T) {
	// No data: renderData falls back to the bundle sample.
	req := httptest.NewRequest("POST", "/api/render",
		strings.NewReader(`{"lib":"invoice"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	uiTestMux(t).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Tax Invoice") {
		t.Error("rendered sample invoice should contain its title")
	}
}

func TestUIRenderErrors(t *testing.T) {
	cases := []struct {
		name, body string
		status     int
	}{
		{"missing lib", `{}`, http.StatusBadRequest},
		{"bad json", `{`, http.StatusBadRequest},
		{"unknown template", `{"lib":"no-such-template"}`, http.StatusUnprocessableEntity},
	}
	for _, c := range cases {
		req := httptest.NewRequest("POST", "/api/render", strings.NewReader(c.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		uiTestMux(t).ServeHTTP(rec, req)
		if rec.Code != c.status {
			t.Errorf("%s: status %d, want %d (%s)", c.name, rec.Code, c.status, rec.Body.String())
		}
	}
}

func TestUITemplateBundleExposesPartSources(t *testing.T) {
	rec := httptest.NewRecorder()
	uiTestMux(t).ServeHTTP(rec, httptest.NewRequest("GET", "/api/templates/invoice", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"partSources"`) {
		t.Fatalf("expected partSources in response: %s", body)
	}
	if !strings.Contains(body, `"builtin":true`) {
		t.Fatalf("expected builtin:true for invoice: %s", body)
	}
}

func TestUISaveTemplateWritesUserTierAndIsImmediatelyRenderable(t *testing.T) {
	isolateHome(t)

	body := `{"name":"my-note","renderer":"liquid","parts":{"html":"<p>{{ x }}</p>"},"sample":{"x":"hi"}}`
	req := httptest.NewRequest("POST", "/api/templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	uiTestMux(t).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	renderReq := httptest.NewRequest("POST", "/api/render", strings.NewReader(`{"lib":"my-note","data":{"x":"hello"}}`))
	renderReq.Header.Set("Content-Type", "application/json")
	renderRec := httptest.NewRecorder()
	uiTestMux(t).ServeHTTP(renderRec, renderReq)
	if renderRec.Code != http.StatusOK {
		t.Fatalf("render status %d: %s", renderRec.Code, renderRec.Body.String())
	}
	var out struct {
		Output string `json:"output"`
	}
	if err := json.Unmarshal(renderRec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode render response: %v (%s)", err, renderRec.Body.String())
	}
	if out.Output != "<p>hello</p>" {
		t.Fatalf("expected rendered saved template, got %q", out.Output)
	}
}

func TestUISaveTemplateRequiresNameAndParts(t *testing.T) {
	isolateHome(t)
	cases := []string{
		`{}`,
		`{"name":"x"}`,
		`{"name":"","parts":{"html":"x"}}`,
	}
	for _, body := range cases {
		req := httptest.NewRequest("POST", "/api/templates", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		uiTestMux(t).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body %q: status %d, want 400", body, rec.Code)
		}
	}
}

func TestUIBatchReturnsZipWithOneEntryPerRow(t *testing.T) {
	body := `{"lib":"invoice","rows":"{\"invoice\":{\"number\":\"A1\"}}\n{\"invoice\":{\"number\":\"A2\"}}","name":"{{ invoice.number }}.html"}`
	req := httptest.NewRequest("POST", "/api/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	uiTestMux(t).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/zip" {
		t.Fatalf("content-type = %q", ct)
	}
	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	if !names["A1.html"] || !names["A2.html"] {
		t.Fatalf("expected A1.html and A2.html in zip, got %v", names)
	}
	if names["_errors.json"] {
		t.Fatalf("did not expect _errors.json when all rows succeed: %v", names)
	}
}

func TestUIBatchContinuesPastBadRowAndReportsErrors(t *testing.T) {
	body := `{"lib":"invoice","rows":"{\"invoice\":{\"number\":\"GOOD\"}}\nnot json","name":"{{ invoice.number }}.html"}`
	req := httptest.NewRequest("POST", "/api/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	uiTestMux(t).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Batch-Errors") != "1" {
		t.Fatalf("X-Batch-Errors = %q", rec.Header().Get("X-Batch-Errors"))
	}
	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}
	var errsRaw []byte
	found := false
	for _, f := range zr.File {
		if f.Name == "GOOD.html" {
			found = true
		}
		if f.Name == "_errors.json" {
			rc, _ := f.Open()
			errsRaw, _ = io.ReadAll(rc)
			rc.Close()
		}
	}
	if !found {
		t.Fatal("expected GOOD.html to still be written")
	}
	if !strings.Contains(string(errsRaw), `"row": 2`) {
		t.Fatalf("expected row 2 in _errors.json, got %s", errsRaw)
	}
}

func TestUIBatchRequiresLibAndName(t *testing.T) {
	cases := []string{
		`{"rows":"{}","name":"x.html"}`,
		`{"lib":"invoice","rows":"{}"}`,
		`{"lib":"invoice","name":"x.html","rows":""}`,
	}
	for _, body := range cases {
		req := httptest.NewRequest("POST", "/api/batch", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		uiTestMux(t).ServeHTTP(rec, req)
		if rec.Code < 400 {
			t.Errorf("body %q: status %d, want 4xx", body, rec.Code)
		}
	}
}

func TestUIOriginGuard(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	guard := originGuard(4180, inner)

	req := httptest.NewRequest("POST", "/api/render", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	guard.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("foreign origin: status %d, want 403", rec.Code)
	}

	req = httptest.NewRequest("POST", "/api/render", nil)
	req.Header.Set("Origin", "http://127.0.0.1:4180")
	rec = httptest.NewRecorder()
	guard.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("same origin: status %d, want 200", rec.Code)
	}

	req = httptest.NewRequest("GET", "/healthz", nil) // no Origin header
	rec = httptest.NewRecorder()
	guard.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("no origin: status %d, want 200", rec.Code)
	}
}
