import { useRef, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from "@/components/ui/empty";
import {
  ConnectionProfileList,
  DeleteProfileDialog,
  ProfileDialog,
} from "@/features/connection-profiles/components";
import { useConnectionProfiles } from "@/features/connection-profiles/hooks/useConnectionProfiles";
import type { ConnectionProfile } from "@/features/connection-profiles/types";

export function ConnectionProfilesPage() {
  const {
    profiles,
    activeProfileId,
    loadState,
    operation,
    loadProfiles,
    saveProfile,
    activateProfile,
    deleteProfile,
  } = useConnectionProfiles();
  const [editingProfile, setEditingProfile] = useState<
    ConnectionProfile | null | undefined
  >(undefined);
  const [deletingProfile, setDeletingProfile] =
    useState<ConnectionProfile | null>(null);
  const addButtonRef = useRef<HTMLButtonElement>(null);
  const dialogTriggerRef = useRef<HTMLElement | null>(null);
  const isDialogOpen = editingProfile !== undefined || deletingProfile !== null;
  const actionsDisabled = operation !== null || isDialogOpen;
  const activeProfile = profiles.find(
    (profile) => profile.id === activeProfileId,
  );

  const openCreateDialog = (trigger: HTMLButtonElement) => {
    dialogTriggerRef.current = trigger;
    setEditingProfile(null);
  };

  const openEditDialog = (
    profile: ConnectionProfile,
    trigger: HTMLButtonElement,
  ) => {
    dialogTriggerRef.current = trigger;
    setEditingProfile(profile);
  };

  const openDeleteDialog = (
    profile: ConnectionProfile,
    trigger: HTMLButtonElement,
  ) => {
    dialogTriggerRef.current = trigger;
    setDeletingProfile(profile);
  };

  const saveAndCloseDialog = async (
    draft: Parameters<typeof saveProfile>[0],
  ) => {
    if (await saveProfile(draft)) {
      setEditingProfile(undefined);
    }
  };

  const deleteAndCloseDialog = async () => {
    if (deletingProfile && (await deleteProfile(deletingProfile.id))) {
      setDeletingProfile(null);
    }
  };

  return (
    <main
      aria-busy={loadState === "loading"}
      className="min-h-screen bg-muted/30 px-4 py-8 text-foreground sm:px-8"
    >
      <section className="mx-auto w-full">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <p className="text-sm font-bold tracking-wide text-muted-foreground">
              DB-checker
            </p>
            <h1 className="mt-1 text-3xl font-bold tracking-tight">
              接続プロファイル
            </h1>
          </div>
          <Button
            data-testid="add-connection-profile"
            disabled={loadState === "loading" || actionsDisabled}
            ref={addButtonRef}
            type="button"
            onClick={(event) => openCreateDialog(event.currentTarget)}
          >
            接続プロファイルを追加
          </Button>
        </div>

        {loadState === "loading" ? (
          <p
            aria-live="polite"
            className="mt-8 rounded-lg border bg-card px-5 py-8 text-base text-muted-foreground"
            role="status"
          >
            接続プロファイルを読み込んでいます…
          </p>
        ) : null}
        {loadState === "error" ? (
          <Alert
            className="mt-8 sm:flex sm:items-center sm:justify-between sm:gap-4"
            variant="destructive"
          >
            <div>
              <AlertTitle>接続プロファイルを読み込めませんでした</AlertTitle>
              <AlertDescription>
                時間をおいてから、もう一度お試しください。
              </AlertDescription>
            </div>
            <Button
              className="mt-4 shrink-0 sm:mt-0"
              disabled={actionsDisabled}
              type="button"
              onClick={() => void loadProfiles()}
            >
              再試行
            </Button>
          </Alert>
        ) : null}
        {loadState === "success" ? (
          <section
            aria-labelledby="active-profile-title"
            aria-live="polite"
            className="mt-8 rounded-xl border bg-card p-5 shadow-soft"
          >
            <h2
              className="text-sm font-bold text-muted-foreground"
              id="active-profile-title"
            >
              現在の接続先
            </h2>
            {activeProfile ? (
              <>
                <p className="mt-2 break-words text-lg font-semibold">
                  {activeProfile.name}
                </p>
                <p className="mt-1 break-words text-base text-muted-foreground">
                  {activeProfile.dbType === "mysql" ? "MySQL" : "PostgreSQL"} ·{" "}
                  {activeProfile.host}:{activeProfile.port} ·{" "}
                  {activeProfile.database}
                  {activeProfile.dbType === "postgres"
                    ? ` · ${activeProfile.schema}`
                    : ""}
                </p>
              </>
            ) : (
              <>
                <p className="mt-2 font-semibold">未選択</p>
                <p className="mt-1 text-base text-muted-foreground">
                  保存済みの接続先から使用するデータベースを選択してください。
                </p>
              </>
            )}
          </section>
        ) : null}
        {loadState === "success" && profiles.length === 0 ? (
          <Empty className="mt-8">
            <EmptyHeader>
              <EmptyTitle>接続プロファイルがありません</EmptyTitle>
              <EmptyDescription>
                接続先を追加して、使用するデータベースを選択してください。
              </EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button
                disabled={actionsDisabled}
                type="button"
                onClick={(event) => openCreateDialog(event.currentTarget)}
              >
                接続プロファイルを追加
              </Button>
            </EmptyContent>
          </Empty>
        ) : null}
        {loadState === "success" && profiles.length > 0 ? (
          <section aria-labelledby="saved-profiles-title" className="mt-8">
            <h2
              className="text-sm font-bold text-muted-foreground"
              id="saved-profiles-title"
            >
              保存済みの接続先
            </h2>
            <ConnectionProfileList
              actionsDisabled={actionsDisabled}
              activeProfileId={activeProfileId}
              operation={operation}
              profiles={profiles}
              onActivate={(profileId) => void activateProfile(profileId)}
              onDelete={openDeleteDialog}
              onEdit={openEditDialog}
            />
          </section>
        ) : null}
      </section>

      {editingProfile !== undefined ? (
        <ProfileDialog
          fallbackFocusRef={addButtonRef}
          isSaving={operation === "saving"}
          profile={editingProfile}
          returnFocusRef={dialogTriggerRef}
          onClose={() => setEditingProfile(undefined)}
          onSave={saveAndCloseDialog}
        />
      ) : null}
      {deletingProfile ? (
        <DeleteProfileDialog
          fallbackFocusRef={addButtonRef}
          isActive={activeProfileId === deletingProfile.id}
          isDeleting={operation === "deleting"}
          profile={deletingProfile}
          returnFocusRef={dialogTriggerRef}
          onClose={() => setDeletingProfile(null)}
          onDelete={deleteAndCloseDialog}
        />
      ) : null}
    </main>
  );
}
