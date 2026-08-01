import { destroyAppEnvironment } from "../fixtures/app-environment";

/** E2E専用の設定領域を破棄する。 */
export default function teardownAppEnvironment() {
  destroyAppEnvironment();
}
