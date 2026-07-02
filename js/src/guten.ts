import {
  DefaultRenderer,
  LiquidRenderer,
  type Compiled,
  type Renderer,
} from "./renderer.js";

/** A named bundle of template sources keyed by output part. */
export interface Template {
  name: string;
  renderer?: string;
  /** Name of an already-registered template to inherit from: this template's
   * parts overlay the base's (per-part). Cross-runtime safe (engine-resolved). */
  extends?: string;
  parts: Record<string, string>;
}

/** The result of rendering a Template with data. */
export interface Rendered {
  template: string;
  parts: Record<string, string>;
}

export const PartSubject = "subject";
export const PartHTML = "html";
export const PartText = "text";

interface StoredTemplate {
  renderer: string;
  parts: Map<string, Compiled>;
}

/**
 * Engine compiles templates once at registration and renders them on demand.
 * It is the JS counterpart of the Go guten.Engine and implements the same
 * template-manifest contract (spec/template-manifest.md).
 */
export class Engine {
  private renderersByName = new Map<string, Renderer>();
  private defaultRenderer = DefaultRenderer;
  private tmpls = new Map<string, StoredTemplate>();
  private raw = new Map<string, Template>();

  constructor() {
    this.registerRenderer(new LiquidRenderer());
  }

  registerRenderer(r: Renderer): void {
    this.renderersByName.set(r.name(), r);
  }

  setDefaultRenderer(name: string): void {
    if (!this.renderersByName.has(name)) {
      throw new Error(`guten: renderer ${JSON.stringify(name)} not registered`);
    }
    this.defaultRenderer = name;
  }

  register(t: Template): void {
    if (!t.name) throw new Error("guten: empty template name");
    // Resolve inheritance: overlay the base template's parts with this one's.
    let renderer = t.renderer;
    let parts: Record<string, string> = t.parts ?? {};
    if (t.extends) {
      const base = this.raw.get(t.extends);
      if (!base) {
        throw new Error(
          `guten: template ${JSON.stringify(t.name)} extends unknown template ${JSON.stringify(t.extends)}`,
        );
      }
      parts = { ...(base.parts ?? {}), ...(t.parts ?? {}) };
      renderer = renderer || base.renderer;
    }
    if (Object.keys(parts).length === 0) {
      throw new Error(`guten: template ${JSON.stringify(t.name)} has no parts`);
    }
    const rendererName = renderer || this.defaultRenderer;
    const r = this.renderersByName.get(rendererName);
    if (!r) {
      throw new Error(
        `guten: template ${JSON.stringify(t.name)} uses unknown renderer ${JSON.stringify(rendererName)}`,
      );
    }
    const compiled = new Map<string, Compiled>();
    for (const [part, src] of Object.entries(parts)) {
      try {
        compiled.set(part, r.compile(src));
      } catch (e) {
        throw new Error(
          `guten: parse template ${JSON.stringify(t.name)} part ${JSON.stringify(part)} (${rendererName}): ${(e as Error).message}`,
        );
      }
    }
    this.tmpls.set(t.name, { renderer: rendererName, parts: compiled });
    this.raw.set(t.name, { name: t.name, renderer: rendererName, parts });
  }

  render(name: string, data: Record<string, unknown> = {}): Rendered {
    const st = this.tmpls.get(name);
    if (!st) throw new Error(`guten: template ${JSON.stringify(name)} not registered`);
    const parts: Record<string, string> = {};
    for (const [part, c] of st.parts) parts[part] = c.render(data);
    return { template: name, parts };
  }

  renderPart(name: string, part: string, data: Record<string, unknown> = {}): string {
    const st = this.tmpls.get(name);
    if (!st) throw new Error(`guten: template ${JSON.stringify(name)} not registered`);
    const c = st.parts.get(part);
    if (!c) {
      throw new Error(
        `guten: template ${JSON.stringify(name)} has no part ${JSON.stringify(part)}`,
      );
    }
    return c.render(data);
  }

  listTemplates(): string[] {
    return [...this.tmpls.keys()].sort();
  }

  listRenderers(): string[] {
    return [...this.renderersByName.keys()].sort();
  }
}
