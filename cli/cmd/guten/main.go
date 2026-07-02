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
      --theme      @file|JSON merged into the data under "theme" (fonts/colors/…)
      --set        key=value override into the data (repeatable), e.g. theme.accent=#0ea5e9
      --css        extra CSS (@file or literal) injected before </head> to override styling (repeatable)
      --header     fill data.slots.header (@file or literal) for inheritance-aware templates
      --footer     fill data.slots.footer (@file or literal)
      --slot       name=<src|@file> fill data.slots.<name> (repeatable)
      --chrome     Chrome/Edge/Chromium path for PDF (else auto-detected; env GUTEN_CHROME)

Examples:
  guten render -t 'Hi {{ name }}' -d '{"name":"Ada"}'
  guten export -t @invoice.html -d @invoice.json -o invoice.html -o invoice.pdf
  guten export -t @invoice.html -d @invoice.json --set theme.accent=#0ea5e9 --css @brand.css -o out.pdf
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
	theme    string
	sets     []string
	css      []string
	header   string
	footer   string
	slots    []string
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
		case "--theme":
			o.theme, err = next()
		case "--set":
			var v string
			if v, err = next(); err == nil {
				o.sets = append(o.sets, v)
			}
		case "--css":
			var v string
			if v, err = next(); err == nil {
				o.css = append(o.css, v)
			}
		case "--header":
			o.header, err = next()
		case "--footer":
			o.footer, err = next()
		case "--slot":
			var v string
			if v, err = next(); err == nil {
				o.slots = append(o.slots, v)
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
	data, err := renderData(o)
	if err != nil {
		return "", err
	}
	out, err := e.RenderPart(name, o.part, data)
	if err != nil {
		return "", err
	}
	if o.part == guten.PartHTML {
		return injectCSS(out, o.css)
	}
	return out, nil
}

// renderData builds the render data: base --data, then --theme merged under
// "theme", then --set key=value overrides applied by dotted path.
func renderData(o opts) (map[string]any, error) {
	data, err := loadData(o.data)
	if err != nil {
		return nil, err
	}
	if o.theme != "" {
		raw, err := loadArg(o.theme)
		if err != nil {
			return nil, err
		}
		var th map[string]any
		if err := json.Unmarshal([]byte(raw), &th); err != nil {
			return nil, fmt.Errorf("parse theme: %w", err)
		}
		base, _ := data["theme"].(map[string]any)
		if base == nil {
			base = map[string]any{}
		}
		for k, v := range th {
			base[k] = v
		}
		data["theme"] = base
	}
	for _, s := range o.sets {
		k, v, ok := strings.Cut(s, "=")
		if !ok {
			return nil, fmt.Errorf("--set expects key=value, got %q", s)
		}
		setPath(data, strings.TrimSpace(k), v)
	}
	// Slots: --header/--footer/--slot name=<src|@file> fill data.slots.* which
	// inheritance-aware templates render as {{ slots.<name> | default: ... }}.
	slots := map[string]any{}
	if o.header != "" {
		v, err := loadArg(o.header)
		if err != nil {
			return nil, err
		}
		slots["header"] = v
	}
	if o.footer != "" {
		v, err := loadArg(o.footer)
		if err != nil {
			return nil, err
		}
		slots["footer"] = v
	}
	for _, s := range o.slots {
		k, v, ok := strings.Cut(s, "=")
		if !ok {
			return nil, fmt.Errorf("--slot expects name=value, got %q", s)
		}
		val, err := loadArg(v)
		if err != nil {
			return nil, err
		}
		slots[strings.TrimSpace(k)] = val
	}
	if len(slots) > 0 {
		base, _ := data["slots"].(map[string]any)
		if base == nil {
			base = map[string]any{}
		}
		for k, v := range slots {
			base[k] = v
		}
		data["slots"] = base
	}
	return data, nil
}

func setPath(m map[string]any, dotted, value string) {
	keys := strings.Split(dotted, ".")
	cur := m
	for i, k := range keys {
		if i == len(keys)-1 {
			cur[k] = value
			return
		}
		next, ok := cur[k].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[k] = next
		}
		cur = next
	}
}

// injectCSS appends CLI-supplied CSS as a <style> block before </head> so it
// wins the cascade over the template's own styles. It is template-agnostic.
func injectCSS(htmlStr string, cssBlocks []string) (string, error) {
	if len(cssBlocks) == 0 {
		return htmlStr, nil
	}
	var b strings.Builder
	b.WriteString("<style>\n")
	for _, c := range cssBlocks {
		s, err := loadArg(c)
		if err != nil {
			return "", err
		}
		b.WriteString(s)
		b.WriteString("\n")
	}
	b.WriteString("</style>")
	inject := b.String()
	if idx := strings.Index(strings.ToLower(htmlStr), "</head>"); idx >= 0 {
		return htmlStr[:idx] + inject + htmlStr[idx:], nil
	}
	return inject + htmlStr, nil
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
	data, err := renderData(o)
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
			htmlStr, rerr = injectCSS(htmlStr, o.css)
			if rerr != nil {
				return written, rerr
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
			if part == guten.PartHTML {
				if str, rerr = injectCSS(str, o.css); rerr != nil {
					return written, rerr
				}
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
