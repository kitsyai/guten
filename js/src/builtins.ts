import type { BuiltinTemplateSpec } from "./builtins.generated.js";
import { builtinTemplates } from "./builtins.generated.js";
import { Engine, type Template } from "./guten.js";

/** builtins returns guten's batteries-included templates. */
export function builtins(): Template[] {
  return builtinTemplates.map((item: BuiltinTemplateSpec) => ({
    name: item.name,
    renderer: item.renderer,
    extends: item.extends,
    parts: item.parts,
  }));
}

/** newWithBuiltins returns an Engine pre-loaded with builtins(). */
export function newWithBuiltins(): Engine {
  const e = new Engine();
  for (const t of builtins()) e.register(t);
  return e;
}
