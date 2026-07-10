import { defineConfig } from "tsup";

const common = {
  platform: "browser",
  sourcemap: true,
  target: "es2022",
} as const;

const reactUMD = {
  ...common,
  entry: ["src/react.ts"],
  format: ["iife"],
  globalName: "GutenReact",
  outExtension: () => ({
    js: ".umd.js",
  }),
  dts: false,
  clean: false,
};

const reactModules = {
  ...common,
  entry: ["src/react.ts"],
  format: ["esm", "cjs"],
  external: ["react", "react-dom"],
  outExtension: (ctx) => ({
    js: ctx.format === "cjs" ? ".cjs" : ".js",
  }),
  dts: true,
  clean: false,
};

export default defineConfig([
  {
    ...common,
    entry: ["src/index.ts"],
    format: ["esm", "cjs", "iife"],
    globalName: "Guten",
    outExtension: (ctx) => ({
      js: ctx.format === "iife" ? ".umd.js" : ctx.format === "cjs" ? ".cjs" : ".js",
    }),
    dts: true,
    clean: true,
  },
  reactUMD,
  reactModules,
]);
