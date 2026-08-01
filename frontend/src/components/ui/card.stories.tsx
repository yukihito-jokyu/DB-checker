import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button } from "./button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "./card";

const meta = {
  title: "Components/UI/Card",
  component: Card,
  render: (args) => (
    <Card className="max-w-md" {...args}>
      <CardHeader>
        <CardTitle>PostgreSQL ローカル</CardTitle>
        <CardDescription>開発環境の接続プロファイル</CardDescription>
      </CardHeader>
      <CardContent>
        <p>127.0.0.1:5432 · db_checker · public</p>
      </CardContent>
      <CardFooter className="justify-end gap-2">
        <Button variant="outline">編集</Button>
        <Button>使用する</Button>
      </CardFooter>
    </Card>
  ),
} satisfies Meta<typeof Card>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const LongContent: Story = {
  render: (args) => (
    <Card className="max-w-xs" {...args}>
      <CardHeader>
        <CardTitle className="break-words">
          非常に長い名前を持つステージング環境の読み取り専用接続プロファイル
        </CardTitle>
        <CardDescription>
          狭い画面でも情報がカードの外側にはみ出さないことを確認します。
        </CardDescription>
      </CardHeader>
      <CardContent>
        <p className="break-all">
          database.staging.internal.example.com:5432 ·
          application_reporting_database · analytics
        </p>
      </CardContent>
    </Card>
  ),
  parameters: {
    viewport: {
      defaultViewport: "mobile1",
    },
  },
};
