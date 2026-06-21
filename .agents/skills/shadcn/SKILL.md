---
name: shadcn
description: shadcn components と projects を管理します — 追加、検索、修正、デバッグ、スタイリング、UI 構成。project context、component docs、usage examples を提供します。shadcn/ui、component registries、presets、--preset codes、または components.json file を持つ project を扱うときに適用します。"shadcn init"、"--preset で app を作成"、"--preset に切り替え" にも反応します。
user-invocable: false
allowed-tools: Bash(npx shadcn@latest *), Bash(pnpm dlx shadcn@latest *), Bash(bunx --bun shadcn@latest *)
---

# shadcn/ui

UI、components、design systems を構築するための framework です。コンポーネントは CLI によって source code としてユーザーの project に追加されます。

> **IMPORTANT:** すべての CLI commands は、project の `packageManager` に応じて project の package runner で実行します: `npx shadcn@latest`、`pnpm dlx shadcn@latest`、または `bunx --bun shadcn@latest`。以下の例は `npx shadcn@latest` を使っていますが、project に合う runner に置き換えてください。

## 現在のプロジェクトコンテキスト

```json
!`npx shadcn@latest info --json`
```

上の JSON には project config と installed components が含まれます。任意の component の documentation と example URLs を取得するには `npx shadcn@latest docs <component>` を使います。

## Principles

1. **まず既存 components を使う。** custom UI を書く前に `npx shadcn@latest search` で registries を確認します。community registries も確認します。
2. **再発明せず composition する。** Settings page = Tabs + Card + form controls。Dashboard = Sidebar + Card + Chart + Table。
3. **custom styles より built-in variants を先に使う。** `variant="outline"`、`size="sm"` など。
4. **semantic colors を使う。** `bg-primary`、`text-muted-foreground`。`bg-blue-500` のような raw values は使いません。

## Critical Rules

これらの rules は**常に適用**されます。各項目は誤り / 正しい code pairs を含む file に link しています。

### Styling & Tailwind → [styling.md](./rules/styling.md)

- **`className` は layout 用であり styling 用ではありません。** component colors や typography を上書きしません。
- **`space-x-*` または `space-y-*` は使いません。** `gap-*` 付きの `flex` を使います。vertical stack では `flex flex-col gap-*`。
- **width と height が等しい場合は `size-*` を使います。** `w-10 h-10` ではなく `size-10`。
- **`truncate` shorthand を使います。** `overflow-hidden text-ellipsis whitespace-nowrap` は使いません。
- **手動の `dark:` color overrides は使いません。** semantic tokens（`bg-background`、`text-muted-foreground`）を使います。
- **conditional classes には `cn()` を使います。** manual template literal ternaries は書きません。
- **overlay components に手動 `z-index` を付けません。** Dialog、Sheet、Popover などは自身の stacking を処理します。

### Forms & Inputs → [forms.md](./rules/forms.md)

- **Forms は `FieldGroup` + `Field` を使います。** form layout に raw `div` + `space-y-*` や `grid gap-*` を使いません。
- **`InputGroup` は `InputGroupInput`/`InputGroupTextarea` を使います。** `InputGroup` 内に raw `Input`/`Textarea` を置きません。
- **input 内の buttons は `InputGroup` + `InputGroupAddon` を使います。**
- **選択肢セット（2〜7択）は `ToggleGroup` を使います。** manual active state 付きで `Button` を loop しません。
- **関連 checkbox/radio の grouping には `FieldSet` + `FieldLegend` を使います。** heading 付き `div` は使いません。
- **Field validation は `data-invalid` + `aria-invalid` を使います。** `Field` に `data-invalid`、control に `aria-invalid`。disabled では `Field` に `data-disabled`、control に `disabled`。

### Component Structure → [composition.md](./rules/composition.md)

- **Items は必ず Group の中に置きます。** `SelectItem` → `SelectGroup`。`DropdownMenuItem` → `DropdownMenuGroup`。`CommandItem` → `CommandGroup`。
- **custom triggers には `asChild` (radix) または `render` (base) を使います。** `npx shadcn@latest info` の `base` field を確認します。→ [base-vs-radix.md](./rules/base-vs-radix.md)
- **Dialog、Sheet、Drawer には必ず Title が必要です。** アクセシビリティのため `DialogTitle`、`SheetTitle`、`DrawerTitle` が必要です。視覚的に隠す場合は `className="sr-only"` を使います。
- **完全な Card composition を使います。** `CardHeader`/`CardTitle`/`CardDescription`/`CardContent`/`CardFooter`。すべてを `CardContent` に押し込まないでください。
- **Button に `isPending`/`isLoading` はありません。** `Spinner` + `data-icon` + `disabled` で構成します。
- **`TabsTrigger` は `TabsList` 内に置く必要があります。** trigger を `Tabs` 直下に直接 render しません。
- **`Avatar` には必ず `AvatarFallback` が必要です。** image load 失敗時のためです。

### Custom Markup ではなく Components を使う → [composition.md](./rules/composition.md)

- **custom markup より先に既存 components を使います。** styled `div` を書く前に component が存在するか確認します。
- **Callouts には `Alert` を使います。** custom styled div を作りません。
- **Empty states には `Empty` を使います。** custom empty state markup を作りません。
- **Toast は `sonner` 経由です。** `sonner` の `toast()` を使います。
- **`<hr>` や `<div className="border-t">` ではなく `Separator` を使います。**
- **loading placeholders には `Skeleton` を使います。** custom `animate-pulse` divs は使いません。
- **custom styled spans ではなく `Badge` を使います。**

### Icons → [icons.md](./rules/icons.md)

- **`Button` 内の icons には `data-icon` を使います。** icon に `data-icon="inline-start"` または `data-icon="inline-end"` を付けます。
- **components 内の icons に sizing classes を付けません。** component は CSS で icon sizing を処理します。`size-4` や `w-4 h-4` は使いません。
- **icons は string keys ではなく objects として渡します。** string lookup ではなく `icon={CheckIcon}`。

### CLI

- **preset codes を decode したり preset URLs を手動で構築したりしてはいけません。** `npx shadcn@latest preset decode <code>`、`preset url <code>`、または `preset open <code>` を使います。project-aware preset detection には `npx shadcn@latest preset resolve` を使います。
- **preset codes は CLI へ直接適用します。** 既存 projects には `npx shadcn@latest apply <code>`、初期化時には `npx shadcn@latest init --preset <code>` を使います。

## Key Patterns

これらは正しい shadcn/ui code を特徴づける最も一般的な patterns です。edge cases は上記の linked rule files を参照してください。

```tsx
// Form layout: div + Label ではなく FieldGroup + Field。
<FieldGroup>
  <Field>
    <FieldLabel htmlFor="email">Email</FieldLabel>
    <Input id="email" />
  </Field>
</FieldGroup>

// Validation: Field に data-invalid、control に aria-invalid。
<Field data-invalid>
  <FieldLabel>Email</FieldLabel>
  <Input aria-invalid />
  <FieldDescription>Invalid email.</FieldDescription>
</Field>

// Buttons 内の icons: data-icon、sizing classes なし。
<Button>
  <SearchIcon data-icon="inline-start" />
  Search
</Button>

// Spacing: space-y-* ではなく gap-*。
<div className="flex flex-col gap-4">  // correct
<div className="space-y-4">           // wrong

// Equal dimensions: w-* h-* ではなく size-*。
<Avatar className="size-10">   // correct
<Avatar className="w-10 h-10"> // wrong

// Status colors: raw colors ではなく Badge variants または semantic tokens。
<Badge variant="secondary">+20.1%</Badge>    // correct
<span className="text-emerald-600">+20.1%</span> // wrong
```

## Component Selection

| 必要なもの | 使うもの |
| ---------- | -------- |
| Button/action | 適切な variant を持つ `Button` |
| Form inputs | `Input`, `Select`, `Combobox`, `Switch`, `Checkbox`, `RadioGroup`, `Textarea`, `InputOTP`, `Slider` |
| 2〜5 options の切り替え | `ToggleGroup` + `ToggleGroupItem` |
| Data display | `Table`, `Card`, `Badge`, `Avatar` |
| Navigation | `Sidebar`, `NavigationMenu`, `Breadcrumb`, `Tabs`, `Pagination` |
| Overlays | `Dialog` (modal), `Sheet` (side panel), `Drawer` (bottom sheet), `AlertDialog` (confirmation) |
| Feedback | `sonner` (toast), `Alert`, `Progress`, `Skeleton`, `Spinner` |
| Command palette | `Dialog` 内の `Command` |
| Charts | `Chart` (Recharts を wrap) |
| Layout | `Card`, `Separator`, `Resizable`, `ScrollArea`, `Accordion`, `Collapsible` |
| Empty states | `Empty` |
| Menus | `DropdownMenu`, `ContextMenu`, `Menubar` |
| Tooltips/info | `Tooltip`, `HoverCard`, `Popover` |

## Key Fields

注入される project context には次の key fields が含まれます。

- **`aliases`** → imports には実際の alias prefix（例: `@/`, `~/`）を使います。hardcode しません。
- **`isRSC`** → `true` の場合、`useState`、`useEffect`、event handlers、browser APIs を使う components には file 先頭に `"use client"` が必要です。この directive を助言するときは必ずこの field を参照します。
- **`tailwindVersion`** → `"v4"` は `@theme inline` blocks を使い、`"v3"` は `tailwind.config.js` を使います。
- **`tailwindCssFile`** → custom CSS variables が定義される global CSS file。必ずこの file を編集し、新規 file は作りません。
- **`style`** → component visual treatment（例: `nova`, `vega`）。
- **`base`** → primitive library（`radix` または `base`）。component APIs と available props に影響します。
- **`iconLibrary`** → icon imports を決めます。`lucide` なら `lucide-react`、`tabler` なら `@tabler/icons-react` などを使います。`lucide-react` だと決めつけません。
- **`resolvedPaths`** → components、utils、hooks などの正確な file-system destinations。
- **`framework`** → routing と file conventions（例: Next.js App Router vs Vite SPA）。
- **`packageManager`** → shadcn 以外の dependency installs に使います（例: `pnpm add date-fns` vs `npm install date-fns`）。
- **`preset`** → current project の resolved preset code と values。preset 情報だけが必要な場合は `npx shadcn@latest preset resolve --json` を使います。

完全な field reference は [cli.md — `info` command](./cli.md) を参照してください。

## Component Docs, Examples, and Usage

component の documentation、examples、API reference の URLs を取得するには `npx shadcn@latest docs <component>` を実行します。実際の content を得るにはこれらの URLs を fetch します。

```bash
npx shadcn@latest docs button dialog select
```

**component を作成、修正、デバッグ、または使用するときは、必ず最初に `npx shadcn@latest docs` を実行し、URLs を fetch します。** これにより、推測ではなく正しい API と usage patterns に基づいて作業できます。

## Workflow

1. **project context を取得する** — 上ですでに注入されています。refresh が必要な場合は `npx shadcn@latest info` を再実行します。
2. **installed components を先に確認する** — `add` を実行する前に、project context の `components` list または `resolvedPaths.ui` directory を必ず確認します。追加されていない components を import せず、すでに installed のものを再追加しません。
3. **components を探す** — `npx shadcn@latest search`。
4. **docs と examples を取得する** — `npx shadcn@latest docs <component>` で URLs を取得し、その後 fetch します。installed していない registry items を閲覧するには `npx shadcn@latest view` を使います。installed components への変更を preview するには `npx shadcn@latest add --diff` を使います。
5. **install または update する** — `npx shadcn@latest add`。既存 components を update する場合は、先に `--dry-run` と `--diff` で変更を preview します（下の [コンポーネント更新](#コンポーネント更新) 参照）。
6. **third-party components の imports を修正する** — community registries（例: `@bundui`, `@magicui`）から components を追加した後、追加された non-UI files に `@/components/ui/...` のような hardcoded import paths がないか確認します。これらは project の実 alias と一致しない可能性があります。`npx shadcn@latest info` で正しい `ui` alias（例: `@workspace/ui/components`）を取得し、それに合わせて imports を書き換えます。CLI は自身の UI files の imports を書き換えますが、third-party registry components は project に合わない default paths を使う場合があります。
7. **追加された components を review する** — 任意の registry から component または block を追加した後は、**必ず追加 files を読み、正しいことを検証します**。missing sub-components（例: `SelectGroup` なしの `SelectItem`）、missing imports、incorrect composition、[Critical Rules](#critical-rules) 違反を確認します。また、icon imports は project context の `iconLibrary` に置き換えます（例: registry item が `lucide-react` を使っているが project が `hugeicons` を使う場合は、imports と icon names を適切に差し替えます）。すべての問題を修正してから先に進みます。
8. **Registry は明示されている必要がある** — ユーザーが block または component の追加を依頼したとき、**registry を推測してはいけません**。registry が指定されていない場合（例: `@shadcn`、`@tailark`、`owner/repo` などを指定せず「login block を追加して」と言う場合）は、どの registry を使うか確認します。ユーザーの代わりに default registry を選ばないでください。
9. **Presets の切り替え** — まずユーザーに確認します: **overwrite**、**partial**、**merge**、または **skip**?
   - **Inspect current preset**: `npx shadcn@latest preset resolve`。structured values が必要なら `--json` を使います。
   - **Inspect incoming preset**: `npx shadcn@latest preset decode <code>`。preset builder を共有または開くには `preset url <code>` または `preset open <code>` を使います。
   - **Overwrite**: `npx shadcn@latest apply <code>`。detected components、fonts、CSS variables を overwrite します。
   - **Partial**: `npx shadcn@latest apply <code> --only theme,font`。UI components を reinstall せず、選択した preset parts だけを update します。supported values は `theme` と `font` で、comma-separated combinations が可能です。`icon` は意図的にサポートされていません。icon changes は full component reinstall と transforms が必要になる場合があるためです。
   - **Merge**: `npx shadcn@latest init --preset <code> --force --no-reinstall` を実行し、続いて `npx shadcn@latest info` で installed components を list し、各 installed component に `--dry-run` と `--diff` を使って個別に [smart merge](#updating-components) します。
   - **Skip**: `npx shadcn@latest init --preset <code> --force --no-reinstall`。config と CSS だけを update し、components はそのままにします。
   - **Important**: preset commands は必ずユーザーの project directory 内で実行します。`apply` は `components.json` file を持つ既存 project でのみ動作します。CLI は `components.json` から現在の base（`base` vs `radix`）を自動的に保持します。scratch/temp directory を使う必要がある場合（例: `--dry-run` comparisons）、`--base <current-base>` を明示的に渡します。preset codes は base を encode していません。

## コンポーネント更新

ユーザーが local changes を保ったまま upstream から component を update したい場合は、`--dry-run` と `--diff` を使って賢く merge します。**GitHub から raw files を手動取得してはいけません。必ず CLI を使います。**

1. `npx shadcn@latest add <component> --dry-run` を実行し、影響を受けるすべての files を確認します。
2. 各 file について `npx shadcn@latest add <component> --diff <file>` を実行し、upstream と local の差分を確認します。
3. diff に基づいて file ごとに判断します。
   - local changes なし → overwrite して問題ありません。
   - local changes あり → local file を読み、diff を分析し、local modifications を保ちながら upstream updates を適用します。
   - ユーザーが "just update everything" と言う → `--overwrite` を使いますが、先に確認します。
4. **ユーザーの明示的な承認なしに `--overwrite` を使ってはいけません。**

## Quick Reference

```bash
# 新しい project を作成する。
npx shadcn@latest init --name my-app --preset base-nova
npx shadcn@latest init --name my-app --preset a2r6bw --template vite

# monorepo project を作成する。
npx shadcn@latest init --name my-app --preset base-nova --monorepo
npx shadcn@latest init --name my-app --preset base-nova --template next --monorepo

# 既存 project を初期化する。
npx shadcn@latest init --preset base-nova
npx shadcn@latest init --defaults  # shortcut: --template=next --preset=nova (base style implied)

# 既存 project に preset を適用する。
npx shadcn@latest apply a2r6bw
npx shadcn@latest apply a2r6bw --only theme
npx shadcn@latest apply a2r6bw --only font
npx shadcn@latest apply a2r6bw --only theme,font

# preset codes と project preset state を inspect する。
npx shadcn@latest preset decode a2r6bw
npx shadcn@latest preset url a2r6bw
npx shadcn@latest preset open a2r6bw
npx shadcn@latest preset resolve
npx shadcn@latest preset resolve --json

# components を追加する。
npx shadcn@latest add button card dialog
npx shadcn@latest add @magicui/shimmer-button
npx shadcn@latest add owner/repo/item
npx shadcn@latest add --all

# 追加 / update 前に変更を preview する。
npx shadcn@latest add button --dry-run
npx shadcn@latest add button --diff button.tsx
npx shadcn@latest add @acme/form --view button.tsx
npx shadcn@latest add owner/repo/item --dry-run

# registries を検索する。
npx shadcn@latest search @shadcn -q "sidebar"
npx shadcn@latest search @tailark -q "stats"
npx shadcn@latest search owner/repo -q "login"
npx shadcn@latest search                          # all configured registries
npx shadcn@latest search @shadcn -q "menu" -t ui  # filter by item type

# component docs と example URLs を取得する。
npx shadcn@latest docs button dialog select

# registry item details を表示する（未 install の items 用）。
npx shadcn@latest view @shadcn/button
npx shadcn@latest view owner/repo/item
```

**Named presets:** `nova`, `vega`, `maia`, `lyra`, `mira`, `luma`
**Templates:** `next`, `vite`, `start`, `react-router`, `astro`（すべて `--monorepo` 対応）と `laravel`（monorepo 非対応）
**Preset codes:** [ui.shadcn.com](https://ui.shadcn.com) 由来の version-prefixed base62 strings（例: `a2r6bw` または `b0`）。

## 詳細リファレンス

- [rules/forms.md](./rules/forms.md) — FieldGroup、Field、InputGroup、ToggleGroup、FieldSet、validation states
- [rules/composition.md](./rules/composition.md) — Groups、overlays、Card、Tabs、Avatar、Alert、Empty、Toast、Separator、Skeleton、Badge、Button loading
- [rules/icons.md](./rules/icons.md) — data-icon、icon sizing、icons を objects として渡す
- [rules/styling.md](./rules/styling.md) — Semantic colors、variants、className、spacing、size、truncate、dark mode、cn()、z-index
- [rules/base-vs-radix.md](./rules/base-vs-radix.md) — asChild と render、Select、ToggleGroup、Slider、Accordion
- [cli.md](./cli.md) — Commands、flags、presets、templates
- [registry.md](./registry.md) — source registries の authoring、`include`、item definitions、dependencies、GitHub registry rules
- [customization.md](./customization.md) — Theming、CSS variables、components の拡張
