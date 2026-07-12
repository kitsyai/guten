import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  exportPDF,
  getTemplate,
  getVersion,
  listTemplates,
  render,
  type TemplateEntry,
} from "./api";

function download(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

export default function App() {
  const [version, setVersion] = useState("");
  const [templates, setTemplates] = useState<TemplateEntry[]>([]);
  const [filter, setFilter] = useState("");
  const [selected, setSelected] = useState<string>("");
  const [dataText, setDataText] = useState("{}");
  const [html, setHtml] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const dataRef = useRef(dataText);
  dataRef.current = dataText;

  useEffect(() => {
    getVersion().then((v) => setVersion(v.version)).catch(() => {});
    listTemplates()
      .then(setTemplates)
      .catch((e) => setError(String(e)));
  }, []);

  const filtered = useMemo(() => {
    const q = filter.trim().toLowerCase();
    if (!q) return templates;
    return templates.filter(
      (t) =>
        t.name.toLowerCase().includes(q) ||
        t.kind.toLowerCase().includes(q) ||
        t.description.toLowerCase().includes(q),
    );
  }, [templates, filter]);

  const doRender = useCallback(
    async (lib: string, dataStr: string) => {
      setBusy(true);
      setError("");
      try {
        const data = dataStr.trim() ? JSON.parse(dataStr) : {};
        setHtml(await render(lib, data));
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
      } finally {
        setBusy(false);
      }
    },
    [],
  );

  const selectTemplate = useCallback(
    async (name: string) => {
      setSelected(name);
      setError("");
      try {
        const bundle = await getTemplate(name);
        const sample = JSON.stringify(bundle.sample ?? {}, null, 2);
        setDataText(sample);
        await doRender(name, sample);
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
      }
    },
    [doRender],
  );

  const onDownloadPDF = useCallback(async () => {
    if (!selected) return;
    setBusy(true);
    setError("");
    try {
      const data = dataRef.current.trim() ? JSON.parse(dataRef.current) : {};
      download(await exportPDF(selected, data), `${selected}.pdf`);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }, [selected]);

  const onDownloadHTML = useCallback(() => {
    if (!html) return;
    download(new Blob([html], { type: "text/html" }), `${selected || "output"}.html`);
  }, [html, selected]);

  const onEditorKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "Enter" && selected) {
        e.preventDefault();
        void doRender(selected, dataRef.current);
      }
    },
    [selected, doRender],
  );

  return (
    <div className="app">
      <header className="topbar">
        <span className="wordmark">guten</span>
        <span className="version">{version && `v${version}`}</span>
        <span className="spacer" />
        {busy && <span className="busy">working…</span>}
      </header>

      <div className="body">
        <aside className="sidebar">
          <input
            className="filter"
            placeholder="Filter templates…"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
          />
          <ul className="templates">
            {filtered.map((t) => (
              <li key={t.name}>
                <button
                  className={t.name === selected ? "tpl selected" : "tpl"}
                  onClick={() => void selectTemplate(t.name)}
                  title={t.description}
                >
                  <span className="tpl-name">{t.name}</span>
                  <span className={`badge badge-${t.kind}`}>{t.kind}</span>
                </button>
              </li>
            ))}
            {filtered.length === 0 && <li className="empty">no matches</li>}
          </ul>
        </aside>

        <section className="editor">
          <div className="pane-head">
            <span>Data</span>
            <span className="spacer" />
            <button
              className="btn primary"
              disabled={!selected || busy}
              onClick={() => void doRender(selected, dataText)}
              title="Ctrl+Enter"
            >
              Render
            </button>
            <button className="btn" disabled={!selected || busy} onClick={() => void onDownloadPDF()}>
              PDF
            </button>
            <button className="btn" disabled={!html} onClick={onDownloadHTML}>
              HTML
            </button>
          </div>
          <textarea
            className="data"
            spellCheck={false}
            value={dataText}
            onChange={(e) => setDataText(e.target.value)}
            onKeyDown={onEditorKeyDown}
            placeholder={selected ? "{ }" : "Pick a template on the left"}
          />
          {error && <pre className="error">{error}</pre>}
        </section>

        <section className="preview">
          <div className="pane-head">
            <span>Preview</span>
          </div>
          {html ? (
            <iframe className="frame" title="preview" sandbox="" srcDoc={html} />
          ) : (
            <div className="placeholder">
              Select a template to render its sample data.
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
