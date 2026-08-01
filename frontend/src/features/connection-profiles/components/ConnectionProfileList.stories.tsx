import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, fn, userEvent, within } from "storybook/test";

import { longProfile, mysqlProfile, postgresProfile } from "../storybook";
import { ConnectionProfileList } from "./ConnectionProfileList";

const meta = {
  title: "Features/Connection Profiles/Connection Profile List",
  component: ConnectionProfileList,
  args: {
    actionsDisabled: false,
    activeProfileId: postgresProfile.id,
    operation: null,
    profiles: [postgresProfile, mysqlProfile],
    onActivate: fn(),
    onDelete: fn(),
    onEdit: fn(),
  },
} satisfies Meta<typeof ConnectionProfileList>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement);
    const items = canvas.getAllByRole("listitem");
    const mysqlItem = within(items[1]);

    await userEvent.click(
      mysqlItem.getByRole("button", { name: "この接続先を使用" }),
    );
    await expect(args.onActivate).toHaveBeenCalledWith(mysqlProfile.id);

    await userEvent.click(mysqlItem.getByRole("button", { name: "編集" }));
    await expect(args.onEdit).toHaveBeenCalledWith(
      mysqlProfile,
      expect.anything(),
    );

    await userEvent.click(mysqlItem.getByRole("button", { name: "削除" }));
    await expect(args.onDelete).toHaveBeenCalledWith(
      mysqlProfile,
      expect.anything(),
    );
  },
};

export const NoActiveProfile: Story = {
  args: {
    activeProfileId: undefined,
  },
};

export const Activating: Story = {
  args: {
    actionsDisabled: true,
    operation: "activating",
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const switchingButton = canvas.getByRole("button", { name: "切替中…" });

    await expect(switchingButton).toBeDisabled();
  },
};

export const ActionsDisabled: Story = {
  args: {
    actionsDisabled: true,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    for (const button of canvas.getAllByRole("button")) {
      await expect(button).toBeDisabled();
    }
  },
};

export const LongContent: Story = {
  args: {
    activeProfileId: undefined,
    profiles: [longProfile],
  },
  parameters: {
    viewport: {
      defaultViewport: "mobile1",
    },
  },
};
