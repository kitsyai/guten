import { describe, expect, it } from "vitest";
import {
  Engine,
  LayoutRenderer,
  RendererLayout,
  renderToPdf,
  type PdfConverter,
} from "../src/index.js";

describe("layout renderer", () => {
  it("renders blocks to HTML", () => {
    const e = new Engine();
    e.registerRenderer(new LayoutRenderer());
    const src = JSON.stringify({
      style: { accent: "#123456" },
      blocks: [
        { type: "heading", text: "{{ title }}" },
        { type: "paragraph", text: "Hi {{ name }}" },
        { type: "button", text: "Open", url: "{{ url }}" },
      ],
    });
    e.register({ name: "promo", renderer: RendererLayout, parts: { html: src } });
    const out = e.renderPart("promo", "html", { title: "Sale", name: "Ada", url: "https://x.test" });
    expect(out).toContain(`<h1 class="guten-heading" style="color:#123456">Sale</h1>`);
    expect(out).toContain(`<p class="guten-paragraph">Hi Ada</p>`);
    expect(out).toContain(`<a class="guten-button" href="https://x.test" style="background:#123456">Open</a>`);
  });
});

describe("pdf seam", () => {
  it("renders html and delegates to the converter", async () => {
    const e = new Engine();
    e.register({ name: "doc", parts: { html: "<h1>{{ t }}</h1>" } });
    let got = "";
    const conv: PdfConverter = {
      toPdf: (html) => {
        got = html;
        return new TextEncoder().encode("%PDF-1.4\n" + html);
      },
    };
    const pdf = await renderToPdf(e, "doc", { t: "Hi" }, conv);
    expect(new TextDecoder().decode(pdf).startsWith("%PDF")).toBe(true);
    expect(got).toContain("<h1>Hi</h1>");
  });

  it("errors without a converter", async () => {
    const e = new Engine();
    e.register({ name: "doc", parts: { html: "x" } });
    await expect(
      renderToPdf(e, "doc", {}, undefined as unknown as PdfConverter),
    ).rejects.toThrow();
  });
});
