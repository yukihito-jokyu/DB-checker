import type { wails } from "@wails/go/models";
import {
  ActivateConnectionProfile,
  DeleteConnectionProfile,
  ListConnectionProfiles,
  SaveConnectionProfile,
} from "@wails/go/wails/AppHandler";

import { unwrapResponse } from "@/lib/appResponse";

import type { ConnectionProfileDraft, ConnectionProfiles } from "../types";

/** 接続プロファイルのレスポンスを画面用の形へ変換する。 */
function toConnectionProfiles(
  response: wails.ConnectionProfilesResponse,
): ConnectionProfiles {
  return {
    profiles: response.profiles ?? [],
    activeProfileId: response.activeConnectionProfileId,
  };
}

/** 接続プロファイル一覧を取得する。 */
export async function listConnectionProfiles(): Promise<ConnectionProfiles> {
  return toConnectionProfiles(unwrapResponse(await ListConnectionProfiles()));
}

/** 接続プロファイルを保存する。 */
export async function saveConnectionProfile(
  draft: ConnectionProfileDraft,
): Promise<ConnectionProfiles> {
  return toConnectionProfiles(
    unwrapResponse(await SaveConnectionProfile(draft)),
  );
}

/** 使用する接続プロファイルを切り替える。 */
export async function activateConnectionProfile(
  profileId: string,
): Promise<ConnectionProfiles> {
  return toConnectionProfiles(
    unwrapResponse(await ActivateConnectionProfile(profileId)),
  );
}

/** 接続プロファイルを削除する。 */
export async function deleteConnectionProfile(
  profileId: string,
): Promise<ConnectionProfiles> {
  return toConnectionProfiles(
    unwrapResponse(await DeleteConnectionProfile(profileId)),
  );
}
