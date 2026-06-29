import type { wails } from "@wails/go/models";
import { GetConfig } from "@wails/go/wails/AppHandler";
import { unwrapResponse } from "@/lib/appResponse";

export type AppConfig = wails.ConfigData;

/** アプリ設定を Wails binding から取得する。 */
export async function getAppConfig(): Promise<AppConfig> {
	const response = await GetConfig();
	return unwrapResponse(response);
}
