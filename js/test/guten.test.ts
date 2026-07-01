import { describe, expect, it } from "vitest";
import { Engine, newWithBuiltins } from "../src/index.js";

describe("guten engine", () => {
  it("renders the batteries-included notification", () => {
    const e = newWithBuiltins();
    const r = e.render("basic_notification", {
      title: "Welcome",
      name: "Asha",
      body: "Your account is ready.",
      brand_name: "Acme",
      action_url: "https://example.test/start",
      action_label: "Get started",
    });
    expect(r.parts.subject).toBe("Welcome");
    expect(r.parts.text).toContain("Hi Asha,");
    for (const want of ["Acme", "Welcome", "https://example.test/start", "Get started"]) {
      expect(r.parts.html).toContain(want);
    }
  });

  it("subject falls back to title", () => {
    const e = newWithBuiltins();
    const r = e.render("basic_notification", { title: "Order shipped", body: "On its way." });
    expect(r.parts.subject).toBe("Order shipped");
  });

  it("default filter + conditional", () => {
    const e = new Engine();
    e.register({ name: "greet", parts: { text: `{{ name | default: "there" }}{% if vip %} (VIP){% endif %}` } });
    expect(e.renderPart("greet", "text", { vip: true })).toBe("there (VIP)");
    expect(e.renderPart("greet", "text", { name: "Sam" })).toBe("Sam");
  });

  it("html-escapes untrusted data", () => {
    const e = new Engine();
    e.register({ name: "card", parts: { html: "<p>{{ body | escape }}</p>" } });
    const out = e.renderPart("card", "html", { body: "<b>x</b> & y" });
    expect(out).not.toContain("<b>");
    expect(out).toContain("&lt;b&gt;");
  });

  it("rejects bad templates", () => {
    const e = new Engine();
    expect(() => e.register({ name: "", parts: { text: "x" } })).toThrow();
    expect(() => e.register({ name: "noparts", parts: {} })).toThrow();
  });

  it("errors on unknown template", () => {
    const e = new Engine();
    expect(() => e.render("nope")).toThrow();
  });
});
