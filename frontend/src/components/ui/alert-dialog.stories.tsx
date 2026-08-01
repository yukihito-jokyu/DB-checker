import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, fn, userEvent, waitFor, within } from "storybook/test";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "./alert-dialog";
import { Button } from "./button";

const meta = {
  title: "Components/UI/Alert Dialog",
  component: AlertDialog,
  args: {
    onOpenChange: fn(),
  },
  render: (args) => (
    <AlertDialog onOpenChange={args.onOpenChange}>
      <AlertDialogTrigger asChild>
        <Button variant="destructive">削除</Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>接続プロファイルを削除</AlertDialogTitle>
          <AlertDialogDescription>
            この操作は元に戻せません。本当に削除しますか？
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>キャンセル</AlertDialogCancel>
          <AlertDialogAction className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
            削除する
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  ),
} satisfies Meta<typeof AlertDialog>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Confirm: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement);
    const body = within(canvasElement.ownerDocument.body);

    await userEvent.click(canvas.getByRole("button", { name: "削除" }));
    const dialog = await body.findByRole("alertdialog", {
      name: "接続プロファイルを削除",
    });
    await userEvent.click(
      within(dialog).getByRole("button", { name: "削除する" }),
    );

    await expect(args.onOpenChange).toHaveBeenLastCalledWith(false);
  },
};

export const Cancel: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const body = within(canvasElement.ownerDocument.body);
    const trigger = canvas.getByRole("button", { name: "削除" });

    await userEvent.click(trigger);
    const dialog = await body.findByRole("alertdialog", {
      name: "接続プロファイルを削除",
    });
    await userEvent.click(
      within(dialog).getByRole("button", { name: "キャンセル" }),
    );

    await waitFor(() => expect(trigger).toHaveFocus());
  },
};
