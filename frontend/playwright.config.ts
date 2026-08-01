import path from "node:path";
import { defineConfig, devices } from "@playwright/test";

import { createAppEnvironment } from "./e2e/fixtures/app-environment";

const repositoryDir = path.resolve("..");
const appEnvironmentDir = createAppEnvironment();
const goBuildCacheDir = path.join(repositoryDir, ".cache", "go-build");
const baseURL = "http://localhost:34116";

export default defineConfig({
  testDir: "./e2e",
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  failOnFlakyTests: Boolean(process.env.CI),
  outputDir: "test-results",
  reporter: [["html", { outputFolder: "playwright-report", open: "never" }]],
  globalTeardown: "./e2e/setup/environment.teardown.ts",
  use: {
    baseURL,
    trace: "on-first-retry",
    video: "on",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: {
    command: `GOCACHE="${goBuildCacheDir}" DB_CHECKER_E2E_CONFIG_DIR="${appEnvironmentDir}" wails dev -tags e2e -devserver localhost:34116 -noreload -m -skipbindings`,
    cwd: repositoryDir,
    gracefulShutdown: {
      signal: "SIGINT",
      timeout: 5_000,
    },
    url: baseURL,
    reuseExistingServer: false,
    timeout: 120_000,
  },
});
