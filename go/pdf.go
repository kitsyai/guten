package guten

import (
	"context"
	"fmt"
)

// PDFConverter converts rendered HTML to PDF bytes.
//
// HTML -> PDF requires a rendering engine (headless Chromium via chromedp, or
// wkhtmltopdf) — a system dependency — so guten keeps it an injectable seam
// rather than a core dependency, preserving the engine's pure/offline nature.
// The consumer provides the converter; see the README for adapter notes.
type PDFConverter interface {
	ToPDF(ctx context.Context, html []byte) ([]byte, error)
}

// RenderToPDF renders the template's html part and converts it to PDF via the
// provided converter.
func (e *Engine) RenderToPDF(ctx context.Context, name string, data map[string]any, converter PDFConverter) ([]byte, error) {
	if converter == nil {
		return nil, fmt.Errorf("guten: no PDF converter provided")
	}
	htmlBody, err := e.RenderPart(name, PartHTML, data)
	if err != nil {
		return nil, err
	}
	return converter.ToPDF(ctx, []byte(htmlBody))
}
