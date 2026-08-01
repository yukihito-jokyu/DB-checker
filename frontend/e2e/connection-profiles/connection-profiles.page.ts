import type { Locator, Page } from "@playwright/test";

import type {
  ConnectionProfileData,
  DatabaseType,
} from "./connection-profiles.data";

export class ConnectionProfilesPage {
  constructor(private readonly page: Page) {}

  profile(name: string): Locator {
    return this.page.getByRole("listitem").filter({ hasText: name });
  }

  async open() {
    await this.page.goto("/");
  }

  async openCreateDialog(): Promise<Locator> {
    await this.page.getByTestId("add-connection-profile").click();

    return this.page.getByRole("dialog", { name: "接続プロファイルを追加" });
  }

  async openEditDialog(name: string): Promise<Locator> {
    await this.profile(name).getByRole("button", { name: "編集" }).click();

    return this.page.getByRole("dialog", { name: "接続プロファイルを編集" });
  }

  async selectDatabaseType(dialog: Locator, dbType: DatabaseType) {
    await dialog.getByLabel("DB種別").click();
    await this.page
      .getByRole("option", {
        name: dbType === "mysql" ? "MySQL" : "PostgreSQL",
      })
      .click();
  }

  async fillProfile(dialog: Locator, profile: ConnectionProfileData) {
    await dialog.getByLabel("表示名").fill(profile.name);
    await dialog.getByLabel("ホスト").fill(profile.host);
    await dialog.getByLabel("ポート").fill(String(profile.port));
    await dialog.getByLabel("データベース名").fill(profile.database);
    if (profile.dbType === "postgres") {
      await dialog.getByLabel("スキーマ").fill(profile.schema ?? "public");
    }
    await dialog.getByLabel("ユーザー名").fill(profile.user);
  }

  async save(dialog: Locator) {
    await dialog.getByRole("button", { name: "保存" }).click();
  }

  async deleteProfile(name: string) {
    await this.profile(name).getByRole("button", { name: "削除" }).click();
    await this.page
      .getByRole("alertdialog", { name: "接続プロファイルを削除" })
      .getByRole("button", { name: "削除する" })
      .click();
  }
}
