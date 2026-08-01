import { expect, test } from "../fixtures/test";

import { mysqlProfile } from "./connection-profiles.data";
import { ConnectionProfilesPage } from "./connection-profiles.page";

test("利用者は MySQL プロファイルを追加して削除できる", async ({ page }) => {
  const connectionProfiles = new ConnectionProfilesPage(page);
  await connectionProfiles.open();

  const dialog = await connectionProfiles.openCreateDialog();
  await connectionProfiles.selectDatabaseType(dialog, mysqlProfile.dbType);
  await expect(dialog.getByLabel("スキーマ")).toHaveCount(0);
  await connectionProfiles.fillProfile(dialog, mysqlProfile);
  await connectionProfiles.save(dialog);

  await expect(
    page.getByText("接続プロファイルを追加しました。"),
  ).toBeVisible();
  const profile = connectionProfiles.profile(mysqlProfile.name);
  await expect(profile).toContainText(
    `MySQL · ${mysqlProfile.host}:${mysqlProfile.port} · ${mysqlProfile.database}`,
  );
  await expect(profile).not.toContainText("e2e_schema");

  await connectionProfiles.deleteProfile(mysqlProfile.name);
  await expect(
    page.getByText("接続プロファイルを削除しました。"),
  ).toBeVisible();
  await expect(page.getByText("接続プロファイルがありません")).toBeVisible();
});
