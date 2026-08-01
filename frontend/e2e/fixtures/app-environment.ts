import { mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const configFileName = "config.json";

let configDir: string | undefined;

function defaultConfig() {
  return {
    version: 1,
    connectionProfiles: [],
    activeConnectionProfileId: null,
    flowStates: {},
  };
}

/** E2E専用の設定領域を生成する。 */
export function createAppEnvironment(): string {
  if (configDir) {
    return configDir;
  }

  const existingConfigDir = process.env.DB_CHECKER_E2E_CONFIG_DIR;
  configDir =
    existingConfigDir ??
    join(tmpdir(), `db-checker-e2e-${crypto.randomUUID()}`);
  process.env.DB_CHECKER_E2E_CONFIG_DIR = configDir;
  if (!existingConfigDir) {
    mkdirSync(configDir, { mode: 0o700 });
  }
  resetAppEnvironment();

  return configDir;
}

/** E2E専用の設定を初期化する。 */
export function resetAppEnvironment() {
  if (!configDir) {
    throw new Error("E2E application environment has not been created");
  }

  writeFileSync(
    join(configDir, configFileName),
    `${JSON.stringify(defaultConfig(), null, 2)}\n`,
    { mode: 0o600 },
  );
}

/** E2E専用の設定領域を破棄する。 */
export function destroyAppEnvironment() {
  if (!configDir) {
    return;
  }

  rmSync(configDir, { force: true, recursive: true });
  configDir = undefined;
}
