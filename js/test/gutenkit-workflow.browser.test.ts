import { describe, expect, it } from "vitest";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { createContext, Script } from "node:vm";

import {
  GUTENKIT_BASELINE_VERSION,
  hydrateTemplate,
  loadGutenkitTemplates,
  loadTemplateManifest,
  loadTemplateSample,
  selectedTemplateNames,
} from "./gutenkit-workflow-helpers.js";

interface RuntimeLike {
  newWithBuiltins: () => {
    register: (template: Record<string, unknown>) => void;
    render: (name: string, data?: Record<string, unknown>) => {
      parts: Record<string, string>;
    };
  };
}

const { templatesDir, manifest } = loadGutenkitTemplates();
const namesToTest = selectedTemplateNames(manifest, 2);
const distPath = join(dirname(fileURLToPath(import.meta.url)), "../dist/index.umd.js");
const distSource = readFileSync(distPath, "utf8");
const nodeRequire = createRequire(distPath);

function loadBrowserRuntime() {
  const context: Record<string, unknown> = {
    ...(globalThis as Record<string, unknown>),
    console,
    require: nodeRequire,
    global: undefined,
    setTimeout,
    clearTimeout,
    TextEncoder,
    TextDecoder,
  };

  context.globalThis = context;
  context.self = context;
  context.window = context;
  context.global = context;

  const script = new Script(distSource);
  const vmContext = createContext(context);
  script.runInContext(vmContext);

  const runtime = (vmContext as { Guten?: RuntimeLike }).Guten;
  if (!runtime) {
    throw new Error("expected global Guten in UMD bundle");
  }
  return runtime;
}

describe(`gutenkit workflow (browser-like via UMD, baseline ${GUTENKIT_BASELINE_VERSION})`, () => {
  it("bootstraps from the UMD bundle into a browser-like global", () => {
    const runtime = loadBrowserRuntime();
    expect(typeof runtime.newWithBuiltins).toBe("function");
  });

  for (const name of namesToTest) {
    it(`renders ${name} template in browser runtime`, () => {
      const manifestPath = join(templatesDir, name, "template.json");
      const templateManifest = loadTemplateManifest(manifestPath);
      const sample = loadTemplateSample(templatesDir, name);
      const resolvedTemplate = hydrateTemplate(templateManifest, templatesDir);
      const runtime = loadBrowserRuntime();
      const engine = runtime.newWithBuiltins();

      engine.register({
        name: resolvedTemplate.name,
        renderer: resolvedTemplate.renderer,
        parts: resolvedTemplate.parts,
      });

      const rendered = engine.render(name, sample as Record<string, unknown>);
      const partNames = Object.keys(resolvedTemplate.parts);
      expect(partNames.length).toBeGreaterThan(0);
      for (const part of partNames) {
        expect(rendered.parts).toHaveProperty(part);
        expect(rendered.parts[part]).toBeTypeOf("string");
      }
    });
  }
});
