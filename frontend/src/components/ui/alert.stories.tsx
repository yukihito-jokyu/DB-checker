import type { Meta, StoryObj } from "@storybook/react-vite";
import { AlertCircle, CircleCheck } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "./alert";

const meta = {
  title: "Components/UI/Alert",
  component: Alert,
  render: (args) => (
    <Alert {...args}>
      <CircleCheck />
      <AlertTitle>接続を確認しました</AlertTitle>
      <AlertDescription>データベースへ正常に接続できました。</AlertDescription>
    </Alert>
  ),
} satisfies Meta<typeof Alert>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    variant: "success",
  },
};

export const Destructive: Story = {
  args: {
    variant: "destructive",
  },
  render: (args) => (
    <Alert {...args}>
      <AlertCircle />
      <AlertTitle>接続できませんでした</AlertTitle>
      <AlertDescription>
        接続情報を確認して、もう一度お試しください。
      </AlertDescription>
    </Alert>
  ),
};

export const LongDescription: Story = {
  render: (args) => (
    <Alert {...args}>
      <AlertCircle />
      <AlertTitle>設定を確認してください</AlertTitle>
      <AlertDescription>
        ホスト名、ポート番号、データベース名、ユーザー名が接続先の設定と一致しているか確認してから、もう一度接続をお試しください。
      </AlertDescription>
    </Alert>
  ),
  parameters: {
    viewport: {
      defaultViewport: "mobile1",
    },
  },
};
