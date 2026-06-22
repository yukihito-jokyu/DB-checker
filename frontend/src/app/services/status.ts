import { unwrapResponse } from "@/lib/appResponse";
import { Status } from "@wails/go/wails/AppHandler";

export type AppStatus = {
	ready: boolean;
	version: string;
};

/** アプリ全体の疎通状態を Wails binding から取得する。 */
export async function getAppStatus(): Promise<AppStatus> {
	const response = await Status();
	return unwrapResponse(response);
}
