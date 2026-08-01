import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, within } from "storybook/test";

import { Input } from "./input";

const meta = {
  title: "Components/UI/Input",
  component: Input,
  args: {
    "aria-label": "ホスト",
    placeholder: "localhost",
  },
} satisfies Meta<typeof Input>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = canvas.getByRole("textbox", { name: "ホスト" });

    await userEvent.type(input, "db.example.com");

    await expect(input).toHaveValue("db.example.com");
  },
};

export const WithValue: Story = {
  args: {
    defaultValue: "127.0.0.1",
  },
};

export const Disabled: Story = {
  args: {
    defaultValue: "変更できません",
    disabled: true,
  },
};

export const Invalid: Story = {
  args: {
    "aria-describedby": "host-error",
    "aria-invalid": true,
    defaultValue: "invalid host",
  },
  render: (args) => (
    <div className="grid gap-2">
      <Input {...args} />
      <p className="text-sm text-destructive" id="host-error">
        ホストに空白は使用できません。
      </p>
    </div>
  ),
};
