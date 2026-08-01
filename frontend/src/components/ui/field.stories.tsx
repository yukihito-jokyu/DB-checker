import type { Meta, StoryObj } from "@storybook/react-vite";

import {
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSeparator,
  FieldSet,
  FieldTitle,
} from "./field";
import { Input } from "./input";

const meta = {
  title: "Components/UI/Field",
  component: Field,
} satisfies Meta<typeof Field>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Vertical: Story = {
  render: () => (
    <FieldGroup>
      <Field>
        <FieldLabel htmlFor="field-host">ホスト</FieldLabel>
        <Input id="field-host" placeholder="localhost" />
        <FieldDescription>
          データベースサーバーのホスト名を入力します。
        </FieldDescription>
      </Field>
      <Field>
        <FieldLabel htmlFor="field-port">ポート</FieldLabel>
        <Input id="field-port" type="number" defaultValue="5432" />
      </Field>
    </FieldGroup>
  ),
};

export const Horizontal: Story = {
  render: () => (
    <Field orientation="horizontal">
      <FieldLabel htmlFor="horizontal-database">データベース名</FieldLabel>
      <FieldContent>
        <Input id="horizontal-database" defaultValue="db_checker" />
        <FieldDescription>接続対象のデータベースです。</FieldDescription>
      </FieldContent>
    </Field>
  ),
};

export const Responsive: Story = {
  render: () => (
    <FieldGroup>
      <Field orientation="responsive">
        <FieldLabel htmlFor="responsive-user">ユーザー名</FieldLabel>
        <Input id="responsive-user" defaultValue="db_checker" />
      </Field>
    </FieldGroup>
  ),
  parameters: {
    viewport: {
      defaultViewport: "mobile1",
    },
  },
};

export const Invalid: Story = {
  render: () => (
    <Field data-invalid="true">
      <FieldLabel htmlFor="invalid-host">ホスト</FieldLabel>
      <Input
        aria-describedby="invalid-host-error"
        aria-invalid="true"
        defaultValue="invalid host"
        id="invalid-host"
      />
      <FieldError id="invalid-host-error">
        ホストに空白は使用できません。
      </FieldError>
    </Field>
  ),
};

export const Disabled: Story = {
  render: () => (
    <Field data-disabled="true">
      <FieldLabel htmlFor="disabled-user">ユーザー名</FieldLabel>
      <Input disabled id="disabled-user" value="db_checker" readOnly />
    </Field>
  ),
};

export const FieldSetExample: Story = {
  render: () => (
    <FieldSet>
      <FieldLegend>接続設定</FieldLegend>
      <FieldDescription>
        データベースへの接続に使用する情報です。
      </FieldDescription>
      <FieldGroup>
        <Field>
          <FieldTitle>接続先</FieldTitle>
          <FieldDescription>PostgreSQL · localhost:5432</FieldDescription>
        </Field>
        <FieldSeparator>認証情報</FieldSeparator>
        <Field>
          <FieldError
            errors={[
              { message: "ユーザー名を入力してください。" },
              { message: "パスワードを確認してください。" },
            ]}
          />
        </Field>
      </FieldGroup>
    </FieldSet>
  ),
};
