import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

const bundleSource = readFileSync(resolve(process.cwd(), "dist", "index.umd.js"), "utf8");

test("UMD bundle resolves builtins in a headless browser", async ({ page }) => {
  await page.setContent(`<!doctype html><html><body><div id="status"></div></body></html>`);
  await page.addScriptTag({ content: bundleSource });

  const result = await page.evaluate(() => {
    const guten = (window as any).Guten;
    if (!guten || typeof guten.newWithBuiltins !== "function") {
      return { ok: false, error: "UMD init failed" };
    }

    const rendered = guten.newWithBuiltins().render("invoice_bold");
    const html = rendered?.parts?.html || "";
    return {
      ok:
        rendered?.template === "invoice_bold" &&
        typeof html === "string" &&
        /<!doctype html/i.test(html),
      template: rendered?.template,
      hasHtmlPart: !!html,
      preview: html.slice(0, 80),
    };
  });

  expect(result.ok).toBe(true);
  expect(result.template).toBe("invoice_bold");
  expect(result.hasHtmlPart).toBe(true);
});

