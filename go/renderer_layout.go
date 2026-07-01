package guten

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
)

// RendererLayout is the id of the built-in layout renderer.
const RendererLayout = "layout"

// A layout template source is JSON describing a block-based document — the seam
// for a "Canva-like" designer where users compose blocks (heading, paragraph,
// button, image) with data-bound text. It renders to HTML; blocks and styles are
// data/config, so guten still carries no brand or business knowledge.
type layoutBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	URL  string `json:"url,omitempty"`
	Src  string `json:"src,omitempty"`
	Alt  string `json:"alt,omitempty"`
}

type layoutSpec struct {
	Style  map[string]string `json:"style,omitempty"`
	Blocks []layoutBlock     `json:"blocks"`
}

type layoutRenderer struct{}

// NewLayoutRenderer returns the built-in layout renderer.
func NewLayoutRenderer() Renderer { return layoutRenderer{} }

func (layoutRenderer) Name() string { return RendererLayout }

func (layoutRenderer) Compile(source string) (CompiledTemplate, error) {
	var spec layoutSpec
	if err := json.Unmarshal([]byte(source), &spec); err != nil {
		return nil, fmt.Errorf("layout: parse: %w", err)
	}
	return layoutCompiled{spec: spec}, nil
}

type layoutCompiled struct{ spec layoutSpec }

var layoutVar = regexp.MustCompile(`\{\{\s*(\w+)\s*\}\}`)

func (c layoutCompiled) Render(data map[string]any) (string, error) {
	accent := c.spec.Style["accent"]
	if accent == "" {
		accent = "#10b981"
	}
	sub := func(s string) string {
		return layoutVar.ReplaceAllStringFunc(s, func(m string) string {
			key := layoutVar.FindStringSubmatch(m)[1]
			if v, ok := data[key]; ok {
				return html.EscapeString(fmt.Sprint(v))
			}
			return ""
		})
	}
	var b strings.Builder
	b.WriteString(`<div class="guten-layout">`)
	for _, blk := range c.spec.Blocks {
		switch blk.Type {
		case "heading":
			fmt.Fprintf(&b, "\n<h1 class=\"guten-heading\" style=\"color:%s\">%s</h1>", accent, sub(blk.Text))
		case "paragraph":
			fmt.Fprintf(&b, "\n<p class=\"guten-paragraph\">%s</p>", sub(blk.Text))
		case "button":
			fmt.Fprintf(&b, "\n<a class=\"guten-button\" href=\"%s\" style=\"background:%s\">%s</a>", sub(blk.URL), accent, sub(blk.Text))
		case "image":
			fmt.Fprintf(&b, "\n<img class=\"guten-image\" src=\"%s\" alt=\"%s\">", sub(blk.Src), sub(blk.Alt))
		}
	}
	b.WriteString("\n</div>")
	return b.String(), nil
}
