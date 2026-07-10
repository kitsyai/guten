import { describe, expect, it } from "vitest";
import * as React from "react";
import { renderToStaticMarkup } from "react-dom/server";

import { GutenPDF, GutenProvider, GutenView, useGuten } from "../src/react.js";
import { newWithBuiltins } from "../src/index.js";

describe("guten react module", () => {
  it("renders built-in template through component API", () => {
    const html = renderToStaticMarkup(
      React.createElement(GutenView, {
        template: "basic_notification",
        data: {
          title: "Welcome",
          name: "Asha",
          body: "Your account is ready.",
          action_url: "https://example.test/start",
        },
      }),
    );

    expect(html).toContain("<iframe");
    expect(html).toContain("Your account is ready.");
  });

  it("supports useGuten hook on static render", () => {
    const Probe = () =>
      React.createElement("div", null, useGuten({ template: "basic_notification", data: { title: "Hi" } }).output);
    const html = renderToStaticMarkup(React.createElement(Probe));
    expect(html).toContain("Hi");
    expect(html).toContain("!doctype html");
  });

  it("renders via GutenProvider and custom engine", () => {
    const engine = newWithBuiltins();
    const view = React.createElement(
      GutenProvider,
      { engine },
      React.createElement(GutenView, {
        template: "invoice_bold",
        data: { title: "Invoice", name: "Asha", invoice_number: "1001" },
      }),
    );
    const html = renderToStaticMarkup(view);
    expect(html).toContain("<iframe");
    expect(html).toContain("Invoice");
  });

  it("renders PDF component", () => {
    const tree = React.createElement(GutenPDF, {
      template: "invoice_bold",
      data: { title: "Invoice #1001" },
    });
    const html = renderToStaticMarkup(tree);

    expect(html).toContain("<button");
    expect(html).toContain("Download");
    expect(html).toContain("Invoice");
  });
});
