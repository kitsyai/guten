import { describe, expect, it } from "vitest";
import { Engine } from "../src/index.js";

describe("template inheritance (extends)", () => {
  it("overlays base parts; child wins per-part, inherits the rest", () => {
    const e = new Engine();
    e.register({
      name: "email_base",
      parts: { subject: "{{ subject }}", html: "<main>{{ body }}</main>", text: "{{ body }}" },
    });
    e.register({
      name: "welcome",
      extends: "email_base",
      parts: { html: "<div>Welcome</div><main>{{ body }}</main>" },
    });
    const r = e.render("welcome", { subject: "Hi", body: "Ready" });
    expect(r.parts.subject).toBe("Hi");
    expect(r.parts.text).toBe("Ready");
    expect(r.parts.html).toBe("<div>Welcome</div><main>Ready</main>");
  });

  it("errors extending an unknown base", () => {
    const e = new Engine();
    expect(() => e.register({ name: "x", extends: "nope", parts: { html: "y" } })).toThrow();
  });

  it("fills slots from data, with defaults", () => {
    const e = new Engine();
    e.register({ name: "doc", parts: { html: "<h>{{ slots.header | default: 'Default' }}</h>" } });
    expect(e.renderPart("doc", "html", {})).toBe("<h>Default</h>");
    expect(e.renderPart("doc", "html", { slots: { header: "Custom" } })).toBe("<h>Custom</h>");
  });
});
