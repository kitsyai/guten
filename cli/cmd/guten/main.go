// Command guten is the CLI for the guten templating engine: render templates to
// stdout, or export them to html/text/pdf files. It exercises the canonical Go
// engine, so it is also handy for validating templates and their cross-runtime
// output.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pdfconv "github.com/kitsyai/guten/cli/internal/pdf"
	guten "github.com/kitsyai/guten/go"
)

var version = "0.2.0"

const usageText = `guten ` + `— templating engine CLI

Usage:
  guten render  -t <src|@file> [-r liquid|layout] [-d <json|@file>] [--part html]
  guten export  -t <src|@file> [-r liquid|layout] [-d <json|@file>] -o <file> [-o <file> ...]
  guten builtins
  guten version

Flags:
  -t, --template   template source, or @path to a file
      --manifest   @path to a full Template JSON {name, renderer, parts:{...}}
  -r, --renderer   renderer: liquid (default) | layout
  -d, --data       render data as JSON, or @path to a JSON file
      --part       part to render for 'render'/stdout (default: html)
  -o, --out        output file (repeatable); part inferred from extension:
                   .html/.htm -> html, .txt -> text, .pdf -> html rendered then converted
      --chrome     Chrome/Edge/Chromium path for PDF (else auto-detected; env GUTEN_CHROME)

Examples:
  guten render -t 'Hi {{ name }}' -d '{"name":"Ada"}'
  guten export -t @invoice.html -d @invoice.json -o invoice.html -o invoice.pdf
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "render":
		err = cmdRender(os.Args[2:])
	case "export":
		err = cmdExport(os.Args[2:])
	case "builtins":
		err = cmdBuiltins()
	case "version", "--version", "-v":
		fmt.Println("guten " + version)
	case "help", "-h", "--help":
		fmt.Print(usageText)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usageText)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "guten:", err)
		os.Exit(1)
	}
}

type opts struct {
	template string
	manifest string
	renderer string
	data     string
	part     string
	outs     []string
	chrome   string
}

func parseOpts(args []string) (opts, error) {
	o := opts{renderer: guten.RendererLiquid, part: guten.PartHTML}
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
		case "-t", "--template":
			o.template, err = next()
		case "--manifest":
			o.manifest, err = next()
		case "-r", "--renderer":
			o.renderer, err = next()
		case "-d", "--data":
			o.data, err = next()
		case "--part":
			o.part, err = next()
		case "-o", "--out":
			var v string
			if v, err = next(); err == nil {
				o.outs = append(o.outs, v)
			}
		case "--chrome":
			o.chrome, err = next()
		default:
			return o, fmt.Errorf("unknown flag %q", a)
		}
		if err != nil {
			return o, err
		}
	}
	return o, nil
}

// loadArg returns s, or the contents of the file when s begins with '@'.
func loadArg(s string) (string, error) {
	if strings.HasPrefix(s, "@") {
		b, err := os.ReadFile(s[1:])
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return s, nil
}

func loadData(s string) (map[string]any, error) {
	if strings.TrimSpace(s) == "" {
		return map[string]any{}, nil
	}
	raw, err := loadArg(s)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, fmt.Errorf("parse data: %w", err)
	}
	return m, nil
}

// engineAndTemplate builds an engine (Liquid + layout renderers) and registers
// the template from either --manifest or -t/--part, returning the template name.
func engineAndTemplate(o opts) (*guten.Engine, string, error) {
	e := guten.New()
	e.RegisterRenderer(guten.NewLayoutRenderer())
	if o.manifest != "" {
		raw, err := loadArg(o.manifest)
		if err != nil {
			return nil, "", err
		}
		var t guten.Template
		if err := json.Unmarshal([]byte(raw), &t); err != nil {
			return nil, "", fmt.Errorf("parse manifest: %w", err)
		}
		if err := e.Register(t); err != nil {
			return nil, "", err
		}
		return e, t.Name, nil
	}
	if o.template == "" {
		return nil, "", fmt.Errorf("one of -t/--template or --manifest is required")
	}
	src, err := loadArg(o.template)
	if err != nil {
		return nil, "", err
	}
	t := guten.Template{Name: "cli", Renderer: o.renderer, Parts: map[string]string{o.part: src}}
	if err := e.Register(t); err != nil {
		return nil, "", err
	}
	return e, t.Name, nil
}

func partForExt(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt", ".text":
		return guten.PartText
	case ".pdf":
		return "pdf"
	default:
		return guten.PartHTML
	}
}

// runRender renders the requested part and returns it (testable core of render).
func runRender(o opts) (string, error) {
	e, name, err := engineAndTemplate(o)
	if err != nil {
		return "", err
	}
	data, err := loadData(o.data)
	if err != nil {
		return "", err
	}
	return e.RenderPart(name, o.part, data)
}

func cmdRender(args []string) error {
	o, err := parseOpts(args)
	if err != nil {
		return err
	}
	out, err := runRender(o)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

// runExport writes each -o target and returns the paths written (testable core).
func runExport(o opts) ([]string, error) {
	if len(o.outs) == 0 {
		return nil, fmt.Errorf("export requires at least one -o <file>")
	}
	e, name, err := engineAndTemplate(o)
	if err != nil {
		return nil, err
	}
	data, err := loadData(o.data)
	if err != nil {
		return nil, err
	}
	written := make([]string, 0, len(o.outs))
	for _, out := range o.outs {
		part := partForExt(out)
		var payload []byte
		if part == "pdf" {
			htmlStr, rerr := e.RenderPart(name, guten.PartHTML, data)
			if rerr != nil {
				return written, fmt.Errorf("render html for pdf: %w", rerr)
			}
			b, cerr := pdfconv.NewChrome(o.chrome).ToPDF(context.Background(), []byte(htmlStr))
			if cerr != nil {
				return written, cerr
			}
			payload = b
		} else {
			str, rerr := e.RenderPart(name, part, data)
			if rerr != nil {
				return written, rerr
			}
			payload = []byte(str)
		}
		if err := os.WriteFile(out, payload, 0o644); err != nil {
			return written, err
		}
		written = append(written, out)
	}
	return written, nil
}

func cmdExport(args []string) error {
	o, err := parseOpts(args)
	if err != nil {
		return err
	}
	written, err := runExport(o)
	for _, w := range written {
		fmt.Fprintf(os.Stderr, "wrote %s\n", w)
	}
	return err
}

func cmdBuiltins() error {
	e, err := guten.NewWithBuiltins()
	if err != nil {
		return err
	}
	for _, name := range e.Templates() {
		fmt.Println(name)
	}
	return nil
}
