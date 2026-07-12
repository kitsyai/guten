export type TemplateEntry = {
  name: string;
  kind: string;
  source: string;
  description: string;
};

export type Bundle = {
  name: string;
  renderer: string;
  extends?: string;
  parts: string[];
  partSources: Record<string, string>;
  sample: unknown;
  theme?: unknown;
  builtin: boolean;
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

export type BatchResult = {
  blob: Blob;
  total: number;
  written: number;
  errors: number;
};

// runBatch renders `rows` (raw JSONL or CSV text) against `lib` and returns
// the server-built zip (one file per row, named via the `name` filename
// template) plus row-count bookkeeping read off response headers. A row
// failure never fails the request as a whole — see _errors.json inside the
// zip for details — only a request-shaped problem (bad lib, no rows, every
// row failing) throws.
export async function runBatch(
  lib: string,
  rows: string,
  format: "jsonl" | "csv",
  name: string,
): Promise<BatchResult> {
  const r = await fetch("/api/batch", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ lib, rows, format, name }),
  });
  if (!r.ok) {
    const text = await r.text();
    throw new Error(text || `HTTP ${r.status}`);
  }
  const blob = await r.blob();
  return {
    blob,
    total: Number(r.headers.get("X-Batch-Total") ?? 0),
    written: Number(r.headers.get("X-Batch-Written") ?? 0),
    errors: Number(r.headers.get("X-Batch-Errors") ?? 0),
  };
}

export type SaveTemplateRequest = {
  name: string;
  renderer: string;
  parts: Record<string, string>;
  sample: unknown;
  theme?: unknown;
};

// saveTemplate always lands in the user library — builtins are read-only, so
// this is the "duplicate & edit" flow's save step, keyed by request.name
// (which may equal a builtin's name: that only creates a user-tier override
// that shadows the builtin, never modifies it).
export async function saveTemplate(
  req: SaveTemplateRequest,
): Promise<{ name: string; dir: string }> {
  const r = await fetch("/api/templates", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  return jsonOrThrow(r);
}
