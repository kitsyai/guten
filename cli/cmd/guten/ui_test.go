package main

import (
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
	mux.HandleFunc("POST /api/render", o.handleRender)
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
