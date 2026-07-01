import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { Engine, LayoutRenderer, type Template } from "../src/index.js";

const here = dirname(fileURLToPath(import.meta.url));
const corpusDir = join(here, "..", "..", "spec", "corpus");

interface Case {
  name: string;
  template: Template;
  data: Record<string, unknown>;
}

const cases: Case[] = JSON.parse(readFileSync(join(corpusDir, "cases.json"), "utf8"));
const expected: Record<string, Record<string, string>> = JSON.parse(
  readFileSync(join(corpusDir, "expected.json"), "utf8"),
);

// The JS runtime must render every corpus case identically to the Go golden
// (spec/corpus/expected.json), proving cross-runtime Liquid parity.
describe("Liquid parity (Go golden)", () => {
  for (const c of cases) {
    it(`case ${c.name} matches Go`, () => {
      const e = new Engine();
      e.registerRenderer(new LayoutRenderer());
      e.register(c.template);
      const r = e.render(c.template.name, c.data);
      expect(r.parts).toEqual(expected[c.name]);
    });
  }
});
