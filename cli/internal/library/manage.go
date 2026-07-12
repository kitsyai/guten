package library

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	guten "github.com/kitsyai/guten/go"
)

// userTemplateDir returns the directory a user-tier template with this name
// lives (or would live) in: ~/.kitsy/guten/user/templates/<name>.
func userTemplateDir(name string) string {
	return filepath.Join(userDir(), "templates", name)
}

func validTemplateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("template name is required")
	}
	if strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return fmt.Errorf("invalid template name %q", name)
	}
	return nil
}

// NewUserTemplate scaffolds template.json, one file per part, and sample.json
// (plus theme.json when present) into the user tier at
// ~/.kitsy/guten/user/templates/<name>. When from is non-empty it clones an
// existing bundle resolved the normal way (--lib-dir, user, gutenkit,
// builtin) — including builtins, which are otherwise read-only; cloning is
// how their content reaches the (writable) user tier. With no from, it writes
// a minimal starter template. Returns the created directory.
func NewUserTemplate(name, from, libDir string) (string, error) {
	if err := validTemplateName(name); err != nil {
		return "", err
	}
	dir := userTemplateDir(name)
	if _, err := os.Stat(dir); err == nil {
		return "", fmt.Errorf("template %q already exists in the user library (%s)", name, dir)
	}

	m := manifest{Name: name, Parts: map[string]string{}}
	files := map[string]string{}
	var sample map[string]any
	var theme map[string]any

	if strings.TrimSpace(from) != "" {
		b, err := LoadBundle(from, libDir)
		if err != nil {
			return "", fmt.Errorf("load --from %q: %w", from, err)
		}
		m.Renderer = b.Template.Renderer
		m.Extends = b.Template.Extends
		m.Description = fmt.Sprintf("Duplicated from %q for editing.", from)
		for part, src := range b.Template.Parts {
			filename := part + ".liquid"
			m.Parts[part] = "@" + filename
			files[filename] = src
		}
		sample = b.Sample
		theme = b.Theme
	} else {
		m.Kind = "document"
		m.Renderer = guten.RendererLiquid
		m.Parts["html"] = "@html.liquid"
		files["html.liquid"] = "<!doctype html>\n<html>\n  <head><meta charset=\"utf-8\"></head>\n  <body>\n    <p>{{ message | default: \"Hello from " + name + "\" }}</p>\n  </body>\n</html>\n"
		sample = map[string]any{"message": "Hello from " + name}
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := writeJSONFile(filepath.Join(dir, "template.json"), m); err != nil {
		return "", err
	}
	for filename, src := range files {
		if err := os.WriteFile(filepath.Join(dir, filename), []byte(src), 0o644); err != nil {
			return "", err
		}
	}
	if sample == nil {
		sample = map[string]any{}
	}
	if err := writeJSONFile(filepath.Join(dir, "sample.json"), sample); err != nil {
		return "", err
	}
	if len(theme) > 0 {
		if err := writeJSONFile(filepath.Join(dir, "theme.json"), theme); err != nil {
			return "", err
		}
	}
	return dir, nil
}

func writeJSONFile(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// SaveUserTemplate writes a user-tier template from explicit part contents
// and sample/theme data — the save step of a browser "duplicate & edit"
// flow. Unlike NewUserTemplate (which refuses to clobber an existing
// scaffold), this overwrites any existing user-tier template of the same
// name, so repeated edits of the same draft are idempotent saves. It always
// writes to the user tier only; builtins are never touched.
func SaveUserTemplate(name, renderer string, parts map[string]string, sample, theme map[string]any) (string, error) {
	if err := validTemplateName(name); err != nil {
		return "", err
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("template %q has no parts", name)
	}
	dir := userTemplateDir(name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	m := manifest{Name: name, Renderer: renderer, Parts: map[string]string{}}
	for part, src := range parts {
		filename := part + ".liquid"
		m.Parts[part] = "@" + filename
		if err := os.WriteFile(filepath.Join(dir, filename), []byte(src), 0o644); err != nil {
			return "", err
		}
	}
	if err := writeJSONFile(filepath.Join(dir, "template.json"), m); err != nil {
		return "", err
	}
	if sample == nil {
		sample = map[string]any{}
	}
	if err := writeJSONFile(filepath.Join(dir, "sample.json"), sample); err != nil {
		return "", err
	}
	if len(theme) > 0 {
		if err := writeJSONFile(filepath.Join(dir, "theme.json"), theme); err != nil {
			return "", err
		}
	} else {
		_ = os.Remove(filepath.Join(dir, "theme.json")) // re-save without a theme drops a stale one
	}
	return dir, nil
}

// AddUserTemplate copies a template directory (one containing template.json)
// into the user tier at ~/.kitsy/guten/user/templates/<name>, where <name> is
// read from the manifest. It refuses to overwrite an existing user-tier
// template of the same name — remove it first with RemoveUserTemplate.
func AddUserTemplate(srcDir string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(srcDir, "template.json"))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", filepath.Join(srcDir, "template.json"), err)
	}
	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", fmt.Errorf("parse template.json: %w", err)
	}
	if err := validTemplateName(m.Name); err != nil {
		return "", fmt.Errorf("template.json in %s: %w", srcDir, err)
	}
	dstDir := userTemplateDir(m.Name)
	if _, err := os.Stat(dstDir); err == nil {
		return "", fmt.Errorf("template %q already exists in the user library (%s); run `guten lib rm %s` first to replace it", m.Name, dstDir, m.Name)
	}
	if err := copyDir(srcDir, dstDir); err != nil {
		_ = os.RemoveAll(dstDir)
		return "", err
	}
	return dstDir, nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := dst
		if rel != "." {
			target = filepath.Join(dst, rel)
		}
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}

// RemoveUserTemplate deletes a template from the user tier only. It never
// touches the gutenkit cache or the embedded builtins: removing a builtin
// name is refused outright, and removing any other name that isn't present
// in the user tier is refused rather than silently succeeding.
func RemoveUserTemplate(name string) error {
	if err := validTemplateName(name); err != nil {
		return err
	}
	dir := userTemplateDir(name)
	if _, err := os.Stat(dir); err != nil {
		if IsBuiltin(name) {
			return fmt.Errorf("%q is a builtin template and cannot be removed", name)
		}
		return fmt.Errorf("template %q not found in the user library (%s)", name, dir)
	}
	return os.RemoveAll(dir)
}
