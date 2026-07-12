export type TemplateEntry = {
  name: string;
  kind: string;
  source: string;
  description: string;
};

export type Bundle = {
  name: string;
  renderer: string;
  parts: string[];
  sample: unknown;
};

async function jsonOrThrow(r: Response) {
  if (!r.ok) {
    const text = await r.text();
    throw new Error(text || `HTTP ${r.status}`);
  }
  return r.json();
}

export async function getVersion(): Promise<{ name: string; version: string }> {
  return jsonOrThrow(await fetch("/api/version"));
}

export async function listTemplates(): Promise<TemplateEntry[]> {
  return jsonOrThrow(await fetch("/api/templates"));
}

export async function getTemplate(name: string): Promise<Bundle> {
  return jsonOrThrow(await fetch(`/api/templates/${encodeURIComponent(name)}`));
}

export async function render(
  lib: string,
  data: unknown,
  part = "html",
): Promise<string> {
  const r = await fetch("/api/render", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ lib, data, part }),
  });
  const j = await jsonOrThrow(r);
  return j.output as string;
}

export async function exportPDF(lib: string, data: unknown): Promise<Blob> {
  const r = await fetch("/api/export/pdf", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ lib, data }),
  });
  if (!r.ok) {
    const text = await r.text();
    throw new Error(text || `HTTP ${r.status}`);
  }
  return r.blob();
}
