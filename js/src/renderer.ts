import { Liquid, type Template as LiquidTemplate } from "liquidjs";

/** A parsed template ready to render with data. */
export interface Compiled {
  render(data: Record<string, unknown>): string;
}

/**
 * Renderer is a pluggable templating engine. guten ships the Liquid renderer;
 * future renderers (layout/html-css, MJML, …) implement this interface and are
 * added with Engine.registerRenderer — mirrors the Go engine's Renderer.
 */
export interface Renderer {
  name(): string;
  compile(source: string): Compiled;
}

export const RendererLiquid = "liquid";
export const DefaultRenderer = RendererLiquid;

/** LiquidRenderer wraps liquidjs (the JS counterpart of osteele/liquid). */
export class LiquidRenderer implements Renderer {
  private engine: Liquid;

  constructor(engine?: Liquid) {
    this.engine = engine ?? new Liquid();
  }

  name(): string {
    return RendererLiquid;
  }

  compile(source: string): Compiled {
    const tpls: LiquidTemplate[] = this.engine.parse(source);
    const engine = this.engine;
    return {
      render(data: Record<string, unknown>): string {
        return engine.renderSync(tpls, data ?? {}) as string;
      },
    };
  }
}
