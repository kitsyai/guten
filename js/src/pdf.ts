import { type Engine, PartHTML } from "./guten.js";

/**
 * PdfConverter converts rendered HTML to PDF bytes. HTML -> PDF needs a
 * rendering engine (puppeteer / headless Chromium), a system-level dependency,
 * so guten keeps it an injectable seam rather than a core dependency — the
 * consumer provides the converter (see the README for adapter notes).
 */
export interface PdfConverter {
  toPdf(html: string): Promise<Uint8Array> | Uint8Array;
}

/** renderToPdf renders the template's html part and converts it via converter. */
export async function renderToPdf(
  engine: Engine,
  name: string,
  data: Record<string, unknown>,
  converter: PdfConverter,
): Promise<Uint8Array> {
  if (!converter) throw new Error("guten: no PDF converter provided");
  const html = engine.renderPart(name, PartHTML, data);
  return converter.toPdf(html);
}
