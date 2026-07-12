import { useCallback, useEffect, useState } from "react";
import { render, saveTemplate, type Bundle } from "./api";

export default function EditPanel({
  bundle,
  onSaved,
}: {
  bundle: Bundle | null;
  onSaved: (name: string) => void;
}) {
  const [newName, setNewName] = useState("");
  const [parts, setParts] = useState<Record<string, string>>({});
  const [activePart, setActivePart] = useState("");
  const [sampleText, setSampleText] = useState("{}");
  const [html, setHtml] = useState("");
  const [error, setError] = useState("");
  const [status, setStatus] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!bundle) return;
    setNewName(bundle.builtin ? `${bundle.name}-copy` : bundle.name);
    setParts({ ...bundle.partSources });
    const first = Object.keys(bundle.partSources)[0] ?? "";
    setActivePart(first);
    setSampleText(JSON.stringify(bundle.sample ?? {}, null, 2));
    setHtml("");
    setError("");
    setStatus("");
  }, [bundle]);

  const onSave = useCallback(async () => {
    if (!bundle || !newName.trim()) return;
    setBusy(true);
    setError("");
    setStatus("");
    try {
      const sample = sampleText.trim() ? JSON.parse(sampleText) : {};
      await saveTemplate({
        name: newName.trim(),
        renderer: bundle.renderer || "liquid",
        parts,
        sample,
      });
      setStatus(`Saved to the user library as "${newName.trim()}".`);
      onSaved(newName.trim());
      const out = await render(newName.trim(), sample);
      setHtml(out);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }, [bundle, newName, parts, sampleText, onSaved]);

  if (!bundle) {
    return <div className="placeholder">Pick a template on the left to duplicate and edit it.</div>;
  }

  const partNames = Object.keys(parts);

  return (
    <div className="edit">
      <div className="pane-head">
        <span>
          Duplicate {bundle.name}
          {bundle.builtin && <span className="badge readonly">builtin · read-only</span>}
        </span>
      </div>
      <div className="edit-row">
        <label className="edit-label">Save as</label>
        <input
          className="new-name"
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          placeholder="my-template"
        />
      </div>

      <div className="pane-head">
        <span>Parts</span>
        <span className="spacer" />
        {partNames.map((p) => (
          <button
            key={p}
            className={p === activePart ? "tab-chip active" : "tab-chip"}
            onClick={() => setActivePart(p)}
          >
            {p}
          </button>
        ))}
      </div>
      <textarea
        className="data part-editor"
        spellCheck={false}
        value={parts[activePart] ?? ""}
        onChange={(e) => setParts((p) => ({ ...p, [activePart]: e.target.value }))}
      />

      <div className="pane-head">
        <span>Sample data</span>
      </div>
      <textarea
        className="data sample-editor"
        spellCheck={false}
        value={sampleText}
        onChange={(e) => setSampleText(e.target.value)}
      />

      <div className="pane-head">
        <span />
        <span className="spacer" />
        <button className="btn primary" disabled={busy || !newName.trim()} onClick={() => void onSave()}>
          Save to my library &amp; render
        </button>
      </div>
      {status && <div className="status">{status}</div>}
      {error && <pre className="error">{error}</pre>}

      {html ? (
        <iframe className="frame edit-frame" title="edit preview" sandbox="" srcDoc={html} />
      ) : (
        <div className="placeholder">Save to render the edited template.</div>
      )}
    </div>
  );
}
