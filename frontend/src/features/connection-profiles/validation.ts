import type { ConnectionProfile, ConnectionProfileDraft } from "./types";

export type ProfileFormValues = Omit<
  ConnectionProfileDraft,
  "port" | "schema"
> & {
  port: string;
  schema: string;
};

export type ProfileFormErrors = Partial<
  Record<keyof ProfileFormValues, string>
>;

/** プロファイル入力値の初期値を作る。 */
export function toProfileFormValues(
  profile: ConnectionProfile | null,
): ProfileFormValues {
  if (profile) {
    return {
      id: profile.id,
      name: profile.name,
      dbType: profile.dbType === "mysql" ? "mysql" : "postgres",
      host: profile.host,
      port: String(profile.port),
      database: profile.database,
      schema: profile.schema,
      user: profile.user,
      password: "",
    };
  }

  return {
    id: "",
    name: "",
    dbType: "postgres",
    host: "",
    port: "5432",
    database: "",
    schema: "public",
    user: "",
    password: "",
  };
}

/** 接続情報用の文字列を検証する。 */
function validateConnectionValue(value: string): boolean {
  return /^[!-~]{1,100}$/.test(value);
}

/** プロファイル入力値を検証する。 */
export function validateProfileForm(
  values: ProfileFormValues,
): ProfileFormErrors {
  const errors: ProfileFormErrors = {};
  const port = Number(values.port);

  if (values.name.trim() === "" || Array.from(values.name).length > 100) {
    errors.name = "表示名は空白以外を含む100文字以内で入力してください。";
  }
  if (!validateConnectionValue(values.host)) {
    errors.host =
      "空白を含まないASCII可視文字を100文字以内で入力してください。";
  }
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    errors.port = "ポートは1から65535の整数で入力してください。";
  }
  if (!validateConnectionValue(values.database)) {
    errors.database =
      "空白を含まないASCII可視文字を100文字以内で入力してください。";
  }
  if (values.dbType === "postgres" && !validateConnectionValue(values.schema)) {
    errors.schema =
      "空白を含まないASCII可視文字を100文字以内で入力してください。";
  }
  if (!validateConnectionValue(values.user)) {
    errors.user =
      "空白を含まないASCII可視文字を100文字以内で入力してください。";
  }

  return errors;
}
