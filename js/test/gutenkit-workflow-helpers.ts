import { readFileSync } from "node:fs";
import { existsSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

export interface TemplateEntry {
  name: string;
  path: string;
}

export interface TemplateManifest {
  name: string;
  renderer?: string;
  parts: Record<string, string>;
}

interface Registry {
  templates: TemplateEntry[];
}

export const GUTENKIT_BASELINE_VERSION = "0.2.4";

function gutenkitRoot(): string {
  const explicit = process.env.GUTENKIT_TEMPLATE_PATH;
  if (explicit) {
    if (!existsSync(explicit)) {
      throw new Error(`GUTENKIT_TEMPLATE_PATH does not exist: ${explicit}`);
    }
    return explicit;
  }

  const self = dirname(fileURLToPath(import.meta.url));

  const siblingRoot = join(self, "..", "..", "..", "gutenkit");
  if (existsSync(siblingRoot)) {
    return siblingRoot;
  }

  const moduleRoot = join(self, "../node_modules/@kitsy/gutenkit");
  if (!existsSync(moduleRoot)) {
    throw new Error(
      `Cannot resolve @kitsy/gutenkit templates. Expected either:\n` +
        `- installed dependency at node_modules/@kitsy/gutenkit, or\n` +
        `- sibling path ../gutenkit, or\n` +
        "- set GUTENKIT_TEMPLATE_PATH.",
    );
  }
  return moduleRoot;
}

export function loadGutenkitTemplates(): { templatesDir: string; manifest: Registry } {
  const root = gutenkitRoot();
  const packageJson = JSON.parse(readFileSync(join(root, "package.json"), "utf8"));
  if (packageJson.version !== GUTENKIT_BASELINE_VERSION) {
    throw new Error(`expected @kitsy/gutenkit@${GUTENKIT_BASELINE_VERSION}, got ${packageJson.version}`);
  }

  const templatesDir = join(root, "templates");
  const manifestPath = join(templatesDir, "index.json");
  const manifest = JSON.parse(readFileSync(manifestPath, "utf8")) as Registry;
  return { templatesDir, manifest };
}

function readTemplatePart(templatesDir: string, name: string, partSource: string): string {
  if (partSource.startsWith("@")) {
    return readFileSync(join(templatesDir, name, partSource.slice(1)), "utf8");
  }
  return partSource;
}

export function hydrateTemplate(manifest: TemplateManifest, templatesDir: string): {
  name: string;
  renderer?: string;
  parts: Record<string, string>;
} {
  const parts: Record<string, string> = {};
  for (const [part, partSource] of Object.entries(manifest.parts ?? {})) {
    parts[part] = readTemplatePart(templatesDir, manifest.name, partSource);
  }

  return {
    name: manifest.name,
    renderer: manifest.renderer,
    parts,
  };
}

export function loadTemplateManifest(manifestPath: string): TemplateManifest {
  return JSON.parse(readFileSync(manifestPath, "utf8")) as TemplateManifest;
}

export function loadTemplateSample(templatesDir: string, name: string): Record<string, unknown> {
  const path = join(templatesDir, name, "sample.json");
  try {
    return JSON.parse(readFileSync(path, "utf8")) as Record<string, unknown>;
  } catch {
    return {};
  }
}

export function selectedTemplateNames(manifest: Registry, count = 3): string[] {
  const requested = new Set(["invoice", "notification", "otp", "password_reset", "receipt", "welcome"]);
  const preferred = manifest.templates.map((entry) => entry.name).filter((name) => requested.has(name));

  if (preferred.length >= count) {
    return preferred.slice(0, count);
  }

  return manifest.templates.map((entry) => entry.name).slice(0, count);
}
