package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/kitsyai/guten/cli/internal/library"
	pdfconv "github.com/kitsyai/guten/cli/internal/pdf"
	"github.com/kitsyai/guten/cli/internal/webui"
)

// heyContractVersion is the app-contract generation this server implements
// (see github.com/heypkv/hey docs/app-contract-v0.md).
const heyContractVersion = 0

type uiOpts struct {
	port   int
	json   bool
	noOpen bool
	libDir string
	chrome string
}

// cmdUI serves the embedded web UI plus a JSON API over loopback. With
// --json it prints the hey handshake line; otherwise it opens the browser.
func cmdUI(args []string) error {
	o := uiOpts{port: 4180}
	for i := 0; i < len(args); i++ {
		a := args[i]
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("missing value for %s", a)
			}
			i++
			return args[i], nil
		}
		var err error
		switch a {
		case "--port":
			var v string
			if v, err = next(); err == nil {
				o.port, err = strconv.Atoi(v)
			}
		case "--json":
			o.json = true
		case "--no-open":
			o.noOpen = true
		case "--lib-dir":
			o.libDir, err = next()
		case "--chrome":
			o.chrome, err = next()
		default:
			return fmt.Errorf("unknown flag %q", a)
		}
		if err != nil {
			return err
		}
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", o.port))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	serverURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	mux := http.NewServeMux()
	mux.Handle("/", webui.Handler())
	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"name": "guten", "version": version})
	})
	mux.HandleFunc("GET /api/templates", o.handleTemplates)
	mux.HandleFunc("GET /api/templates/{name}", o.handleTemplate)
	mux.HandleFunc("POST /api/render", o.handleRender)
	mux.HandleFunc("POST /api/export/pdf", o.handleExportPDF)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /hey/shutdown", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		go func() {
			time.Sleep(100 * time.Millisecond)
			os.Exit(0)
		}()
	})

	if o.json {
		// hey app contract: one flushed stdout line once the listener is bound.
		hs, _ := json.Marshal(map[string]any{
			"hey": 1, "name": "guten", "version": version,
			"url": serverURL, "pid": os.Getpid(), "port": port,
		})
		fmt.Println(string(hs))
	} else {
		fmt.Fprintf(os.Stderr, "guten ui at %s (Ctrl+C to stop)\n", serverURL)
		if !o.noOpen {
			openBrowser(serverURL)
		}
	}

	return http.Serve(ln, originGuard(port, mux))
}

// originGuard rejects cross-origin browser requests. The UI is same-origin,
// so any foreign Origin header means some other website is poking the local
// server — localhost is not a security boundary.
func originGuard(port int, next http.Handler) http.Handler {
	allowed := map[string]bool{
		fmt.Sprintf("http://127.0.0.1:%d", port): true,
		fmt.Sprintf("http://localhost:%d", port): true,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && !allowed[origin] {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func apiError(w http.ResponseWriter, status int, err error) {
	http.Error(w, err.Error(), status)
}

func (o uiOpts) handleTemplates(w http.ResponseWriter, r *http.Request) {
	entries := library.List(o.libDir)
	type entry struct {
		Name        string `json:"name"`
		Kind        string `json:"kind"`
		Source      string `json:"source"`
		Description string `json:"description"`
	}
	out := make([]entry, 0, len(entries))
	for _, e := range entries {
		out = append(out, entry{e.Name, e.Kind, e.Source, e.Description})
	}
	writeJSON(w, out)
}

func (o uiOpts) handleTemplate(w http.ResponseWriter, r *http.Request) {
	name, err := url.PathUnescape(r.PathValue("name"))
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	b, err := library.LoadBundle(name, o.libDir)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	parts := make([]string, 0, len(b.Template.Parts))
	for p := range b.Template.Parts {
		parts = append(parts, p)
	}
	writeJSON(w, map[string]any{
		"name":     b.Template.Name,
		"renderer": b.Template.Renderer,
		"parts":    parts,
		"sample":   b.Sample,
	})
}

type renderRequest struct {
	Lib  string          `json:"lib"`
	Data json.RawMessage `json:"data"`
	Part string          `json:"part"`
}

// toOpts translates an API render request into the CLI's opts, so the HTTP
// surface and the CLI share runRender exactly.
func (rr renderRequest) toOpts(o uiOpts) (opts, error) {
	if rr.Lib == "" {
		return opts{}, fmt.Errorf("lib is required")
	}
	ro := opts{lib: rr.Lib, libDir: o.libDir, chrome: o.chrome, renderer: "liquid", part: "html"}
	if rr.Part != "" {
		ro.part = rr.Part
	}
	if len(rr.Data) > 0 && string(rr.Data) != "null" {
		ro.data = string(rr.Data)
	}
	return ro, nil
}

func decodeRenderRequest(w http.ResponseWriter, r *http.Request) (renderRequest, bool) {
	var rr renderRequest
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	if err := json.NewDecoder(r.Body).Decode(&rr); err != nil {
		apiError(w, http.StatusBadRequest, fmt.Errorf("parse request: %w", err))
		return rr, false
	}
	return rr, true
}

func (o uiOpts) handleRender(w http.ResponseWriter, r *http.Request) {
	rr, ok := decodeRenderRequest(w, r)
	if !ok {
		return
	}
	ro, err := rr.toOpts(o)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	out, err := runRender(ro)
	if err != nil {
		apiError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, map[string]string{"output": out})
}

func (o uiOpts) handleExportPDF(w http.ResponseWriter, r *http.Request) {
	rr, ok := decodeRenderRequest(w, r)
	if !ok {
		return
	}
	rr.Part = "html"
	ro, err := rr.toOpts(o)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	htmlStr, err := runRender(ro)
	if err != nil {
		apiError(w, http.StatusUnprocessableEntity, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	pdf, err := pdfconv.NewChrome(o.chrome).ToPDF(ctx, []byte(htmlStr))
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", rr.Lib+".pdf"))
	_, _ = w.Write(pdf)
}

func openBrowser(u string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	case "darwin":
		cmd = exec.Command("open", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "could not open browser (%v) — open %s yourself\n", err, u)
		return
	}
	go func() { _ = cmd.Wait() }()
}
