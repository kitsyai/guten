// Package library resolves guten template bundles across a Maven/Gradle-style
// precedence: an explicit --lib-dir, then the user's ~/.kitsy/guten/user, then
// the gutenkit cache ~/.kitsy/guten/gutenkit (synced with Pull), then the
// bundled internal templates. A bundle is a directory with a template.json
// manifest, optional theme.json / sample.json, and part source files.
package library

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	guten "github.com/kitsyai/guten/go"
)

// GutenkitRepo is the online tier synced by Pull.
const GutenkitRepo = "https://github.com/kitsyai/gutenkit.git"
const gutenkitPrefix = "@gutenkit/"
const builtinSource = "builtin"

// Bundle is a loaded template bundle.
type Bundle struct {
	Template guten.Template
	Theme    map[string]any
	Sample   map[string]any
}

type manifest struct {
	Name        string            `json:"name"`
	Kind        string            `json:"kind"`
	Renderer    string            `json:"renderer"`
	Extends     string            `json:"extends"`
	Description string            `json:"description"`
	Parts       map[string]string `json:"parts"`
}

// Entry is a summary row for `lib list`.
type Entry struct {
	Name        string
	Kind        string
	Description string
	Source      string
}

type templateRef struct {
	name     string
	gutenkit bool
}

func parseTemplateRef(raw string) (templateRef, error) {
	if strings.TrimSpace(raw) == "" {
		return templateRef{}, fmt.Errorf("template name is empty")
	}
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "@") {
		return templateRef{name: raw}, nil
	}
	if !strings.HasPrefix(raw, gutenkitPrefix) {
		return templateRef{}, fmt.Errorf("unsupported template prefix in %q", raw)
	}
	name := strings.TrimPrefix(raw, gutenkitPrefix)
	if name == "" || strings.Contains(name, "/") {
		return templateRef{}, fmt.Errorf("invalid template ref %q", raw)
	}
	return templateRef{name: name, gutenkit: true}, nil
}

func baseDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".kitsy", "guten")
	}
	return filepath.Join(home, ".kitsy", "guten")
}

// GutenkitDir is the local cache synced from the gutenkit repo.
func GutenkitDir() string { return filepath.Join(baseDir(), "gutenkit") }

func userDir() string { return filepath.Join(baseDir(), "user") }

type root struct {
	fsys fs.FS
	base string // dir within fsys containing template dirs
	src  string
}

// roots returns the search roots in precedence order (highest first).
func roots(libDir string) []root {
	var rs []root
	if libDir != "" {
		rs = append(rs,
			root{os.DirFS(libDir), ".", "lib-dir"},
			root{os.DirFS(libDir), "templates", "lib-dir"},
		)
	}
	rs = append(rs,
		root{os.DirFS(userDir()), "templates", "user"},
		root{os.DirFS(GutenkitDir()), "templates", "gutenkit"},
	)
	return rs
}

// LoadBundle resolves and loads a template bundle by name.
func LoadBundle(name, libDir string) (Bundle, error) {
	ref, err := parseTemplateRef(name)
	if err != nil {
		return Bundle{}, err
	}
	if ref.gutenkit {
		return loadFromGutenkit(ref.name, libDir)
	}
	for _, r := range roots(libDir) {
		dir := path.Join(r.base, ref.name)
		if _, err := fs.Stat(r.fsys, path.Join(dir, "template.json")); err == nil {
			return loadFrom(r.fsys, dir)
		}
	}
	return loadFromBuiltins(ref.name)
}

func loadFromGutenkit(name, libDir string) (Bundle, error) {
	search := []root{
		{os.DirFS(GutenkitDir()), "templates", "gutenkit"},
	}
	if libDir != "" {
		search = append([]root{
			{os.DirFS(libDir), "templates", "lib-dir"},
			{os.DirFS(libDir), ".", "lib-dir"},
		}, search...)
	}
	for _, r := range search {
		dir := path.Join(r.base, name)
		if _, err := fs.Stat(r.fsys, path.Join(dir, "template.json")); err == nil {
			return loadFrom(r.fsys, dir)
		}
	}

	pulled, err := Pull()
	if err != nil {
		return Bundle{}, err
	}
	gutenkitRoot := path.Join("templates", name)
	if _, err := os.Stat(filepath.Join(pulled, gutenkitRoot, "template.json")); err == nil {
		return loadFrom(os.DirFS(pulled), gutenkitRoot)
	}
	return Bundle{}, fmt.Errorf("template %q not found in @gutenkit path", name)
}

func loadFromBuiltins(name string) (Bundle, error) {
	for _, entry := range builtinTemplateCatalog {
		if entry.Name != name {
			continue
		}
		t := guten.Template{
			Name:     entry.Name,
			Renderer: entry.Renderer,
			Extends:  entry.Extends,
			Parts:    copyTemplateParts(entry.Parts),
		}
		b := Bundle{Template: t}
		if err := parseBundledJSON(entry.Theme, &b.Theme, "builtin template theme"); err != nil {
			return Bundle{}, err
		}
		if err := parseBundledJSON(entry.Sample, &b.Sample, "builtin template sample"); err != nil {
			return Bundle{}, err
		}
		return b, nil
	}
	return Bundle{}, fmt.Errorf("template %q not found (searched --lib-dir, ~/.kitsy/guten/user, ~/.kitsy/guten/gutenkit, %s)", name, builtinSource)
}

func parseBundledJSON(raw json.RawMessage, dst *map[string]any, label string) error {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	return nil
}

func copyTemplateParts(parts map[string]string) map[string]string {
	if parts == nil {
		return nil
	}
	out := make(map[string]string, len(parts))
	for k, v := range parts {
		out[k] = v
	}
	return out
}

func loadFrom(fsys fs.FS, dir string) (Bundle, error) {
	raw, err := fs.ReadFile(fsys, path.Join(dir, "template.json"))
	if err != nil {
		return Bundle{}, err
	}
	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return Bundle{}, fmt.Errorf("parse template.json: %w", err)
	}
	parts := make(map[string]string, len(m.Parts))
	for part, val := range m.Parts {
		if strings.HasPrefix(val, "@") {
			b, err := fs.ReadFile(fsys, path.Join(dir, val[1:]))
			if err != nil {
				return Bundle{}, fmt.Errorf("read part %q file %q: %w", part, val[1:], err)
			}
			parts[part] = string(b)
		} else {
			parts[part] = val
		}
	}
	b := Bundle{Template: guten.Template{Name: m.Name, Renderer: m.Renderer, Extends: m.Extends, Parts: parts}}
	if raw, err := fs.ReadFile(fsys, path.Join(dir, "theme.json")); err == nil {
		_ = json.Unmarshal(raw, &b.Theme)
	}
	if raw, err := fs.ReadFile(fsys, path.Join(dir, "sample.json")); err == nil {
		_ = json.Unmarshal(raw, &b.Sample)
	}
	return b, nil
}

// List returns the available templates across all roots (first source wins).
func List(libDir string) []Entry {
	seen := map[string]bool{}
	var out []Entry
	for _, r := range roots(libDir) {
		entries, err := fs.ReadDir(r.fsys, r.base)
		if err != nil {
			continue
		}
		for _, de := range entries {
			if !de.IsDir() || seen[de.Name()] {
				continue
			}
			raw, err := fs.ReadFile(r.fsys, path.Join(r.base, de.Name(), "template.json"))
			if err != nil {
				continue
			}
			var m manifest
			_ = json.Unmarshal(raw, &m)
			seen[de.Name()] = true
			out = append(out, Entry{Name: de.Name(), Kind: m.Kind, Description: m.Description, Source: r.src})
		}
	}
	for _, entry := range builtinTemplateCatalog {
		if seen[entry.Name] {
			continue
		}
		seen[entry.Name] = true
		out = append(out, Entry{Name: entry.Name, Kind: entry.Kind, Description: entry.Description, Source: builtinSource})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Pull syncs the gutenkit repo into GutenkitDir() using git.
func Pull() (string, error) {
	dir := GutenkitDir()
	if _, err := exec.LookPath("git"); err != nil {
		return "", fmt.Errorf("git not found; install git to pull %s", GutenkitRepo)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		if out, err := exec.Command("git", "-C", dir, "pull", "--ff-only").CombinedOutput(); err != nil {
			return "", fmt.Errorf("git pull: %v: %s", err, strings.TrimSpace(string(out)))
		}
		return dir, nil
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return "", err
	}
	if out, err := exec.Command("git", "clone", "--depth", "1", GutenkitRepo, dir).CombinedOutput(); err != nil {
		return "", fmt.Errorf("git clone: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return dir, nil
}
