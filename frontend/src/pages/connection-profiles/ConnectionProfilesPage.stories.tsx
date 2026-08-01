import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, mocked, userEvent, waitFor, within } from "storybook/test";

import { Toaster } from "@/components/ui/sonner";
import {
  activateConnectionProfile,
  deleteConnectionProfile,
  listConnectionProfiles,
  saveConnectionProfile,
} from "@/features/connection-profiles/services/connectionProfiles";
import {
  activatedProfiles,
  defaultProfiles,
  mysqlProfile,
  postgresProfile,
} from "@/features/connection-profiles/storybook";

import { ConnectionProfilesPage } from "./ConnectionProfilesPage";

const meta = {
  title: "Pages/Connection Profiles",
  component: ConnectionProfilesPage,
  decorators: [
    (Story) => (
      <>
        <Story />
        <Toaster />
      </>
    ),
  ],
  parameters: {
    layout: "fullscreen",
  },
  beforeEach: () => {
    mocked(listConnectionProfiles).mockResolvedValue(defaultProfiles);
    mocked(saveConnectionProfile).mockResolvedValue(defaultProfiles);
    mocked(activateConnectionProfile).mockResolvedValue(activatedProfiles);
    mocked(deleteConnectionProfile).mockResolvedValue(defaultProfiles);
  },
} satisfies Meta<typeof ConnectionProfilesPage>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const profileList = await canvas.findByRole("list", {
      name: "接続プロファイル一覧",
    });
    const profileItems = within(profileList).getAllByRole("listitem");

    await expect(
      within(profileItems[0]).getByText("使用中"),
    ).toBeInTheDocument();

    await userEvent.click(
      within(profileItems[1]).getByRole("button", {
        name: "この接続先を使用",
      }),
    );

    await expect(activateConnectionProfile).toHaveBeenCalledWith(
      mysqlProfile.id,
    );
    await waitFor(() =>
      expect(within(profileItems[1]).getByText("使用中")).toBeInTheDocument(),
    );
  },
};

export const Loading: Story = {
  beforeEach: () => {
    mocked(listConnectionProfiles).mockImplementation(
      () => new Promise(() => undefined),
    );
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(canvas.getByRole("status")).toHaveTextContent(
      "接続プロファイルを読み込んでいます…",
    );
  },
};

export const Empty: Story = {
  beforeEach: () => {
    mocked(listConnectionProfiles).mockResolvedValue({ profiles: [] });
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const body = within(canvasElement.ownerDocument.body);
    const addButtons = await canvas.findAllByRole("button", {
      name: "接続プロファイルを追加",
    });

    await userEvent.click(addButtons[0]);

    await expect(
      await body.findByRole("dialog", {
        name: "接続プロファイルを追加",
      }),
    ).toBeInTheDocument();
  },
};

export const LoadFailure: Story = {
  beforeEach: () => {
    mocked(listConnectionProfiles).mockRejectedValue(
      new Error("Storybook: 接続プロファイルの取得失敗"),
    );
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(await canvas.findByRole("alert")).toHaveTextContent(
      "接続プロファイルを読み込めませんでした",
    );
    await expect(canvas.getByRole("button", { name: "再試行" })).toBeEnabled();
  },
};

export const RetrySuccess: Story = {
  beforeEach: () => {
    mocked(listConnectionProfiles)
      .mockRejectedValueOnce(new Error("Storybook: 接続プロファイルの取得失敗"))
      .mockResolvedValue(defaultProfiles);
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await userEvent.click(
      await canvas.findByRole("button", { name: "再試行" }),
    );

    await expect(
      await canvas.findByRole("list", {
        name: "接続プロファイル一覧",
      }),
    ).toBeInTheDocument();
    await expect(listConnectionProfiles).toHaveBeenCalledTimes(2);
  },
};

export const CreateSuccess: Story = {
  beforeEach: () => {
    mocked(listConnectionProfiles).mockResolvedValue({ profiles: [] });
    mocked(saveConnectionProfile).mockResolvedValue({
      profiles: [postgresProfile],
      activeProfileId: postgresProfile.id,
    });
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const body = within(canvasElement.ownerDocument.body);
    const addButtons = await canvas.findAllByRole("button", {
      name: "接続プロファイルを追加",
    });

    await userEvent.click(addButtons[0]);
    const dialog = await body.findByRole("dialog", {
      name: "接続プロファイルを追加",
    });
    const dialogCanvas = within(dialog);

    await userEvent.type(
      dialogCanvas.getByRole("textbox", { name: "表示名" }),
      postgresProfile.name,
    );
    await userEvent.type(
      dialogCanvas.getByRole("textbox", { name: "ホスト" }),
      postgresProfile.host,
    );
    await userEvent.type(
      dialogCanvas.getByRole("textbox", { name: "データベース名" }),
      postgresProfile.database,
    );
    await userEvent.type(
      dialogCanvas.getByRole("textbox", { name: "ユーザー名" }),
      postgresProfile.user,
    );
    await userEvent.click(dialogCanvas.getByRole("button", { name: "保存" }));

    await expect(saveConnectionProfile).toHaveBeenCalledWith({
      id: "",
      name: postgresProfile.name,
      dbType: "postgres",
      host: postgresProfile.host,
      port: 5432,
      database: postgresProfile.database,
      schema: "public",
      user: postgresProfile.user,
      password: "",
    });
    await waitFor(() =>
      expect(
        body.queryByRole("dialog", {
          name: "接続プロファイルを追加",
        }),
      ).not.toBeInTheDocument(),
    );
    const createdList = canvas.getByRole("list", {
      name: "接続プロファイル一覧",
    });
    await expect(
      within(within(createdList).getAllByRole("listitem")[0]).getByText(
        postgresProfile.name,
      ),
    ).toBeInTheDocument();
    await expect(
      await body.findByText("接続プロファイルを追加しました。"),
    ).toBeInTheDocument();
  },
};

export const CreateFailure: Story = {
  beforeEach: () => {
    mocked(listConnectionProfiles).mockResolvedValue({ profiles: [] });
    mocked(saveConnectionProfile).mockRejectedValue(
      new Error("Storybook: 接続プロファイルの保存失敗"),
    );
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const body = within(canvasElement.ownerDocument.body);
    const addButtons = await canvas.findAllByRole("button", {
      name: "接続プロファイルを追加",
    });

    await userEvent.click(addButtons[0]);
    const dialog = await body.findByRole("dialog", {
      name: "接続プロファイルを追加",
    });
    const dialogCanvas = within(dialog);

    await userEvent.type(
      dialogCanvas.getByRole("textbox", { name: "表示名" }),
      "保存失敗確認",
    );
    await userEvent.type(
      dialogCanvas.getByRole("textbox", { name: "ホスト" }),
      "localhost",
    );
    await userEvent.type(
      dialogCanvas.getByRole("textbox", { name: "データベース名" }),
      "app",
    );
    await userEvent.type(
      dialogCanvas.getByRole("textbox", { name: "ユーザー名" }),
      "viewer",
    );
    await userEvent.click(dialogCanvas.getByRole("button", { name: "保存" }));

    await expect(saveConnectionProfile).toHaveBeenCalledOnce();
    await expect(dialog).toBeInTheDocument();
    await expect(
      dialogCanvas.getByRole("textbox", { name: "表示名" }),
    ).toHaveValue("保存失敗確認");
    await expect(
      await body.findByText(
        "操作に失敗しました。時間をおいて再試行してください。",
      ),
    ).toBeInTheDocument();
  },
};

export const EditSuccess: Story = {
  beforeEach: () => {
    mocked(saveConnectionProfile).mockResolvedValue({
      profiles: [{ ...postgresProfile, name: "PostgreSQL 更新済み" }],
      activeProfileId: postgresProfile.id,
    });
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const body = within(canvasElement.ownerDocument.body);
    const list = await canvas.findByRole("list", {
      name: "接続プロファイル一覧",
    });
    const firstItem = within(within(list).getAllByRole("listitem")[0]);

    await userEvent.click(firstItem.getByRole("button", { name: "編集" }));
    const dialog = await body.findByRole("dialog", {
      name: "接続プロファイルを編集",
    });
    const dialogCanvas = within(dialog);
    const nameInput = dialogCanvas.getByRole("textbox", { name: "表示名" });

    await userEvent.clear(nameInput);
    await userEvent.type(nameInput, "PostgreSQL 更新済み");
    await userEvent.click(dialogCanvas.getByRole("button", { name: "保存" }));

    await expect(saveConnectionProfile).toHaveBeenCalledWith({
      id: postgresProfile.id,
      name: "PostgreSQL 更新済み",
      dbType: "postgres",
      host: postgresProfile.host,
      port: postgresProfile.port,
      database: postgresProfile.database,
      schema: postgresProfile.schema,
      user: postgresProfile.user,
      password: "",
    });
    const updatedList = canvas.getByRole("list", {
      name: "接続プロファイル一覧",
    });
    await expect(
      within(within(updatedList).getAllByRole("listitem")[0]).getByText(
        "PostgreSQL 更新済み",
      ),
    ).toBeInTheDocument();
  },
};

export const DeleteSuccess: Story = {
  beforeEach: () => {
    mocked(deleteConnectionProfile).mockResolvedValue({
      profiles: [postgresProfile],
      activeProfileId: postgresProfile.id,
    });
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const body = within(canvasElement.ownerDocument.body);
    const list = await canvas.findByRole("list", {
      name: "接続プロファイル一覧",
    });
    const secondItem = within(within(list).getAllByRole("listitem")[1]);

    await userEvent.click(secondItem.getByRole("button", { name: "削除" }));
    const dialog = await body.findByRole("alertdialog", {
      name: "接続プロファイルを削除",
    });
    await userEvent.click(
      within(dialog).getByRole("button", { name: "削除する" }),
    );

    await expect(deleteConnectionProfile).toHaveBeenCalledWith(mysqlProfile.id);
    await waitFor(() =>
      expect(canvas.queryByText(mysqlProfile.name)).not.toBeInTheDocument(),
    );
    await expect(
      await body.findByText("接続プロファイルを削除しました。"),
    ).toBeInTheDocument();
  },
};

export const DeleteFailure: Story = {
  beforeEach: () => {
    mocked(deleteConnectionProfile).mockRejectedValue(
      new Error("Storybook: 接続プロファイルの削除失敗"),
    );
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const body = within(canvasElement.ownerDocument.body);
    const list = await canvas.findByRole("list", {
      name: "接続プロファイル一覧",
    });
    const secondItem = within(within(list).getAllByRole("listitem")[1]);

    await userEvent.click(secondItem.getByRole("button", { name: "削除" }));
    const dialog = await body.findByRole("alertdialog", {
      name: "接続プロファイルを削除",
    });
    await userEvent.click(
      within(dialog).getByRole("button", { name: "削除する" }),
    );

    await expect(deleteConnectionProfile).toHaveBeenCalledWith(mysqlProfile.id);
    await expect(dialog).toBeInTheDocument();
    await expect(
      await body.findByText(
        "操作に失敗しました。時間をおいて再試行してください。",
      ),
    ).toBeInTheDocument();
  },
};

export const ActivateFailure: Story = {
  beforeEach: () => {
    mocked(activateConnectionProfile).mockRejectedValue(
      new Error("Storybook: 接続プロファイルの切り替え失敗"),
    );
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const body = within(canvasElement.ownerDocument.body);
    const list = await canvas.findByRole("list", {
      name: "接続プロファイル一覧",
    });
    const secondItem = within(within(list).getAllByRole("listitem")[1]);

    await userEvent.click(
      secondItem.getByRole("button", {
        name: "この接続先を使用",
      }),
    );

    await expect(activateConnectionProfile).toHaveBeenCalledWith(
      mysqlProfile.id,
    );
    const activeProfileRegion = canvas.getByRole("region", {
      name: "現在の接続先",
    });
    await expect(
      within(activeProfileRegion).getByText(postgresProfile.name),
    ).toBeInTheDocument();
    await expect(secondItem.queryByText("使用中")).not.toBeInTheDocument();
    await expect(
      await body.findByText(
        "操作に失敗しました。時間をおいて再試行してください。",
      ),
    ).toBeInTheDocument();
  },
};
