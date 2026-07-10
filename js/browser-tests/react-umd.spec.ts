import { expect, test } from "@playwright/test";
import { existsSync, readFileSync as readBundle } from "node:fs";
import { resolve } from "node:path";

const reactScriptCandidates = [
  resolve(process.cwd(), "node_modules", "react", "umd", "react.development.js"),
  resolve(process.cwd(), "node_modules", "react", "cjs", "react.development.js"),
  resolve(process.cwd(), "node_modules", "react", "cjs", "react.production.js"),
];

const reactDOMScriptCandidates = [
  resolve(process.cwd(), "node_modules", "react-dom", "umd", "react-dom.development.js"),
  resolve(process.cwd(), "node_modules", "react-dom", "cjs", "react-dom.development.js"),
];

function firstExisting(paths: string[]): string | null {
  return paths.find((filePath) => existsSync(filePath)) ?? null;
}

const reactScript = firstExisting(reactScriptCandidates);
const reactDOMScript = firstExisting(reactDOMScriptCandidates);
const bundleSource = readBundle(resolve(process.cwd(), "dist", "react.umd.js"), "utf8");

test("UMD react bundle can mount GutenReact views", async ({ page }) => {
  if (!reactScript || !reactDOMScript) {
    test.skip(true, "react + react-dom not installed in this workspace");
  }

  await page.setContent(`<!doctype html><html><body><div id="root"></div></body></html>`);
  await page.addScriptTag({ path: reactScript });
  await page.addScriptTag({ path: reactDOMScript });
  await page.addScriptTag({ content: bundleSource });

  const result = await page.evaluate(async () => {
    const reactRuntime = (window as any).GutenReact;
    const GutenReact = reactRuntime?.render ? reactRuntime : reactRuntime?.GutenReact;
    const root = document.getElementById("root");
    if (!GutenReact?.render || !root) {
      return { ok: false, reason: "render unavailable", hasRuntime: !!reactRuntime };
    }

    await GutenReact.render("#root", "html", "basic_notification", {
      data: {
        title: "Welcome",
        name: "Asha",
        body: "Your account is ready.",
        brand_name: "Acme",
      },
    });
    await new Promise((resolve) => setTimeout(resolve, 50));

    return {
      ok: true,
      hasIframe: root.innerHTML.includes("<iframe"),
      raw: root.innerHTML.slice(0, 120),
    };
  });

  expect(result.ok).toBe(true);
  expect(result.reason).toBe(undefined);
  expect(result.hasIframe).toBe(true);
  expect(result.raw).toContain("iframe");
});
