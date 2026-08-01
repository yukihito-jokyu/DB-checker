import type { RefObject } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

import type { ConnectionProfile } from "../types";

type DeleteProfileDialogProps = {
  isActive: boolean;
  isDeleting: boolean;
  returnFocusRef: RefObject<HTMLElement | null>;
  fallbackFocusRef: RefObject<HTMLElement | null>;
  profile: ConnectionProfile;
  onClose: () => void;
  onDelete: () => Promise<void>;
};

export function DeleteProfileDialog({
  isActive,
  isDeleting,
  returnFocusRef,
  fallbackFocusRef,
  profile,
  onClose,
  onDelete,
}: DeleteProfileDialogProps) {
  const close = () => {
    if (!isDeleting) {
      onClose();
    }
  };

  return (
    <Dialog open onOpenChange={(open) => !open && close()}>
      <DialogContent
        role="alertdialog"
        onCloseAutoFocus={(event) => {
          event.preventDefault();
          const returnFocus = returnFocusRef.current;
          window.requestAnimationFrame(() => {
            const focusTarget = returnFocus?.isConnected
              ? returnFocus
              : fallbackFocusRef.current;
            focusTarget?.focus();
          });
        }}
        onEscapeKeyDown={(event) => {
          if (isDeleting) {
            event.preventDefault();
          }
        }}
        onOverlayClick={close}
        onPointerDownOutside={(event) => {
          if (isDeleting) {
            event.preventDefault();
          }
        }}
      >
        <DialogHeader>
          <DialogTitle>接続プロファイルを削除</DialogTitle>
          <DialogDescription>
            「{profile.name}」を削除します。この操作は元に戻せません。
          </DialogDescription>
          {isActive ? (
            <p className="text-sm text-destructive">
              この接続先は現在使用中です。削除すると接続先は未選択になります。
            </p>
          ) : null}
        </DialogHeader>
        <DialogFooter className="gap-3">
          <Button
            disabled={isDeleting}
            type="button"
            variant="outline"
            onClick={close}
          >
            キャンセル
          </Button>
          <Button
            disabled={isDeleting}
            type="button"
            variant="destructive"
            onClick={() => void onDelete()}
          >
            {isDeleting ? "削除中…" : "削除する"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
