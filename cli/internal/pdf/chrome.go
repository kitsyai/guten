// Package pdf provides the CLI's concrete HTML->PDF converter. guten's core
// keeps PDF as an injectable seam (guten.PDFConverter); this shells out to a
// headless Chromium/Chrome/Edge, so no browser is bundled and html/text output
// needs no browser at all.
package pdf

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Chrome converts HTML to PDF via a headless browser (--headless --print-to-pdf).
// Set Binary explicitly (CLI --chrome or GUTEN_CHROME); otherwise it is
// auto-detected.
type Chrome struct {
	Binary string
}

// NewChrome returns a converter, optionally pinned to a browser binary path.
func NewChrome(binary string) *Chrome { return &Chrome{Binary: binary} }

// ToPDF renders html to a PDF using the browser's print-to-pdf.
func (c *Chrome) ToPDF(ctx context.Context, html []byte) ([]byte, error) {
	bin := c.Binary
	if bin == "" {
		bin = os.Getenv("GUTEN_CHROME")
	}
	if bin == "" {
		bin = DetectBrowser()
	}
	if bin == "" {
		return nil, fmt.Errorf("no Chrome/Edge/Chromium found for PDF output; set --chrome or GUTEN_CHROME (html/text output needs no browser)")
	}

	dir, err := os.MkdirTemp("", "guten-pdf-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	inPath := filepath.Join(dir, "in.html")
	outPath := filepath.Join(dir, "out.pdf")
	if err := os.WriteFile(inPath, html, 0o644); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, bin,
		"--headless=new",
		"--disable-gpu",
		"--no-sandbox",
		"--no-pdf-header-footer",
		"--print-to-pdf="+outPath,
		fileURL(inPath),
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Some browser builds exit non-zero yet still write the PDF; only fail if
		// the output is actually missing.
		if _, statErr := os.Stat(outPath); statErr != nil {
			return nil, fmt.Errorf("browser pdf failed (%s): %w: %s", bin, err, stderr.String())
		}
	}
	b, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("read pdf: %w", err)
	}
	if len(b) == 0 {
		return nil, fmt.Errorf("browser produced an empty pdf")
	}
	return b, nil
}

func fileURL(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	abs = filepath.ToSlash(abs)
	if runtime.GOOS == "windows" {
		return "file:///" + abs
	}
	return "file://" + abs
}

// DetectBrowser returns the first Chromium/Chrome/Edge binary it can find, or "".
func DetectBrowser() string {
	for _, name := range []string{"chrome", "google-chrome", "chromium", "chromium-browser", "msedge", "microsoft-edge"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	var candidates []string
	switch runtime.GOOS {
	case "windows":
		for _, base := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LocalAppData")} {
			if base == "" {
				continue
			}
			candidates = append(candidates,
				filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe"),
				filepath.Join(base, "Microsoft", "Edge", "Application", "msedge.exe"),
			)
		}
	case "darwin":
		candidates = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		}
	default:
		candidates = []string{"/usr/bin/google-chrome", "/usr/bin/chromium", "/usr/bin/chromium-browser", "/usr/bin/microsoft-edge"}
	}
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
