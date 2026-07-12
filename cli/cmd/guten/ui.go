package main

import (
	"archive/zip"
	"bytes"
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
	"strings"
	"time"

	"github.com/kitsyai/guten/cli/internal/library"
	pdfconv "github.com/kitsyai/guten/cli/internal/pdf"
	"github.com/kitsyai/guten/cli/internal/webui"
	guten "github.com/kitsyai/guten/go"
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
	mux.HandleFunc("POST /api/templates", o.handleSaveTemplate)
	mux.HandleFunc("POST /api/render", o.handleRender)
	mux.HandleFunc("POST /api/export/pdf", o.handleExportPDF)
	mux.HandleFunc("POST /api/batch", o.handleBatch)
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
	full, err := effectiveParts(b, o.libDir)
	if err != nil {
		apiError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, map[string]any{
		"name":        b.Template.Name,
		"renderer":    b.Template.Renderer,
		"extends":     b.Template.Extends,
		"parts":       parts,
		"partSources": full,
		"sample":      b.Sample,
		"theme":       b.Theme,
		"builtin":     library.IsBuiltin(name),
	})
}

// effectiveParts resolves a bundle's parts including anything it extends (a
// base overlaid, per-part, by the child), so the "duplicate & edit" UI always
// shows each part's final effective source rather than just this bundle's
// own overrides.
func effectiveParts(b library.Bundle, libDir string) (map[string]string, error) {
	parts := map[string]string{}
	if b.Template.Extends != "" {
		base, err := library.LoadBundle(b.Template.Extends, libDir)
		if err != nil {
			return nil, fmt.Errorf("load base %q: %w", b.Template.Extends, err)
		}
		baseParts, err := effectiveParts(base, libDir)
		if err != nil {
			return nil, err
		}
		for k, v := range baseParts {
			parts[k] = v
		}
	}
	for k, v := range b.Template.Parts {
		parts[k] = v
	}
	return parts, nil
}

// saveTemplateRequest is the "duplicate & edit" save step: the browser has
// edited a builtin's (or any template's) liquid part(s) and/or sample data,
// and wants it saved into the user tier under a (possibly new) name.
type saveTemplateRequest struct {
	Name     string            `json:"name"`
	Renderer string            `json:"renderer"`
	Parts    map[string]string `json:"parts"`
	Sample   json.RawMessage   `json:"sample"`
	Theme    json.RawMessage   `json:"theme"`
}

// handleSaveTemplate always writes to the user tier (library.SaveUserTemplate),
// never to the gutenkit cache or the embedded builtins — builtins stay
// read-only; this is where edits land, per the workbench spec's guardrail.
// Saving under a builtin's own name is allowed: it creates a user-tier
// override that shadows the builtin in the library search order without
// modifying the builtin itself.
func (o uiOpts) handleSaveTemplate(w http.ResponseWriter, r *http.Request) {
	var req saveTemplateRequest
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, http.StatusBadRequest, fmt.Errorf("parse request: %w", err))
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("name is required"))
		return
	}
	if len(req.Parts) == 0 {
		apiError(w, http.StatusBadRequest, fmt.Errorf("at least one part is required"))
		return
	}
	var sample map[string]any
	if len(req.Sample) > 0 {
		if err := json.Unmarshal(req.Sample, &sample); err != nil {
			apiError(w, http.StatusBadRequest, fmt.Errorf("parse sample: %w", err))
			return
		}
	}
	var theme map[string]any
	if len(req.Theme) > 0 {
		if err := json.Unmarshal(req.Theme, &theme); err != nil {
			apiError(w, http.StatusBadRequest, fmt.Errorf("parse theme: %w", err))
			return
		}
	}
	dir, err := library.SaveUserTemplate(req.Name, req.Renderer, req.Parts, sample, theme)
	if err != nil {
		apiError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, map[string]string{"name": req.Name, "dir": dir})
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

// batchAPIRequest is the UI's batch-render request: rows pasted/uploaded as
// raw JSONL or CSV text (format defaults to jsonl), a template selector, and
// the same --name filename template the CLI's `batch` command takes.
type batchAPIRequest struct {
	Lib    string `json:"lib"`
	Rows   string `json:"rows"`
	Format string `json:"format"`
	Name   string `json:"name"`
}

type batchRowErrorJSON struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// handleBatch renders every row against Lib and streams back a zip: one file
// per successful row (named by rendering the Name template against that
// row's data), plus an _errors.json listing any row failures — the run never
// stops early, exactly like `guten batch`. It only fails the whole request
// (4xx/5xx, no body) for request-shaped problems or when literally every row
// failed.
func (o uiOpts) handleBatch(w http.ResponseWriter, r *http.Request) {
	var req batchAPIRequest
	r.Body = http.MaxBytesReader(w, r.Body, 32<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, http.StatusBadRequest, fmt.Errorf("parse request: %w", err))
		return
	}
	if req.Lib == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("lib is required"))
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("name (filename template) is required"))
		return
	}

	var rows []parsedRow
	var err error
	if strings.EqualFold(req.Format, "csv") {
		rows, err = parseCSVRows(req.Rows)
	} else {
		rows, err = parseJSONLRows(req.Rows)
	}
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if len(rows) == 0 {
		apiError(w, http.StatusBadRequest, fmt.Errorf("no rows found"))
		return
	}

	base := opts{lib: req.Lib, libDir: o.libDir, chrome: o.chrome, renderer: guten.RendererLiquid, part: guten.PartHTML}
	e, name, baseTheme, baseSample, err := engineAndTemplate(base)
	if err != nil {
		apiError(w, http.StatusUnprocessableEntity, err)
		return
	}
	nameTpl, err := guten.NewLiquidRenderer().Compile(req.Name)
	if err != nil {
		apiError(w, http.StatusBadRequest, fmt.Errorf("parse name template: %w", err))
		return
	}
	part := partForExt(req.Name)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	var rowErrs []batchRowErrorJSON
	written := 0
	for _, pr := range rows {
		if pr.err != nil {
			rowErrs = append(rowErrs, batchRowErrorJSON{Row: pr.n, Message: pr.err.Error()})
			continue
		}
		rowJSON, merr := json.Marshal(pr.data)
		if merr != nil {
			rowErrs = append(rowErrs, batchRowErrorJSON{Row: pr.n, Message: merr.Error()})
			continue
		}
		rowOpts := base
		rowOpts.data = string(rowJSON)
		rowOpts.part = part
		data, derr := renderData(rowOpts, baseTheme, baseSample)
		if derr != nil {
			rowErrs = append(rowErrs, batchRowErrorJSON{Row: pr.n, Message: derr.Error()})
			continue
		}
		filename, ferr := nameTpl.Render(data)
		if ferr != nil {
			rowErrs = append(rowErrs, batchRowErrorJSON{Row: pr.n, Message: "render filename: " + ferr.Error()})
			continue
		}
		filename = strings.TrimSpace(filename)
		if filename == "" {
			rowErrs = append(rowErrs, batchRowErrorJSON{Row: pr.n, Message: "rendered filename is empty"})
			continue
		}
		payload, perr := renderPayload(ctx, e, name, part, data, nil, o.chrome)
		if perr != nil {
			rowErrs = append(rowErrs, batchRowErrorJSON{Row: pr.n, Message: perr.Error()})
			continue
		}
		fw, cerr := zw.Create(filename)
		if cerr != nil {
			rowErrs = append(rowErrs, batchRowErrorJSON{Row: pr.n, Message: cerr.Error()})
			continue
		}
		if _, werr := fw.Write(payload); werr != nil {
			rowErrs = append(rowErrs, batchRowErrorJSON{Row: pr.n, Message: werr.Error()})
			continue
		}
		written++
	}
	if len(rowErrs) > 0 {
		if fw, cerr := zw.Create("_errors.json"); cerr == nil {
			b, _ := json.MarshalIndent(rowErrs, "", "  ")
			_, _ = fw.Write(b)
		}
	}
	if cerr := zw.Close(); cerr != nil {
		apiError(w, http.StatusInternalServerError, cerr)
		return
	}
	if written == 0 {
		apiError(w, http.StatusUnprocessableEntity, fmt.Errorf("all %d row(s) failed", len(rows)))
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", req.Lib+"-batch.zip"))
	w.Header().Set("X-Batch-Total", strconv.Itoa(len(rows)))
	w.Header().Set("X-Batch-Written", strconv.Itoa(written))
	w.Header().Set("X-Batch-Errors", strconv.Itoa(len(rowErrs)))
	_, _ = w.Write(buf.Bytes())
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
