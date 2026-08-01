import type { Meta, StoryObj } from "@storybook/react-vite";
import { type ComponentProps, useRef, useState } from "react";
import { expect, fn, userEvent, waitFor, within } from "storybook/test";

import { Button } from "@/components/ui/button";

import { longProfile, postgresProfile } from "../storybook";
import { DeleteProfileDialog } from "./DeleteProfileDialog";

type DeleteDialogStoryProps = ComponentProps<typeof DeleteProfileDialog>;

function DeleteDialogStory(props: DeleteDialogStoryProps) {
  const [open, setOpen] = useState(true);
  const returnFocusRef = useRef<HTMLButtonElement>(null);

  const close = () => {
    props.onClose();
    setOpen(false);
  };

  return (
    <>
      <Button ref={returnFocusRef} type="button">
        削除を開いたボタン
      </Button>
      {open ? (
        <DeleteProfileDialog
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
  title: "Features/Connection Profiles/Delete Profile Dialog",
  component: DeleteProfileDialog,
  args: {
    fallbackFocusRef: { current: null },
    isActive: false,
    isDeleting: false,
    profile: postgresProfile,
    returnFocusRef: { current: null },
    onClose: fn(),
    onDelete: fn(async () => undefined),
  },
  render: (args) => <DeleteDialogStory {...args} />,
} satisfies Meta<typeof DeleteProfileDialog>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  play: async ({ args, canvasElement }) => {
    const body = within(canvasElement.ownerDocument.body);
    const dialog = await body.findByRole("alertdialog", {
      name: "接続プロファイルを削除",
    });

    await userEvent.click(
      within(dialog).getByRole("button", { name: "削除する" }),
    );

    await expect(args.onDelete).toHaveBeenCalledOnce();
  },
};

export const ActiveProfile: Story = {
  args: {
    isActive: true,
  },
  play: async ({ canvasElement }) => {
    const body = within(canvasElement.ownerDocument.body);
    const dialog = await body.findByRole("alertdialog", {
      name: "接続プロファイルを削除",
    });

    await expect(dialog).toHaveTextContent(
      "この接続先は現在使用中です。削除すると接続先は未選択になります。",
    );
  },
};

export const Deleting: Story = {
  args: {
    isDeleting: true,
  },
  play: async ({ args, canvasElement }) => {
    const body = within(canvasElement.ownerDocument.body);
    const dialog = await body.findByRole("alertdialog", {
      name: "接続プロファイルを削除",
    });
    const dialogCanvas = within(dialog);

    await expect(
      dialogCanvas.getByRole("button", { name: "削除中…" }),
    ).toBeDisabled();
    await expect(
      dialogCanvas.getByRole("button", { name: "キャンセル" }),
    ).toBeDisabled();
    await userEvent.keyboard("{Escape}");
    await expect(dialog).toBeInTheDocument();
    await expect(args.onClose).not.toHaveBeenCalled();
  },
};

export const LongProfileName: Story = {
  args: {
    profile: longProfile,
  },
  parameters: {
    viewport: {
      defaultViewport: "mobile1",
    },
  },
};

export const CancelAndRestoreFocus: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement);
    const body = within(canvasElement.ownerDocument.body);
    const dialog = await body.findByRole("alertdialog", {
      name: "接続プロファイルを削除",
    });

    await userEvent.click(
      within(dialog).getByRole("button", { name: "キャンセル" }),
    );

    await expect(args.onClose).toHaveBeenCalledOnce();
    await waitFor(() =>
      expect(
        canvas.getByRole("button", { name: "削除を開いたボタン" }),
      ).toHaveFocus(),
    );
  },
};
