import { useCallback, useMemo, useState } from "react";
import { render, runBatch } from "./api";

function download(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

// parseRowsPreview mirrors the server's row parsing closely enough for a
// client-side preview (row count, and picking a row to preview): JSONL is
// one JSON object per non-blank line; CSV is a header row plus records.
function parseRowsPreview(text: string, format: "jsonl" | "csv"): Record<string, unknown>[] {
  if (format === "csv") {
    const lines = text.split(/\r?\n/).filter((l) => l.trim() !== "");
    if (lines.length < 2) return [];
    const header = lines[0].split(",").map((h) => h.trim());
    return lines.slice(1).map((line) => {
      const cells = line.split(",");
      const row: Record<string, unknown> = {};
      header.forEach((h, i) => (row[h] = cells[i]?.trim() ?? ""));
      return row;
    });
  }
  const rows: Record<string, unknown>[] = [];
  for (const line of text.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed) continue;
    try {
      rows.push(JSON.parse(trimmed));
    } catch {
      // malformed rows are surfaced by the server on batch/preview; skip here
    }
  }
  return rows;
}

export default function BatchPanel({ selected }: { selected: string }) {
  const [rowsText, setRowsText] = useState("");
  const [format, setFormat] = useState<"jsonl" | "csv">("jsonl");
  const [name, setName] = useState("{{ invoice.number }}.pdf");
  const [rowIndex, setRowIndex] = useState(0);
  const [previewHtml, setPreviewHtml] = useState("");
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const rows = useMemo(() => parseRowsPreview(rowsText, format), [rowsText, format]);

  const onFile = useCallback(
    (f: File) => {
      f.text().then((text) => {
        setRowsText(text);
        if (/\.csv$/i.test(f.name)) setFormat("csv");
        else setFormat("jsonl");
      });
    },
    [],
  );

  const onPreviewRow = useCallback(async () => {
    if (!selected || rows.length === 0) return;
    setBusy(true);
    setError("");
    try {
      const row = rows[Math.min(rowIndex, rows.length - 1)] ?? {};
      setPreviewHtml(await render(selected, row));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }, [selected, rows, rowIndex]);

  const onDownloadZip = useCallback(async () => {
    if (!selected) return;
    setBusy(true);
    setError("");
    setStatus("");
    try {
      const res = await runBatch(selected, rowsText, format, name);
      download(res.blob, `${selected}-batch.zip`);
      setStatus(
        res.errors > 0
          ? `${res.written} of ${res.total} row(s) written; ${res.errors} failed — see _errors.json in the zip.`
          : `${res.written} of ${res.total} row(s) written.`,
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }, [selected, rowsText, format, name]);

  return (
    <div className="batch">
      <div className="pane-head">
        <span>Rows</span>
        <span className="spacer" />
        <select
          className="select"
          value={format}
          onChange={(e) => setFormat(e.target.value as "jsonl" | "csv")}
        >
          <option value="jsonl">JSONL</option>
          <option value="csv">CSV</option>
        </select>
        <label className="btn upload">
          Upload
          <input
            type="file"
            accept=".jsonl,.csv,.txt,.json"
            hidden
            onChange={(e) => e.target.files?.[0] && onFile(e.target.files[0])}
          />
        </label>
      </div>
      <textarea
        className="data batch-rows"
        spellCheck={false}
        value={rowsText}
        onChange={(e) => setRowsText(e.target.value)}
        placeholder={
          format === "jsonl"
            ? '{"invoice":{"number":"INV-0001"}}\n{"invoice":{"number":"INV-0002"}}'
            : "invoice.number\nINV-0001\nINV-0002"
        }
      />

      <div className="pane-head">
        <span>Output filename template</span>
      </div>
      <input
        className="filename-tpl"
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder='{{ invoice.number }}.pdf'
      />

      <div className="pane-head">
        <span>Preview row</span>
        <span className="spacer" />
        <input
          className="row-index"
          type="number"
          min={0}
          max={Math.max(0, rows.length - 1)}
          value={rowIndex}
          onChange={(e) => setRowIndex(Number(e.target.value))}
          disabled={rows.length === 0}
        />
        <span className="muted">of {rows.length}</span>
        <button className="btn" disabled={!selected || rows.length === 0 || busy} onClick={() => void onPreviewRow()}>
          Preview
        </button>
        <button
          className="btn primary"
          disabled={!selected || rows.length === 0 || busy}
          onClick={() => void onDownloadZip()}
        >
          Download all as zip
        </button>
      </div>
      {status && <div className="status">{status}</div>}
      {error && <pre className="error">{error}</pre>}

      {previewHtml ? (
        <iframe className="frame batch-frame" title="row preview" sandbox="" srcDoc={previewHtml} />
      ) : (
        <div className="placeholder">Paste or upload rows, then Preview a row.</div>
      )}
    </div>
  );
}
