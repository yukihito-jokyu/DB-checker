import type { Meta, StoryObj } from "@storybook/react-vite";

import { Separator } from "./separator";

const meta = {
  title: "Components/UI/Separator",
  component: Separator,
} satisfies Meta<typeof Separator>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Horizontal: Story = {
  render: () => (
    <div className="grid gap-4">
      <span>接続情報</span>
      <Separator />
      <span>認証情報</span>
    </div>
  ),
};

export const Vertical: Story = {
  render: () => (
    <div className="flex h-8 items-center gap-4">
      <span>PostgreSQL</span>
      <Separator orientation="vertical" />
      <span>localhost:5432</span>
    </div>
  ),
};

export const Semantic: Story = {
  args: {
    "aria-label": "接続情報と認証情報の区切り",
    decorative: false,
  },
};
