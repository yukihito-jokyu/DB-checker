import type { Meta, StoryObj } from "@storybook/react-vite";
import { Database } from "lucide-react";
import { expect, fn, userEvent, within } from "storybook/test";

import { Button } from "./button";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "./empty";

const onAction = fn();

const meta = {
  title: "Components/UI/Empty",
  component: Empty,
  beforeEach: () => {
    onAction.mockClear();
  },
  render: () => (
    <Empty>
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <Database />
        </EmptyMedia>
        <EmptyTitle>接続プロファイルがありません</EmptyTitle>
        <EmptyDescription>
          接続先を追加して、使用するデータベースを選択してください。
        </EmptyDescription>
      </EmptyHeader>
      <EmptyContent>
        <Button onClick={onAction}>接続プロファイルを追加</Button>
      </EmptyContent>
    </Empty>
  ),
} satisfies Meta<typeof Empty>;

export default meta;
type Story = StoryObj<typeof meta>;

export const WithAction: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await userEvent.click(
      canvas.getByRole("button", { name: "接続プロファイルを追加" }),
    );

    await expect(onAction).toHaveBeenCalledOnce();
  },
};

export const DescriptionOnly: Story = {
  render: () => (
    <Empty>
      <EmptyHeader>
        <EmptyTitle>検索結果がありません</EmptyTitle>
        <EmptyDescription>
          条件を変更して、もう一度検索してください。
        </EmptyDescription>
      </EmptyHeader>
    </Empty>
  ),
};
