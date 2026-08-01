import { expect, test } from "../fixtures/test";

import {
  postgresProfile,
  updatedMySQLProfile,
} from "./connection-profiles.data";
import { ConnectionProfilesPage } from "./connection-profiles.page";

test("利用者は PostgreSQL プロファイルを MySQL として変更できる", async ({
  page,
}) => {
  const connectionProfiles = new ConnectionProfilesPage(page);
  await connectionProfiles.open();

  const createDialog = await connectionProfiles.openCreateDialog();
  await connectionProfiles.selectDatabaseType(
    createDialog,
    postgresProfile.dbType,
  );
  await connectionProfiles.fillProfile(createDialog, postgresProfile);
  await connectionProfiles.save(createDialog);
  await expect(connectionProfiles.profile(postgresProfile.name)).toBeVisible();

  const editDialog = await connectionProfiles.openEditDialog(
    postgresProfile.name,
  );
  await connectionProfiles.selectDatabaseType(
    editDialog,
    updatedMySQLProfile.dbType,
  );
  await expect(editDialog.getByLabel("スキーマ")).toHaveCount(0);
  await connectionProfiles.fillProfile(editDialog, updatedMySQLProfile);
  await connectionProfiles.save(editDialog);

  await expect(
    page.getByText("接続プロファイルを更新しました。"),
  ).toBeVisible();
  const updatedProfile = connectionProfiles.profile(updatedMySQLProfile.name);
  await expect(updatedProfile).toContainText(
    `MySQL · ${updatedMySQLProfile.host}:${updatedMySQLProfile.port} · ${updatedMySQLProfile.database}`,
  );

  await page.reload();
  await expect(
    connectionProfiles.profile(updatedMySQLProfile.name),
  ).toContainText(`ユーザー: ${updatedMySQLProfile.user}`);
  await expect(connectionProfiles.profile(postgresProfile.name)).toHaveCount(0);
});
