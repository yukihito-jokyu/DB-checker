import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, within } from "storybook/test";

import { Input } from "./input";
import { Label } from "./label";

const meta = {
  title: "Components/UI/Label",
  component: Label,
  render: () => (
    <div className="grid gap-2">
      <Label htmlFor="database-name">データベース名</Label>
      <Input id="database-name" />
    </div>
  ),
} satisfies Meta<typeof Label>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = canvas.getByRole("textbox", { name: "データベース名" });

    await userEvent.click(canvas.getByText("データベース名"));

    await expect(input).toHaveFocus();
  },
};

export const DisabledControl: Story = {
  render: () => (
    <div className="grid gap-2">
      <Label htmlFor="disabled-database">データベース名</Label>
      <Input disabled id="disabled-database" value="db_checker" readOnly />
    </div>
  ),
};
