import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, fn, userEvent, within } from "storybook/test";

import { Label } from "./label";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "./select";

const meta = {
  title: "Components/UI/Select",
  component: Select,
  args: {
    onValueChange: fn(),
  },
  render: (args) => (
    <div className="grid max-w-xs gap-2">
      <Label htmlFor="database-type">DB種別</Label>
      <Select {...args} defaultValue="postgres">
        <SelectTrigger id="database-type">
          <SelectValue placeholder="DB種別を選択" />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            <SelectLabel>対応データベース</SelectLabel>
            <SelectItem value="postgres">PostgreSQL</SelectItem>
            <SelectItem value="mysql">MySQL</SelectItem>
            <SelectSeparator />
            <SelectItem disabled value="sqlite">
              SQLite（未対応）
            </SelectItem>
          </SelectGroup>
        </SelectContent>
      </Select>
    </div>
  ),
} satisfies Meta<typeof Select>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement);
    const body = within(canvasElement.ownerDocument.body);

    await userEvent.click(canvas.getByRole("combobox", { name: "DB種別" }));
    await userEvent.click(await body.findByRole("option", { name: "MySQL" }));

    await expect(args.onValueChange).toHaveBeenCalledWith("mysql");
    await expect(
      canvas.getByRole("combobox", { name: "DB種別" }),
    ).toHaveTextContent("MySQL");
  },
};

export const KeyboardSelection: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement);
    const trigger = canvas.getByRole("combobox", { name: "DB種別" });

    trigger.focus();
    await userEvent.keyboard("{Enter}{ArrowDown}{Enter}");

    await expect(args.onValueChange).toHaveBeenCalledWith("mysql");
  },
};

export const Placeholder: Story = {
  render: (args) => (
    <div className="grid max-w-xs gap-2">
      <Label htmlFor="empty-database-type">DB種別</Label>
      <Select {...args}>
        <SelectTrigger id="empty-database-type">
          <SelectValue placeholder="DB種別を選択" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="postgres">PostgreSQL</SelectItem>
          <SelectItem value="mysql">MySQL</SelectItem>
        </SelectContent>
      </Select>
    </div>
  ),
};

export const Disabled: Story = {
  args: {
    disabled: true,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(
      canvas.getByRole("combobox", { name: "DB種別" }),
    ).toBeDisabled();
  },
};
