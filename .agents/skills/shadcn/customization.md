# カスタマイズとテーマ

コンポーネントはセマンティックな CSS variable token を参照します。変数を変更すると、すべてのコンポーネントに反映されます。

## 目次

- 仕組み（CSS variables → Tailwind utilities → components）
- Color variables と OKLCH format
- Dark mode setup
- Theme の変更（presets、CSS variables）
- Custom colors の追加（Tailwind v3 と v4）
- Border radius
- コンポーネントのカスタマイズ（variants、className、wrappers）
- Updates の確認

---

## 仕組み

1. CSS variables を `:root`（light）と `.dark`（dark mode）に定義します。
2. Tailwind がそれらを `bg-primary`、`text-muted-foreground` などの utilities に map します。
3. コンポーネントはこれらの utilities を使います。variable を変更すると、それを参照するすべての components が変わります。

---

## Color Variables

すべての色は `name` / `name-foreground` の規約に従います。base variable は background 用、`-foreground` はその背景上の text/icons 用です。

| Variable                                     | 目的 |
| -------------------------------------------- | ---- |
| `--background` / `--foreground`              | ページ背景とデフォルトテキスト |
| `--card` / `--card-foreground`               | Card surface |
| `--primary` / `--primary-foreground`         | Primary buttons と actions |
| `--secondary` / `--secondary-foreground`     | Secondary actions |
| `--muted` / `--muted-foreground`             | Muted/disabled states |
| `--accent` / `--accent-foreground`           | Hover と accent states |
| `--destructive` / `--destructive-foreground` | Error と destructive actions |
| `--border`                                   | デフォルト border color |
| `--input`                                    | Form input borders |
| `--ring`                                     | Focus ring color |
| `--chart-1` から `--chart-5`                 | Chart/data visualization |
| `--sidebar-*`                                | Sidebar 固有の colors |
| `--surface` / `--surface-foreground`         | Secondary surface |

色は OKLCH を使います。例: `--primary: oklch(0.205 0 0)`。値は lightness（0–1）、chroma（0 = gray）、hue（0–360）です。

---

## Dark Mode

root element 上の `.dark` による class-based toggle です。Next.js では `next-themes` を使います。

```tsx
import { ThemeProvider } from "next-themes"

<ThemeProvider attribute="class" defaultTheme="system" enableSystem>
  {children}
</ThemeProvider>
```

---

## Theme の変更

```bash
# ui.shadcn.com の preset code を適用する。
npx shadcn@latest apply --preset a2r6bw

# positional shorthand も使える。
npx shadcn@latest apply a2r6bw

# named preset に切り替え、既存 components を上書きする。
npx shadcn@latest apply --preset nova

# 代わりに既存 components を維持する。
npx shadcn@latest init --preset nova --force --no-reinstall

# custom theme URL を使う。
npx shadcn@latest apply --preset "https://ui.shadcn.com/init?base=radix&style=nova&theme=blue&..."
```

または、`globals.css` の CSS variables を直接編集します。

---

## Custom Colors の追加

`npx shadcn@latest info` の `tailwindCssFile` にあるファイル（通常は `globals.css`）へ variables を追加します。このために新しい CSS ファイルを作ってはいけません。

```css
/* 1. global CSS file に定義する。 */
:root {
  --warning: oklch(0.84 0.16 84);
  --warning-foreground: oklch(0.28 0.07 46);
}
.dark {
  --warning: oklch(0.41 0.11 46);
  --warning-foreground: oklch(0.99 0.02 95);
}
```

```css
/* 2a. Tailwind v4 (@theme inline) に登録する。 */
@theme inline {
  --color-warning: var(--warning);
  --color-warning-foreground: var(--warning-foreground);
}
```

`tailwindVersion` が `"v3"` の場合（`npx shadcn@latest info` で確認）、代わりに `tailwind.config.js` に登録します。

```js
// 2b. Tailwind v3 (tailwind.config.js) に登録する。
module.exports = {
  theme: {
    extend: {
      colors: {
        warning: "oklch(var(--warning) / <alpha-value>)",
        "warning-foreground":
          "oklch(var(--warning-foreground) / <alpha-value>)",
      },
    },
  },
}
```

```tsx
// 3. components で使う。
<div className="bg-warning text-warning-foreground">Warning</div>
```

---

## Border Radius

`--radius` は border radius をグローバルに制御します。コンポーネントはそこから値を派生させます（`rounded-lg` = `var(--radius)`、`rounded-md` = `calc(var(--radius) - 2px)`）。

---

## コンポーネントのカスタマイズ

誤り / 正しい例は [rules/styling.md](./rules/styling.md) も参照してください。

次の順で優先します。

### 1. Built-in variants

```tsx
<Button variant="outline" size="sm">
  Click
</Button>
```

### 2. `className` 経由の Tailwind classes

```tsx
<Card className="mx-auto max-w-md">...</Card>
```

### 3. 新しい variant を追加する

component source を編集し、`cva` で variant を追加します。

```tsx
// components/ui/button.tsx
warning: "bg-warning text-warning-foreground hover:bg-warning/90",
```

### 4. Wrapper components

shadcn/ui primitives をより上位の components に構成します。

```tsx
export function ConfirmDialog({ title, description, onConfirm, children }) {
  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>{children}</AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm}>Confirm</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
```

---

## Updates の確認

```bash
npx shadcn@latest add button --diff
```

更新前に実際の変更内容を preview するには、`--dry-run` と `--diff` を使います。

```bash
npx shadcn@latest add button --dry-run        # 影響を受けるすべての files を見る
npx shadcn@latest add button --diff button.tsx # 特定 file の diff を見る
```

完全な smart merge workflow は [SKILL.md の コンポーネント更新](./SKILL.md#コンポーネント更新) を参照してください。
