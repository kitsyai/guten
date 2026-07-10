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
const bundleSource = readBundle(resolve(process.cwd(), "dist", "index.umd.js"), "utf8");

test("React component can mount Guten output in headless browser", async ({ page }) => {
  if (!reactScript || !reactDOMScript) {
    test.skip(true, "react + react-dom not installed in this workspace");
  }

  await page.setContent(`<!doctype html><html><body><div id="root"></div></body></html>`);
  await page.addScriptTag({ path: reactScript });
  await page.addScriptTag({ path: reactDOMScript });
  await page.addScriptTag({ content: bundleSource });

  const result = await page.evaluate(async () => {
    const React = (window as any).React;
    const ReactDOM = (window as any).ReactDOM;
    const Guten = (window as any).Guten;

    if (!React || !ReactDOM) {
      return { ok: false, reason: "react globals unavailable" };
    }
    if (!Guten || typeof Guten.newWithBuiltins !== "function") {
      return { ok: false, reason: "guten globals unavailable" };
    }

    const output = Guten.newWithBuiltins().render("basic_notification", {
      title: "Welcome",
      name: "Asha",
      body: "Your account is ready.",
      action_url: "https://example.test/start",
    });
    const html = output?.parts?.html || "";

    const App = function () {
      return React.createElement(
        "div",
        null,
        React.createElement("h1", { id: "guten-title" }, "React + Guten"),
        React.createElement("pre", { id: "guten-html" }, html.slice(0, 128)),
        React.createElement("pre", { id: "guten-status" }, html.includes("<!doctype html") ? "mounted" : "missing"),
      );
    };

    const rootNode = document.getElementById("root");
    if (!rootNode) {
      return { ok: false, reason: "root node not found" };
    }

    const node = React.createElement(App);
    if (typeof ReactDOM.createRoot === "function") {
      ReactDOM.createRoot(rootNode).render(node);
    } else if (typeof ReactDOM.render === "function") {
      ReactDOM.render(node, rootNode);
    } else {
      return { ok: false, reason: "no known react renderer" };
    }

    await new Promise((resolve) => setTimeout(resolve, 20));

    const status = document.getElementById("guten-status");
    const title = document.getElementById("guten-title");
    return {
      ok: !!status && status.textContent === "mounted" && !!title && title.textContent === "React + Guten",
      status: status?.textContent ?? null,
      title: title?.textContent ?? null,
      hasHtml: /Hi|Welcome/.test(html),
    };
  });

  expect(result.ok).toBe(true);
  expect(result.hasHtml).toBe(true);
  expect(result.title).toBe("React + Guten");
});
