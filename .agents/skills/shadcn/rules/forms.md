# フォームと入力

## 目次

- フォームには FieldGroup + Field を使う
- InputGroup には InputGroupInput/InputGroupTextarea が必要
- 入力内のボタンには InputGroup + InputGroupAddon を使う
- 選択肢セット（2〜7択）には ToggleGroup を使う
- 関連フィールドのグループ化には FieldSet + FieldLegend を使う
- フィールド検証と disabled 状態

---

## フォームには FieldGroup + Field を使う

必ず `FieldGroup` + `Field` を使います。`space-y-*` を付けた生の `div` は使いません。

```tsx
<FieldGroup>
  <Field>
    <FieldLabel htmlFor="email">Email</FieldLabel>
    <Input id="email" type="email" />
  </Field>
  <Field>
    <FieldLabel htmlFor="password">Password</FieldLabel>
    <Input id="password" type="password" />
  </Field>
</FieldGroup>
```

設定ページでは `Field orientation="horizontal"` を使います。視覚的に隠すラベルには `FieldLabel className="sr-only"` を使います。

**フォームコントロールの選び方:**

- シンプルなテキスト入力 → `Input`
- 事前定義された選択肢のドロップダウン → `Select`
- 検索可能なドロップダウン → `Combobox`
- ネイティブ HTML select（JS なし） → `native-select`
- 真偽値トグル → `Switch`（設定向け）または `Checkbox`（フォーム向け）
- 少数の選択肢からの単一選択 → `RadioGroup`
- 2〜5個の選択肢の切り替え → `ToggleGroup` + `ToggleGroupItem`
- OTP / 認証コード → `InputOTP`
- 複数行テキスト → `Textarea`

---

## InputGroup には InputGroupInput/InputGroupTextarea が必要

`InputGroup` 内で生の `Input` や `Textarea` を使ってはいけません。

**誤り:**

```tsx
<InputGroup>
  <Input placeholder="Search..." />
</InputGroup>
```

**正しい:**

```tsx
import { InputGroup, InputGroupInput } from "@/components/ui/input-group"

<InputGroup>
  <InputGroupInput placeholder="Search..." />
</InputGroup>
```

---

## 入力内のボタンには InputGroup + InputGroupAddon を使う

カスタム位置指定で `Button` を `Input` の直下や隣に直接置いてはいけません。

**誤り:**

```tsx
<div className="relative">
  <Input placeholder="Search..." className="pr-10" />
  <Button className="absolute right-0 top-0" size="icon">
    <SearchIcon />
  </Button>
</div>
```

**正しい:**

```tsx
import { InputGroup, InputGroupInput, InputGroupAddon } from "@/components/ui/input-group"

<InputGroup>
  <InputGroupInput placeholder="Search..." />
  <InputGroupAddon>
    <Button size="icon">
      <SearchIcon data-icon="inline-start" />
    </Button>
  </InputGroupAddon>
</InputGroup>
```

---

## 選択肢セット（2〜7択）には ToggleGroup を使う

active state を手動管理しながら `Button` コンポーネントをループしないでください。

**誤り:**

```tsx
const [selected, setSelected] = useState("daily")

<div className="flex gap-2">
  {["daily", "weekly", "monthly"].map((option) => (
    <Button
      key={option}
      variant={selected === option ? "default" : "outline"}
      onClick={() => setSelected(option)}
    >
      {option}
    </Button>
  ))}
</div>
```

**正しい:**

```tsx
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"

<ToggleGroup spacing={2}>
  <ToggleGroupItem value="daily">Daily</ToggleGroupItem>
  <ToggleGroupItem value="weekly">Weekly</ToggleGroupItem>
  <ToggleGroupItem value="monthly">Monthly</ToggleGroupItem>
</ToggleGroup>
```

ラベル付き toggle group では `Field` と組み合わせます。

```tsx
<Field orientation="horizontal">
  <FieldTitle id="theme-label">Theme</FieldTitle>
  <ToggleGroup aria-labelledby="theme-label" spacing={2}>
    <ToggleGroupItem value="light">Light</ToggleGroupItem>
    <ToggleGroupItem value="dark">Dark</ToggleGroupItem>
    <ToggleGroupItem value="system">System</ToggleGroupItem>
  </ToggleGroup>
</Field>
```

> **Note:** `defaultValue` と `type`/`multiple` props は base と radix で異なります。[base-vs-radix.md](./base-vs-radix.md#togglegroup) を参照してください。

---

## 関連フィールドのグループ化には FieldSet + FieldLegend を使う

関連する checkbox、radio、switch には `FieldSet` + `FieldLegend` を使います。見出し付きの `div` は使いません。

```tsx
<FieldSet>
  <FieldLegend variant="label">Preferences</FieldLegend>
  <FieldDescription>Select all that apply.</FieldDescription>
  <FieldGroup className="gap-3">
    <Field orientation="horizontal">
      <Checkbox id="dark" />
      <FieldLabel htmlFor="dark" className="font-normal">Dark mode</FieldLabel>
    </Field>
  </FieldGroup>
</FieldSet>
```

---

## フィールド検証と disabled 状態

両方の属性が必要です。`data-invalid`/`data-disabled` は field（label、description）をスタイルし、`aria-invalid`/`disabled` は control をスタイルします。

```tsx
// Invalid.
<Field data-invalid>
  <FieldLabel htmlFor="email">Email</FieldLabel>
  <Input id="email" aria-invalid />
  <FieldDescription>Invalid email address.</FieldDescription>
</Field>

// Disabled.
<Field data-disabled>
  <FieldLabel htmlFor="email">Email</FieldLabel>
  <Input id="email" disabled />
</Field>
```

`Input`、`Textarea`、`Select`、`Checkbox`、`RadioGroupItem`、`Switch`、`Slider`、`NativeSelect`、`InputOTP` のすべてで機能します。
