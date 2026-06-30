package guten

import (
	"encoding/json"
	"fmt"
	"strings"

	cnos "github.com/kitsyai/cnos/packages/go"
)

// Runtime is the minimal cnos read surface guten needs. *cnos.Runtime satisfies
// it; tests pass a fake.
//
// guten reads configuration ONLY through cnos — never from process environment
// variables. The cnos runtime owns layering/superposition (defaults, profiles,
// overrides); guten just reads resolved values from the `guten.*` namespace.
type Runtime interface {
	Value(path string) (any, bool, error)
}

// Config is guten's runtime configuration, resolved from cnos.
type Config struct {
	// DefaultRenderer is the renderer used for templates that don't name one.
	DefaultRenderer string
	// Templates are supplied entirely via config ("templates-as-config").
	Templates []Template
}

// Defaults returns guten's code-level configuration defaults. These apply when
// a cnos value is absent, so guten works batteries-included with zero config.
func Defaults() Config {
	return Config{DefaultRenderer: DefaultRenderer}
}

// Load resolves a Config from the ambient cnos runtime (discovered by cnos-go
// from the projection / working dir). All values come from the `guten.*` cnos
// namespace, layered by cnos.
func Load() (Config, error) {
	rt, err := cnos.Load(cnos.Options{})
	if err != nil {
		return Config{}, fmt.Errorf("guten: load cnos runtime: %w", err)
	}
	return LoadFrom(rt)
}

// LoadFrom resolves a Config from a provided cnos runtime. A consuming service
// that already holds a *cnos.Runtime should pass it here so guten reads from the
// same layered config as the service.
func LoadFrom(rt Runtime) (Config, error) {
	cfg := Defaults()
	if v, ok := cnosString(rt, "guten.default_renderer"); ok {
		cfg.DefaultRenderer = v
	}
	ts, err := cnosTemplates(rt, "guten.templates")
	if err != nil {
		return Config{}, err
	}
	cfg.Templates = ts
	return cfg, nil
}

// NewFromConfig builds an Engine: built-in renderers + batteries-included
// builtins, the configured default renderer, then any config-supplied templates
// (which override builtins of the same name).
func NewFromConfig(cfg Config) (*Engine, error) {
	e, err := NewWithBuiltins()
	if err != nil {
		return nil, err
	}
	if cfg.DefaultRenderer != "" {
		if err := e.SetDefaultRenderer(cfg.DefaultRenderer); err != nil {
			return nil, err
		}
	}
	for _, t := range cfg.Templates {
		if err := e.Register(t); err != nil {
			return nil, err
		}
	}
	return e, nil
}

// NewFromCnos resolves config from the ambient cnos runtime and builds an
// Engine. Equivalent to Load() + NewFromConfig().
func NewFromCnos() (*Engine, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}
	return NewFromConfig(cfg)
}

// NewFromRuntime builds an Engine from a cnos runtime the caller already holds.
// Equivalent to LoadFrom(rt) + NewFromConfig().
func NewFromRuntime(rt Runtime) (*Engine, error) {
	cfg, err := LoadFrom(rt)
	if err != nil {
		return nil, err
	}
	return NewFromConfig(cfg)
}

func cnosString(rt Runtime, path string) (string, bool) {
	if rt == nil {
		return "", false
	}
	v, ok, err := rt.Value(path)
	if err != nil || !ok {
		return "", false
	}
	s, ok := v.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return "", false
	}
	return strings.TrimSpace(s), true
}

// cnosTemplates reads templates-as-config. The value is a JSON string for now
// (per the cnos string-config convention); a structured list is also accepted
// via a JSON round-trip so it keeps working once cnos supports richer configs.
func cnosTemplates(rt Runtime, path string) ([]Template, error) {
	if rt == nil {
		return nil, nil
	}
	v, ok, err := rt.Value(path)
	if err != nil || !ok || v == nil {
		return nil, nil
	}
	var data []byte
	switch typed := v.(type) {
	case string:
		s := strings.TrimSpace(typed)
		if s == "" || s == "[]" {
			return nil, nil
		}
		data = []byte(s)
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("guten: encode cnos value %q: %w", path, err)
		}
		data = encoded
	}
	var ts []Template
	if err := json.Unmarshal(data, &ts); err != nil {
		return nil, fmt.Errorf("guten: parse cnos value %q as templates: %w", path, err)
	}
	return ts, nil
}
