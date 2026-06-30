# templates/

Batteries-included template sources live here as the engine grows. In v0 the
starter template (`basic_notification`) is defined in Go in
[`../go/builtins.go`](../go/builtins.go) so it ships compiled with the module;
this directory is the home for file-loaded templates (shared across the Go and
JS runtimes) once the loader and parity corpus land.

Templates are **brand-neutral** and fully parameterised by data — guten carries
no business, brand, or channel knowledge. See [`../spec/template-manifest.md`](../spec/template-manifest.md).
