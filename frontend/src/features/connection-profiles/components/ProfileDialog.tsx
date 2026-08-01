import {
  type FormEvent,
  type ReactNode,
  type RefObject,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

import type { ConnectionProfile, ConnectionProfileDraft } from "../types";
import {
  type ProfileFormErrors,
  type ProfileFormValues,
  toProfileFormValues,
  validateProfileForm,
} from "../validation";

type ProfileDialogProps = {
  profile: ConnectionProfile | null;
  isSaving: boolean;
  returnFocusRef: RefObject<HTMLElement | null>;
  fallbackFocusRef: RefObject<HTMLElement | null>;
  onClose: () => void;
  onSave: (draft: ConnectionProfileDraft) => Promise<void>;
};

export function ProfileDialog({
  profile,
  isSaving,
  returnFocusRef,
  fallbackFocusRef,
  onClose,
  onSave,
}: ProfileDialogProps) {
  const initialValues = useMemo(() => toProfileFormValues(profile), [profile]);
  const [values, setValues] = useState<ProfileFormValues>(initialValues);
  const [errors, setErrors] = useState<ProfileFormErrors>({});
  const nameInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setValues(initialValues);
    setErrors({});
  }, [initialValues]);

  const title = profile ? "接続プロファイルを編集" : "接続プロファイルを追加";

  const updateValue = <Key extends keyof ProfileFormValues>(
    key: Key,
    value: ProfileFormValues[Key],
  ) => {
    setValues((current) => ({ ...current, [key]: value }));
    setErrors((current) => ({ ...current, [key]: undefined }));
  };

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const nextErrors = validateProfileForm(values);
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) {
      return;
    }

    const { schema, ...draftValues } = values;
    await onSave({
      ...draftValues,
      port: Number(values.port),
      ...(values.dbType === "postgres" ? { schema } : {}),
    });
  };

  const close = () => {
    if (!isSaving) {
      onClose();
    }
  };

  return (
    <Dialog open onOpenChange={(open) => !open && close()}>
      <DialogContent
        className="flex max-h-[calc(100vh-2rem)] max-w-2xl flex-col overflow-hidden"
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
          if (isSaving) {
            event.preventDefault();
          }
        }}
        onOpenAutoFocus={(event) => {
          event.preventDefault();
          nameInputRef.current?.focus();
        }}
        onPointerDownOutside={(event) => {
          if (isSaving) {
            event.preventDefault();
          }
        }}
      >
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>
            パスワードは保存後に画面へ表示されません。
          </DialogDescription>
        </DialogHeader>
        <form className="mt-2 flex min-h-0 flex-1 flex-col" onSubmit={submit}>
          <FieldGroup className="grid min-h-0 flex-1 gap-4 overflow-y-auto pb-4 pr-1 sm:grid-cols-2">
            <ProfileField
              disabled={isSaving}
              error={errors.name}
              htmlFor="profile-name"
              label="表示名"
            >
              <Input
                aria-describedby={
                  errors.name ? "profile-name-error" : undefined
                }
                aria-invalid={Boolean(errors.name)}
                disabled={isSaving}
                id="profile-name"
                ref={nameInputRef}
                value={values.name}
                onChange={(event) => updateValue("name", event.target.value)}
              />
            </ProfileField>
            <ProfileField
              disabled={isSaving}
              htmlFor="profile-db-type"
              label="DB種別"
            >
              <Select
                disabled={isSaving}
                value={values.dbType}
                onValueChange={(value) =>
                  updateValue(
                    "dbType",
                    value === "mysql" ? "mysql" : "postgres",
                  )
                }
              >
                <SelectTrigger id="profile-db-type">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="postgres">PostgreSQL</SelectItem>
                    <SelectItem value="mysql">MySQL</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </ProfileField>
            <ProfileField
              disabled={isSaving}
              error={errors.host}
              htmlFor="profile-host"
              label="ホスト"
            >
              <Input
                aria-describedby={
                  errors.host ? "profile-host-error" : undefined
                }
                aria-invalid={Boolean(errors.host)}
                disabled={isSaving}
                id="profile-host"
                maxLength={100}
                value={values.host}
                onChange={(event) => updateValue("host", event.target.value)}
              />
            </ProfileField>
            <ProfileField
              disabled={isSaving}
              error={errors.port}
              htmlFor="profile-port"
              label="ポート"
            >
              <Input
                aria-describedby={
                  errors.port ? "profile-port-error" : undefined
                }
                aria-invalid={Boolean(errors.port)}
                disabled={isSaving}
                id="profile-port"
                inputMode="numeric"
                max="65535"
                min="1"
                type="number"
                value={values.port}
                onChange={(event) => updateValue("port", event.target.value)}
              />
            </ProfileField>
            <ProfileField
              disabled={isSaving}
              error={errors.database}
              htmlFor="profile-database"
              label="データベース名"
            >
              <Input
                aria-describedby={
                  errors.database ? "profile-database-error" : undefined
                }
                aria-invalid={Boolean(errors.database)}
                disabled={isSaving}
                id="profile-database"
                maxLength={100}
                value={values.database}
                onChange={(event) =>
                  updateValue("database", event.target.value)
                }
              />
            </ProfileField>
            {values.dbType === "postgres" ? (
              <ProfileField
                disabled={isSaving}
                error={errors.schema}
                htmlFor="profile-schema"
                label="スキーマ"
              >
                <Input
                  aria-describedby={
                    errors.schema ? "profile-schema-error" : undefined
                  }
                  aria-invalid={Boolean(errors.schema)}
                  disabled={isSaving}
                  id="profile-schema"
                  maxLength={100}
                  value={values.schema}
                  onChange={(event) =>
                    updateValue("schema", event.target.value)
                  }
                />
              </ProfileField>
            ) : null}
            <ProfileField
              disabled={isSaving}
              error={errors.user}
              htmlFor="profile-user"
              label="ユーザー名"
            >
              <Input
                aria-describedby={
                  errors.user ? "profile-user-error" : undefined
                }
                aria-invalid={Boolean(errors.user)}
                disabled={isSaving}
                id="profile-user"
                maxLength={100}
                value={values.user}
                onChange={(event) => updateValue("user", event.target.value)}
              />
            </ProfileField>
            <ProfileField
              disabled={isSaving}
              htmlFor="profile-password"
              label={
                profile
                  ? "パスワード（変更する場合のみ）"
                  : "パスワード（任意）"
              }
            >
              <Input
                autoComplete="new-password"
                disabled={isSaving}
                id="profile-password"
                type="password"
                value={values.password}
                onChange={(event) =>
                  updateValue("password", event.target.value)
                }
              />
            </ProfileField>
          </FieldGroup>
          <DialogFooter className="mt-5 shrink-0 bg-background pt-4">
            <Button
              disabled={isSaving}
              type="button"
              variant="outline"
              onClick={close}
            >
              キャンセル
            </Button>
            <Button disabled={isSaving} type="submit">
              {isSaving ? "保存中…" : "保存"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

type ProfileFieldProps = {
  children: ReactNode;
  disabled: boolean;
  error?: string;
  htmlFor: string;
  label: string;
};

function ProfileField({
  children,
  disabled,
  error,
  htmlFor,
  label,
}: ProfileFieldProps) {
  return (
    <Field data-disabled={disabled} data-invalid={Boolean(error)}>
      <FieldLabel htmlFor={htmlFor}>{label}</FieldLabel>
      {children}
      {error ? <FieldError id={`${htmlFor}-error`}>{error}</FieldError> : null}
    </Field>
  );
}
