import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, waitFor, within } from "storybook/test";

import { Button } from "./button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "./dialog";

const meta = {
  title: "Components/UI/Dialog",
  component: Dialog,
  render: () => (
    <Dialog>
      <DialogTrigger asChild>
        <Button>設定を開く</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>接続設定</DialogTitle>
          <DialogDescription>
            使用するデータベースの接続設定を確認します。
          </DialogDescription>
        </DialogHeader>
        <p>PostgreSQL · localhost:5432</p>
        <DialogFooter>
          <DialogClose asChild>
            <Button>完了</Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  ),
} satisfies Meta<typeof Dialog>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const body = within(canvasElement.ownerDocument.body);
    const trigger = canvas.getByRole("button", { name: "設定を開く" });

    await userEvent.click(trigger);

    const dialog = await body.findByRole("dialog", { name: "接続設定" });
    await expect(
      dialog.contains(canvasElement.ownerDocument.activeElement),
    ).toBe(true);

    await userEvent.click(within(dialog).getByRole("button", { name: "完了" }));
    await waitFor(() => expect(trigger).toHaveFocus());
  },
};

export const EscapeToClose: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const body = within(canvasElement.ownerDocument.body);
    const trigger = canvas.getByRole("button", { name: "設定を開く" });

    await userEvent.click(trigger);
    await body.findByRole("dialog", { name: "接続設定" });
    await userEvent.keyboard("{Escape}");

    await waitFor(() =>
      expect(
        body.queryByRole("dialog", { name: "接続設定" }),
      ).not.toBeInTheDocument(),
    );
    await expect(trigger).toHaveFocus();
  },
};
