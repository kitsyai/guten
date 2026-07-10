import { defineConfig } from "tsup";

export default defineConfig({
  entry: ["src/index.ts"],
  platform: "browser",
  format: ["esm", "cjs", "iife"],
  globalName: "Guten",
  outExtension(ctx) {
    if (ctx.format === "iife") return { js: ".umd.js" };
    return { js: ctx.format === "cjs" ? ".cjs" : ".js" };
  },
  dts: true,
  clean: true,
  sourcemap: true,
  target: "es2022",
});
