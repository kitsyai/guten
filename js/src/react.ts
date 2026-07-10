import * as React from "react";
import * as ReactDOM from "react-dom";

import { Engine, type Rendered } from "./guten.js";
import { newWithBuiltins } from "./builtins.js";

type RenderData = Record<string, unknown>;
type ReactNode = unknown;
type ReactElement = unknown;
type ComponentType<P = Record<string, unknown>> = (props: P) => ReactElement | null;
type CSSProperties = Record<string, string | number | null | undefined>;

export interface GutenProviderProps {
  children: ReactNode;
  engine?: Engine;
  includeBuiltins?: boolean;
}

interface GutenContextValue {
  engine: Engine;
}

const GutenContext = React.createContext(null) as { Provider: unknown } & {
  readonly __brand?: { readonly value: GutenContextValue | null };
};

function createEngine(engine?: Engine, includeBuiltins = true): Engine {
  if (engine) return engine;
  return includeBuiltins ? newWithBuiltins() : new Engine();
}

/** Provide a shared Engine for all child nodes. */
export function GutenProvider({
  children,
  engine,
  includeBuiltins = true,
}: GutenProviderProps): ReactElement {
  const value = React.useMemo(
    () => ({
      engine: createEngine(engine, includeBuiltins),
    }),
    [engine, includeBuiltins],
  );

  return React.createElement(GutenContext.Provider, { value }, children);
}

export interface UseGutenOptions {
  template: string;
  data?: RenderData;
  part?: string;
  engine?: Engine;
  fallback?: string;
}

export interface UseGutenResult {
  rendered: Rendered;
  part: string;
  output: string;
}

/** Resolve and render a template in React render flow. */
export function useGuten({
  template,
  data = {},
  part = "html",
  engine,
  fallback = "",
}: UseGutenOptions): UseGutenResult {
  const context = React.useContext(GutenContext);
  const active = engine ?? context?.engine ?? createEngine();
  const rendered = React.useMemo(() => active.render(template, data), [active, data, template]);
  const output = rendered.parts[part] ?? fallback;

  return {
    rendered,
    part,
    output,
  };
}

export interface GutenViewProps {
  template: string;
  data?: RenderData;
  part?: string;
  engine?: Engine;
  className?: string;
  style?: CSSProperties;
  fallback?: string;
  title?: string;
}

function resolveHtmlOutputStyle(style?: CSSProperties): CSSProperties {
  return {
    width: "100%",
    minHeight: 320,
    border: "0",
    ...style,
  };
}

/** Render a template part in React. */
export function GutenView({
  template,
  data = {},
  part = "html",
  engine,
  className,
  style,
  fallback = "",
  title,
}: GutenViewProps): ReactElement {
  const { part: effectivePart, output } = useGuten({
    template,
    data,
    part,
    engine,
    fallback,
  });

  if (effectivePart === "html") {
    return React.createElement("iframe", {
      className,
      style: resolveHtmlOutputStyle(style),
      title,
      srcDoc: output,
      sandbox: "allow-scripts",
    });
  }

  return React.createElement("pre", { className, style, title }, output);
}

export interface GutenPDFProps extends Omit<GutenViewProps, "part"> {
  filename?: string;
  downloadLabel?: string;
  onDownload?: (rendered: Rendered, options: { filename: string }) => void;
}

function toDownloadFilename(filename: string | undefined): string {
  if (typeof filename === "string" && filename.trim()) {
    return filename;
  }
  return "document.pdf";
}

/** Concrete PDF-oriented view with built-in download action. */
export function GutenPDF({
  filename,
  downloadLabel = "Download",
  onDownload,
  ...props
}: GutenPDFProps): ReactElement {
  const ctx = useGuten({ template: props.template, data: props.data, engine: props.engine, fallback: props.fallback });
  const html = ctx.output;
  const downloadTarget = toDownloadFilename(filename);

  const onClick = React.useCallback(() => {
    if (onDownload) {
      onDownload(ctx.rendered, { filename: downloadTarget });
      return;
    }

    if (typeof window === "undefined" || !window.document) {
      return;
    }

    const blob = new Blob([html], { type: "text/html;charset=utf-8" });
    const url = window.URL.createObjectURL(blob);
    const anchor = window.document.createElement("a");
    anchor.href = url;
    anchor.download = downloadTarget;
    anchor.click();
    window.setTimeout(() => window.URL.revokeObjectURL(url), 0);
  }, [ctx.rendered, downloadTarget, onDownload, html]);

  return React.createElement(
    "div",
    null,
    React.createElement(GutenView, {
      ...props,
      part: "html",
    }),
    React.createElement("button", { type: "button", onClick }, downloadLabel),
  );
}

export interface GutenReactRenderOptions {
  mount: Element | string;
  template: string;
  data?: RenderData;
  part?: string;
  className?: string;
  style?: CSSProperties;
  engine?: Engine;
  View?: ComponentType<GutenViewProps>;
}

type GutenRenderCompatOptions = Omit<GutenReactRenderOptions, "mount" | "template" | "part"> & {
  part?: string;
};

/** Mount a React-based view with optional host-side hooks. */
export const GutenReact = {
  render: (
    mountOrOptions: Element | string | GutenReactRenderOptions,
    format?: string,
    template?: string,
    options?: GutenRenderCompatOptions,
  ): Promise<void> => {
    const normalized: GutenReactRenderOptions =
      typeof mountOrOptions === "object" && mountOrOptions !== null && "template" in mountOrOptions
        ? {
            ...mountOrOptions,
          }
        : {
            mount: mountOrOptions,
            template: template ?? "",
            part: options?.part ?? (format === "text" ? "text" : format === "pdf" ? "html" : "html"),
            data: options?.data,
            className: options?.className,
            style: options?.style,
            engine: options?.engine,
            View: options?.View,
          };

    if (!normalized.template) {
      return Promise.reject(new Error("guten: GutenReact.render requires a template name"));
    }

    if (format === "pdf" && normalized.part === "html") {
      normalized.part = "html";
    }

    const mount =
      typeof normalized.mount === "string"
        ? typeof document !== "undefined"
          ? document.querySelector(normalized.mount)
          : null
        : normalized.mount;

    if (!mount || typeof document === "undefined") {
      return Promise.reject(new Error("guten: GutenReact.render requires a DOM mount target"));
    }

    const view = normalized.View ?? GutenView;
    const element = React.createElement(view, {
      template: normalized.template,
      data: normalized.data,
      part: normalized.part,
      engine: normalized.engine,
      className: normalized.className,
      style: normalized.style,
    });

    return new Promise<void>((resolve) => {
      if (typeof (ReactDOM as { createRoot?: Function }).createRoot === "function") {
        (ReactDOM as { createRoot: (node: Element) => { render: (value: ReactElement) => void } }).createRoot(
          mount,
        ).render(element);
        resolve();
        return;
      }

      if (typeof (ReactDOM as { render?: Function }).render === "function") {
        (ReactDOM as { render: (value: ReactElement, root: Element) => void }).render(element, mount);
      }

      resolve();
    });
  },
};

type GlobalLike = Record<string, unknown>;

const maybeGlobal = typeof globalThis === "object" && globalThis !== null ? (globalThis as GlobalLike) : null;
if (maybeGlobal) {
  maybeGlobal.GutenReact = GutenReact;
  maybeGlobal.GutenView = GutenView;
  maybeGlobal.GutenPDF = GutenPDF;
  maybeGlobal.GutenHTML = GutenView;
  maybeGlobal.GutenProvider = GutenProvider;
  maybeGlobal.useGuten = useGuten;
}

export const GutenHTML = GutenView;
