package guten

// Builtins returns guten's batteries-included templates from templates/internal.
//
// The shared catalog is generated from templates/internal into
// builtins.generated.go. Keep generation deterministic by running
// scripts/gen-guten-builtins.mjs after catalog changes.
func Builtins() []Template {
	baselines := builtinTemplates()
	out := make([]Template, len(baselines))
	copy(out, baselines)
	return out
}

// NewWithBuiltins returns an Engine pre-loaded with Builtins().
func NewWithBuiltins() (*Engine, error) {
	e := New()
	for _, t := range Builtins() {
		if err := e.Register(t); err != nil {
			return nil, err
		}
	}
	return e, nil
}
