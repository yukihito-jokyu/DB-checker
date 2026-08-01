import type { Meta, StoryObj } from "@storybook/react-vite";
import { type ComponentProps, useRef, useState } from "react";
import { expect, fn, userEvent, waitFor, within } from "storybook/test";

import { Button } from "@/components/ui/button";

import { postgresProfile } from "../storybook";
import { ProfileDialog } from "./ProfileDialog";

type ProfileDialogStoryProps = ComponentProps<typeof ProfileDialog>;

function ProfileDialogStory(props: ProfileDialogStoryProps) {
  const [open, setOpen] = useState(true);
  const returnFocusRef = useRef<HTMLButtonElement>(null);

  const close = () => {
    props.onClose();
    setOpen(false);
  };

  return (
    <>
      <Button ref={returnFocusRef} type="button">
        ダイアログを開いたボタン
      </Button>
      {open ? (
        <ProfileDialog
          {...props}
          fallbackFocusRef={returnFocusRef}
          returnFocusRef={returnFocusRef}
          onClose={close}
        />
      ) : null}
    </>
  );
}

const meta = {
  title: "Features/Connection Profiles/Profile Dialog",
  component: ProfileDialog,
  args: {
    fallbackFocusRef: { current: null },
    isSaving: false,
    profile: null,
    returnFocusRef: { current: null },
    onClose: fn(),
    onSave: fn(async () => undefined),
  },
  render: (args) => <ProfileDialogStory {...args} />,
} satisfies Meta<typeof ProfileDialog>;

export default meta;
type Story = StoryObj<typeof meta>;

export const CreatePostgres: Story = {
  play: async ({ args, canvasElement }) => {
    const body = within(canvasElement.ownerDocument.body);
    const dialog = await body.findByRole("dialog", {
      name: "接続プロファイルを追加",
    });
    const dialogCanvas = within(dialog);
    const nameInput = dialogCanvas.getByRole("textbox", { name: "表示名" });

    await waitFor(() => expect(nameInput).toHaveFocus());
    await userEvent.type(nameInput, "分析DB");
    await userEvent.type(
      dialogCanvas.getByRole("textbox", { name: "ホスト" }),
      "analytics.internal",
    );
    await userEvent.type(
      dialogCanvas.getByRole("textbox", { name: "データベース名" }),
      "analytics",
    );
    await userEvent.type(
      dialogCanvas.getByRole("textbox", { name: "ユーザー名" }),
      "analyst",
    );
    await userEvent.click(dialogCanvas.getByRole("button", { name: "保存" }));

    await expect(args.onSave).toHaveBeenCalledWith({
      id: "",
      name: "分析DB",
      dbType: "postgres",
      host: "analytics.internal",
      port: 5432,
      database: "analytics",
      schema: "public",
      user: "analyst",
      password: "",
    });
  },
};

export const CreateMySQL: Story = {
  play: async ({ args, canvasElement }) => {
    const body = within(canvasElement.ownerDocument.body);
    const dialog = await body.findByRole("dialog", {
      name: "接続プロファイルを追加",
    });
    const dialogCanvas = within(dialog);

    await userEvent.click(
      dialogCanvas.getByRole("combobox", { name: "DB種別" }),
    );
    await userEvent.click(await body.findByRole("option", { name: "MySQL" }));
    await expect(
      dialogCanvas.queryByRole("textbox", { name: "スキーマ" }),
    ).not.toBeInTheDocument();

    await userEvent.type(
      dialogCanvas.getByRole("textbox", { name: "表示名" }),
      "MySQL 開発",
    );
    await userEvent.type(
      dialogCanvas.getByRole("textbox", { name: "ホスト" }),
      "localhost",
    );
    await userEvent.clear(
      dialogCanvas.getByRole("spinbutton", { name: "ポート" }),
    );
    await userEvent.type(
      dialogCanvas.getByRole("spinbutton", { name: "ポート" }),
      "3306",
    );
    await userEvent.type(
      dialogCanvas.getByRole("textbox", { name: "データベース名" }),
      "app",
    );
    await userEvent.type(
      dialogCanvas.getByRole("textbox", { name: "ユーザー名" }),
      "root",
    );
    await userEvent.click(dialogCanvas.getByRole("button", { name: "保存" }));

    await expect(args.onSave).toHaveBeenCalledWith({
      id: "",
      name: "MySQL 開発",
      dbType: "mysql",
      host: "localhost",
      port: 3306,
      database: "app",
      user: "root",
      password: "",
    });
  },
};

export const Edit: Story = {
  args: {
    profile: postgresProfile,
  },
  play: async ({ canvasElement }) => {
    const body = within(canvasElement.ownerDocument.body);
    const dialog = await body.findByRole("dialog", {
      name: "接続プロファイルを編集",
    });
    const dialogCanvas = within(dialog);

    await expect(
      dialogCanvas.getByRole("textbox", { name: "表示名" }),
    ).toHaveValue(postgresProfile.name);
    await expect(
      dialogCanvas.getByLabelText("パスワード（変更する場合のみ）"),
    ).toHaveValue("");
  },
};

export const ValidationErrors: Story = {
  play: async ({ args, canvasElement }) => {
    const body = within(canvasElement.ownerDocument.body);
    const dialog = await body.findByRole("dialog", {
      name: "接続プロファイルを追加",
    });
    const dialogCanvas = within(dialog);

    await userEvent.click(dialogCanvas.getByRole("button", { name: "保存" }));

    await expect(dialogCanvas.getAllByRole("alert")).toHaveLength(4);
    await expect(
      dialogCanvas.getByRole("textbox", { name: "表示名" }),
    ).toHaveAttribute("aria-invalid", "true");
    await expect(args.onSave).not.toHaveBeenCalled();
  },
};

export const Saving: Story = {
  args: {
    isSaving: true,
  },
  play: async ({ args, canvasElement }) => {
    const body = within(canvasElement.ownerDocument.body);
    const dialog = await body.findByRole("dialog", {
      name: "接続プロファイルを追加",
    });
    const dialogCanvas = within(dialog);

    await expect(
      dialogCanvas.getByRole("button", { name: "保存中…" }),
    ).toBeDisabled();
    await expect(
      dialogCanvas.getByRole("button", { name: "キャンセル" }),
    ).toBeDisabled();
    await userEvent.keyboard("{Escape}");
    await expect(dialog).toBeInTheDocument();
    await expect(args.onClose).not.toHaveBeenCalled();
  },
};

export const CancelAndRestoreFocus: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement);
    const body = within(canvasElement.ownerDocument.body);
    const dialog = await body.findByRole("dialog", {
      name: "接続プロファイルを追加",
    });

    await userEvent.click(
      within(dialog).getByRole("button", { name: "キャンセル" }),
    );

    await expect(args.onClose).toHaveBeenCalledOnce();
    await waitFor(() =>
      expect(
        canvas.getByRole("button", { name: "ダイアログを開いたボタン" }),
      ).toHaveFocus(),
    );
  },
};
