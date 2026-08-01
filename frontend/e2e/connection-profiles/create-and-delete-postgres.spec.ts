import { expect, test } from "../fixtures/test";

import { postgresProfile } from "./connection-profiles.data";
import { ConnectionProfilesPage } from "./connection-profiles.page";

test("利用者は PostgreSQL プロファイルを追加して使用し、削除できる", async ({
  page,
}) => {
  const connectionProfiles = new ConnectionProfilesPage(page);
  await connectionProfiles.open();

  await expect(
    page.getByRole("heading", { name: "接続プロファイル" }),
  ).toBeVisible();
  await expect(page.getByText("接続プロファイルがありません")).toBeVisible();

  const dialog = await connectionProfiles.openCreateDialog();
  await expect(dialog).toBeVisible();
  await connectionProfiles.selectDatabaseType(dialog, postgresProfile.dbType);
  await expect(dialog.getByLabel("スキーマ")).toBeVisible();
  await connectionProfiles.fillProfile(dialog, postgresProfile);
  await connectionProfiles.save(dialog);

  await expect(
    page.getByText("接続プロファイルを追加しました。"),
  ).toBeVisible();
  const profile = connectionProfiles.profile(postgresProfile.name);
  await expect(profile).toContainText(
    `PostgreSQL · ${postgresProfile.host}:${postgresProfile.port} · ${postgresProfile.database} · ${postgresProfile.schema}`,
  );

  await profile.getByRole("button", { name: "この接続先を使用" }).click();
  await expect(
    page.getByText("使用する接続先を切り替えました。"),
  ).toBeVisible();
  await expect(profile.getByText("使用中")).toBeVisible();

  await connectionProfiles.deleteProfile(postgresProfile.name);
  await expect(
    page.getByText("接続プロファイルを削除しました。"),
  ).toBeVisible();
  await expect(page.getByText("接続プロファイルがありません")).toBeVisible();
  await expect(page.getByText("未選択")).toBeVisible();
});
