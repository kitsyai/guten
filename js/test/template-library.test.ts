import { describe, expect, test } from "vitest";
import {
  fetchTemplateRegistry,
  templateAssetFallbackUrl,
  templateAssetUrl,
  templateRegistryFallbackUrl,
  templateRegistryUrl,
} from "../src/template-library.js";
import { GUTENKIT_BASELINE_VERSION } from "./gutenkit-workflow-helpers.js";
import packageInfo from "../package.json" with { type: "json" };

describe("template library helpers", () => {
  test("builds jsDelivr registry urls", () => {
    expect(templateRegistryUrl("gutenkit", { version: GUTENKIT_BASELINE_VERSION })).toBe(
      `https://cdn.jsdelivr.net/npm/@kitsy/gutenkit@${GUTENKIT_BASELINE_VERSION}/templates/index.json`,
    );
    expect(templateRegistryFallbackUrl("gutenkit", { version: GUTENKIT_BASELINE_VERSION })).toBe(
      `https://cdn.jsdelivr.net/gh/kitsyai/gutenkit@v${GUTENKIT_BASELINE_VERSION}/templates/index.json`,
    );
  });

  test("builds jsDelivr asset urls", () => {
    expect(templateAssetUrl("gutenkit", "templates/invoice/template.json", { version: GUTENKIT_BASELINE_VERSION })).toBe(
      `https://cdn.jsdelivr.net/npm/@kitsy/gutenkit@${GUTENKIT_BASELINE_VERSION}/templates/invoice/template.json`,
    );
    expect(templateAssetFallbackUrl("gutenkit", "invoice/html.liquid", { version: GUTENKIT_BASELINE_VERSION })).toBe(
      `https://cdn.jsdelivr.net/gh/kitsyai/gutenkit@v${GUTENKIT_BASELINE_VERSION}/templates/invoice/html.liquid`,
    );
  });

  test("falls back to github registry url for gutenkit if npm is missing", async () => {
    const payload = {
      version: 1,
      templates: [{ name: "invoice", kind: "document", path: "templates/invoice", description: "x" }],
    };

    const calls: string[] = [];
    const fakeFetch = (url: string) => {
      calls.push(url);
      if (url.includes("@kitsy/gutenkit")) {
        return Promise.resolve(
          new Response(null, { status: 404, statusText: "not found", headers: { "Content-Type": "application/json" } }),
        );
      }
      return Promise.resolve(new Response(JSON.stringify(payload), { status: 200, headers: { "Content-Type": "application/json" } }));
    };

    const registry = await fetchTemplateRegistry("gutenkit", {
      fetchImpl: fakeFetch,
      version: GUTENKIT_BASELINE_VERSION,
    });
    expect(registry).toEqual(payload);
    expect(calls).toHaveLength(2);
    expect(calls[0]).toBe(
      `https://cdn.jsdelivr.net/npm/@kitsy/gutenkit@${GUTENKIT_BASELINE_VERSION}/templates/index.json`,
    );
    expect(calls[1]).toBe(`https://cdn.jsdelivr.net/gh/kitsyai/gutenkit@v${GUTENKIT_BASELINE_VERSION}/templates/index.json`);
  });

  test("fetches registry from injected fetch", async () => {
    const payload = {
      version: 1,
      templates: [{ name: "invoice", kind: "document", path: "templates/invoice", description: "x" }],
    };
    const fakeFetch = () =>
      Promise.resolve(
        new Response(JSON.stringify(payload), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );

    const registry = await fetchTemplateRegistry("gutenkit", { fetchImpl: fakeFetch });
    expect(registry).toEqual(payload);
  });

  test("exposes UMD CDN entry points", () => {
    expect(packageInfo.unpkg).toBe("./dist/index.umd.js");
    expect(packageInfo.jsdelivr).toBe("./dist/index.umd.js");
  });

  test("builds unpkg URLs for templates", () => {
    expect(templateRegistryUrl("gutenkit", { host: "https://unpkg.com", version: GUTENKIT_BASELINE_VERSION })).toBe(
      `https://unpkg.com/@kitsy/gutenkit@${GUTENKIT_BASELINE_VERSION}/templates/index.json`,
    );
    expect(
      templateAssetUrl("gutenkit", "templates/invoice/template.json", { host: "https://unpkg.com", version: GUTENKIT_BASELINE_VERSION }),
    ).toBe(`https://unpkg.com/@kitsy/gutenkit@${GUTENKIT_BASELINE_VERSION}/templates/invoice/template.json`);
  });
});
