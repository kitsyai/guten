import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./browser-tests",
  testMatch: /.*\.spec\.ts$/,
  use: {
    headless: true,
  },
});
