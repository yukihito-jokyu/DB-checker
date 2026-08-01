import type { Meta, StoryObj } from "@storybook/react-vite";
import { toast } from "sonner";
import { expect, userEvent, within } from "storybook/test";

import { Button } from "./button";
import { Toaster } from "./sonner";

const meta = {
  title: "Components/UI/Sonner",
  component: Toaster,
  render: () => (
    <>
      <div className="flex gap-3">
        <Button
          type="button"
          onClick={() => toast.success("接続プロファイルを保存しました。")}
        >
          成功通知
        </Button>
        <Button
          type="button"
          variant="destructive"
          onClick={() =>
            toast.error("接続プロファイルを保存できませんでした。")
          }
        >
          エラー通知
        </Button>
      </div>
      <Toaster />
    </>
  ),
} satisfies Meta<typeof Toaster>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Success: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const body = within(canvasElement.ownerDocument.body);

    await userEvent.click(canvas.getByRole("button", { name: "成功通知" }));

    await expect(
      await body.findByText("接続プロファイルを保存しました。"),
    ).toBeInTheDocument();
  },
};

export const ErrorToast: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const body = within(canvasElement.ownerDocument.body);

    await userEvent.click(canvas.getByRole("button", { name: "エラー通知" }));

    await expect(
      await body.findByText("接続プロファイルを保存できませんでした。"),
    ).toBeInTheDocument();
  },
};
