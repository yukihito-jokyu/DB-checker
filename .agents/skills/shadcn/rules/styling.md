# スタイリングとカスタマイズ

テーマ、CSS 変数、カスタムカラーの追加については [customization.md](../customization.md) を参照してください。

## 目次

- セマンティックカラー
- まず built-in variant を使う
- className はレイアウト専用
- space-x-* / space-y-* は使わない
- 幅と高さが等しい場合は w-* h-* より size-* を優先
- truncate shorthand を優先
- 手動の dark: カラー上書きは禁止
- 条件付きクラスには cn() を使う
- オーバーレイコンポーネントに手動 z-index を付けない

---

## セマンティックカラー

**誤り:**

```tsx
<div className="bg-blue-500 text-white">
  <p className="text-gray-600">Secondary text</p>
</div>
```

**正しい:**

```tsx
<div className="bg-primary text-primary-foreground">
  <p className="text-muted-foreground">Secondary text</p>
</div>
```

---

## 状態・ステータス表示に生のカラー値を使わない

positive、negative、status indicator には、Badge variant、`text-destructive` のようなセマンティックトークン、またはカスタム CSS 変数を使います。生の Tailwind カラーに手を伸ばさないでください。

**誤り:**

```tsx
<span className="text-emerald-600">+20.1%</span>
<span className="text-green-500">Active</span>
<span className="text-red-600">-3.2%</span>
```

**正しい:**

```tsx
<Badge variant="secondary">+20.1%</Badge>
<Badge>Active</Badge>
<span className="text-destructive">-3.2%</span>
```

success / positive 用の色がセマンティックトークンとして存在しない場合は、Badge variant を使うか、テーマにカスタム CSS 変数を追加するかをユーザーに確認します（[customization.md](../customization.md) 参照）。

---

## まず built-in variant を使う

**誤り:**

```tsx
<Button className="border border-input bg-transparent hover:bg-accent">
  Click me
</Button>
```

**正しい:**

```tsx
<Button variant="outline">Click me</Button>
```

---

## className はレイアウト専用

`className` は `max-w-md`、`mx-auto`、`mt-4` などのレイアウトに使います。コンポーネントの色やタイポグラフィを上書きするためには使いません。色を変える場合は、セマンティックトークン、built-in variant、または CSS 変数を使います。

**誤り:**

```tsx
<Card className="bg-blue-100 text-blue-900 font-bold">
  <CardContent>Dashboard</CardContent>
</Card>
```

**正しい:**

```tsx
<Card className="max-w-md mx-auto">
  <CardContent>Dashboard</CardContent>
</Card>
```

コンポーネントの見た目をカスタマイズする場合は、次の順で優先します。

1. **Built-in variants** — `variant="outline"`、`variant="destructive"` など。
2. **セマンティックカラートークン** — `bg-primary`、`text-muted-foreground`。
3. **CSS 変数** — グローバル CSS ファイルでカスタムカラーを定義します（[customization.md](../customization.md) 参照）。

---

## space-x-* / space-y-* は使わない

代わりに `gap-*` を使います。`space-y-4` → `flex flex-col gap-4`。`space-x-2` → `flex gap-2`。

```tsx
<div className="flex flex-col gap-4">
  <Input />
  <Input />
  <Button>Submit</Button>
</div>
```

---

## 幅と高さが等しい場合は w-* h-* より size-* を優先

`w-10 h-10` ではなく `size-10` を使います。アイコン、avatar、skeleton などに適用されます。

---

## truncate shorthand を優先

`overflow-hidden text-ellipsis whitespace-nowrap` ではなく `truncate` を使います。

---

## 手動の dark: カラー上書きは禁止

セマンティックトークンを使います。これらは CSS 変数により light/dark を処理します。`bg-white dark:bg-gray-950` ではなく `bg-background text-foreground` を使います。

---

## 条件付きクラスには cn() を使う

条件付きまたは merge された class name には、プロジェクトの `cn()` utility を使います。className 文字列内に手動 ternary を書かないでください。

**誤り:**

```tsx
<div className={`flex items-center ${isActive ? "bg-primary text-primary-foreground" : "bg-muted"}`}>
```

**正しい:**

```tsx
import { cn } from "@/lib/utils"

<div className={cn("flex items-center", isActive ? "bg-primary text-primary-foreground" : "bg-muted")}>
```

---

## オーバーレイコンポーネントに手動 z-index を付けない

`Dialog`、`Sheet`、`Drawer`、`AlertDialog`、`DropdownMenu`、`Popover`、`Tooltip`、`HoverCard` は自身の stacking を処理します。`z-50` や `z-[999]` を追加してはいけません。
