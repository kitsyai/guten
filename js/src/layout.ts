import type { Compiled, Renderer } from "./renderer.js";

export const RendererLayout = "layout";

interface LayoutBlock {
  type: string;
  text?: string;
  url?: string;
  src?: string;
  alt?: string;
}

interface LayoutSpec {
  style?: Record<string, string>;
  blocks?: LayoutBlock[];
}

const VAR = /\{\{\s*(\w+)\s*\}\}/g;

/** escapeHTML matches Go's html.EscapeString for cross-runtime parity. */
function escapeHTML(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&#34;")
    .replace(/'/g, "&#39;");
}

/**
 * LayoutRenderer renders a JSON block layout (heading/paragraph/button/image
 * with data-bound text) to HTML — the seam for a "Canva-like" designer. It is
 * byte-identical to the Go layout renderer (go/renderer_layout.go).
 */
export class LayoutRenderer implements Renderer {
  name(): string {
    return RendererLayout;
  }

  compile(source: string): Compiled {
    let spec: LayoutSpec;
    try {
      spec = JSON.parse(source) as LayoutSpec;
    } catch (e) {
      throw new Error(`layout: parse: ${(e as Error).message}`);
    }
    return {
      render(data: Record<string, unknown>): string {
        const accent = spec.style?.accent || "#10b981";
        const sub = (s: string | undefined): string =>
          (s ?? "").replace(VAR, (_m, key: string) =>
            key in data ? escapeHTML(String(data[key])) : "",
          );
        let out = '<div class="guten-layout">';
        for (const blk of spec.blocks ?? []) {
          switch (blk.type) {
            case "heading":
              out += `\n<h1 class="guten-heading" style="color:${accent}">${sub(blk.text)}</h1>`;
              break;
            case "paragraph":
              out += `\n<p class="guten-paragraph">${sub(blk.text)}</p>`;
              break;
            case "button":
              out += `\n<a class="guten-button" href="${sub(blk.url)}" style="background:${accent}">${sub(blk.text)}</a>`;
              break;
            case "image":
              out += `\n<img class="guten-image" src="${sub(blk.src)}" alt="${sub(blk.alt)}">`;
              break;
          }
        }
        out += "\n</div>";
        return out;
      },
    };
  }
}
