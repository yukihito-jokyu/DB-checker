import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";

import { AppError } from "@/lib/appResponse";

import {
  activateConnectionProfile,
  deleteConnectionProfile,
  listConnectionProfiles,
  saveConnectionProfile,
} from "../services/connectionProfiles";
import type {
  ConnectionProfileDraft,
  ConnectionProfiles,
  LoadState,
  OperationState,
} from "../types";

/** 安全に表示できるエラーメッセージを取得する。 */
function errorMessage(error: unknown): string {
  if (error instanceof AppError) {
    return error.message;
  }

  return "操作に失敗しました。時間をおいて再試行してください。";
}

/** 接続プロファイルの取得と更新状態を管理する。 */
export function useConnectionProfiles() {
  const [connectionProfiles, setConnectionProfiles] =
    useState<ConnectionProfiles>({ profiles: [] });
  const [loadState, setLoadState] = useState<LoadState>("loading");
  const [operation, setOperation] = useState<OperationState>(null);
  const loadGeneration = useRef(0);

  const showToast = useCallback(
    ({ message, type }: { message: string; type: "success" | "error" }) => {
      toast[type](message, { duration: 3000 });
    },
    [],
  );

  const applyProfiles = useCallback((nextProfiles: ConnectionProfiles) => {
    setConnectionProfiles(nextProfiles);
  }, []);

  const loadProfiles = useCallback(async () => {
    const generation = ++loadGeneration.current;
    setLoadState("loading");
    try {
      const result = await listConnectionProfiles();
      if (generation !== loadGeneration.current) {
        return;
      }

      applyProfiles(result);
      setLoadState("success");
    } catch (error) {
      if (generation !== loadGeneration.current) {
        return;
      }

      setLoadState("error");
      showToast({ message: errorMessage(error), type: "error" });
    }
  }, [applyProfiles, showToast]);

  useEffect(() => {
    void loadProfiles();
  }, [loadProfiles]);

  const saveProfile = async (
    draft: ConnectionProfileDraft,
  ): Promise<boolean> => {
    ++loadGeneration.current;
    setOperation("saving");
    try {
      applyProfiles(await saveConnectionProfile(draft));
      setLoadState("success");
      showToast({
        message: draft.id
          ? "接続プロファイルを更新しました。"
          : "接続プロファイルを追加しました。",
        type: "success",
      });
      return true;
    } catch (error) {
      showToast({ message: errorMessage(error), type: "error" });
      return false;
    } finally {
      setOperation(null);
    }
  };

  const activateProfile = async (profileId: string): Promise<void> => {
    ++loadGeneration.current;
    setOperation("activating");
    try {
      applyProfiles(await activateConnectionProfile(profileId));
      showToast({
        message: "使用する接続先を切り替えました。",
        type: "success",
      });
    } catch (error) {
      showToast({ message: errorMessage(error), type: "error" });
    } finally {
      setOperation(null);
    }
  };

  const deleteProfile = async (profileId: string): Promise<boolean> => {
    ++loadGeneration.current;
    setOperation("deleting");
    try {
      applyProfiles(await deleteConnectionProfile(profileId));
      showToast({
        message: "接続プロファイルを削除しました。",
        type: "success",
      });
      return true;
    } catch (error) {
      showToast({ message: errorMessage(error), type: "error" });
      return false;
    } finally {
      setOperation(null);
    }
  };

  return {
    ...connectionProfiles,
    loadState,
    operation,
    loadProfiles,
    saveProfile,
    activateProfile,
    deleteProfile,
  };
}
