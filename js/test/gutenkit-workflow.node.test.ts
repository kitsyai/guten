import { describe, expect, it } from "vitest";
import { join } from "node:path";

import { newWithBuiltins } from "../src/index.js";
import {
  GUTENKIT_BASELINE_VERSION,
  hydrateTemplate,
  loadGutenkitTemplates,
  loadTemplateManifest,
  loadTemplateSample,
  selectedTemplateNames,
} from "./gutenkit-workflow-helpers.js";

const { templatesDir, manifest } = loadGutenkitTemplates();

const namesToTest = selectedTemplateNames(manifest, 3);

function describeName(name: string) {
  return `node runtime renders ${name} using @kitsy/gutenkit@${GUTENKIT_BASELINE_VERSION}`;
}

describe(`gutenkit workflow (node, baseline ${GUTENKIT_BASELINE_VERSION})`, () => {
  it(`renders built-in invoice_bold in node runtime`, () => {
    const e = newWithBuiltins();
    const rendered = e.render("invoice_bold");
    expect(rendered.template).toBe("invoice_bold");
    expect(rendered.parts).toBeTruthy();
    expect(rendered.parts).toHaveProperty("html");
    expect(rendered.parts.html).toContain("<html");
  });

  for (const name of namesToTest) {
    it(describeName(name), () => {
      const manifestPath = join(templatesDir, name, "template.json");
      const templateManifest = loadTemplateManifest(manifestPath);
      const sample = loadTemplateSample(templatesDir, name);
      const resolvedTemplate = hydrateTemplate(templateManifest, templatesDir);

      const e = newWithBuiltins();
      e.register({
        name: resolvedTemplate.name,
        renderer: resolvedTemplate.renderer,
        parts: resolvedTemplate.parts,
      });

      const rendered = e.render(name, sample as Record<string, unknown>);
      const renderedParts = Object.keys(resolvedTemplate.parts);
      expect(rendered.parts).toBeTruthy();
      for (const part of renderedParts) {
        expect(rendered.parts).toHaveProperty(part);
        expect(rendered.parts[part]).toEqual(expect.stringContaining(""));
      }
    });
  }
});
