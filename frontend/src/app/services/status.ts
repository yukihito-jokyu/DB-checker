import { Status } from "@wails/go/wails/AppHandler";
import { unwrapResponse } from "@/lib/appResponse";

export type AppStatus = {
	ready: boolean;
	version: string;
};

/** アプリ全体の疎通状態を Wails binding から取得する。 */
export async function getAppStatus(): Promise<AppStatus> {
	const response = await Status();
	return unwrapResponse(response);
}
