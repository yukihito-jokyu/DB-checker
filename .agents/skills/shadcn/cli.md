# shadcn CLI リファレンス

設定は `components.json` から読み取られます。

> **IMPORTANT:** コマンドは必ずプロジェクトの package runner で実行します: `npx shadcn@latest`、`pnpm dlx shadcn@latest`、または `bunx --bun shadcn@latest`。プロジェクトコンテキストの `packageManager` を確認して適切なものを選びます。以下の例は `npx shadcn@latest` を使っていますが、プロジェクトに合わせて置き換えてください。

> **IMPORTANT:** 下記に記載された flags だけを使います。flag を作ったり推測したりしてはいけません。ここにない flag は存在しません。CLI はプロジェクトの lockfile から package manager を自動検出します。`--package-manager` flag はありません。

## 目次

- コマンド: init, apply, add (dry-run, smart merge), search, view, docs, info, build
- テンプレート: next, vite, start, react-router, astro
- プリセット: named, code, URL formats and fields
- プリセット切り替え

---

## コマンド

### `init` — プロジェクトを初期化または作成する

```bash
npx shadcn@latest init [components...] [options]
```

既存プロジェクトで shadcn/ui を初期化するか、`--name` が指定された場合は新しいプロジェクトを作成します。同じ手順で components を任意で install できます。

| Flag                    | Short | 説明 | Default |
| ----------------------- | ----- | ---- | ------- |
| `--template <template>` | `-t`  | Template (next, start, vite, next-monorepo, react-router) | — |
| `--preset [name]`       | `-p`  | Preset configuration (named, code, or URL) | — |
| `--yes`                 | `-y`  | confirmation prompt を skip | `true` |
| `--defaults`            | `-d`  | defaults を使う（`--template=next --preset=base-nova`） | `false` |
| `--force`               | `-f`  | 既存 configuration を強制 overwrite | `false` |
| `--cwd <cwd>`           | `-c`  | Working directory | current |
| `--name <name>`         | `-n`  | 新規 project 名 | — |
| `--silent`              | `-s`  | output を抑制 | `false` |
| `--rtl`                 |       | RTL support を有効化 | — |
| `--reinstall`           |       | 既存 UI components を再 install | `false` |
| `--monorepo`            |       | monorepo project を scaffold | — |
| `--no-monorepo`         |       | monorepo prompt を skip | — |

`npx shadcn@latest create` は `npx shadcn@latest init` の alias です。

### `apply` — 既存プロジェクトに preset を適用する

```bash
npx shadcn@latest apply [preset] [options]
```

既存プロジェクトに preset を適用し、preset-driven config、fonts、CSS variables、検出済み UI components を overwrite します。

| Flag                | Short | 説明 | Default |
| ------------------- | ----- | ---- | ------- |
| `--preset <preset>` | —     | Preset configuration (named, code, or URL) | — |
| `--yes`             | `-y`  | confirmation prompt を skip | `false` |
| `--cwd <cwd>`       | `-c`  | Working directory | current |
| `--silent`          | `-s`  | output を抑制 | `false` |

`[preset]` は `--preset <preset>` の shorthand です。両方が指定された場合は一致している必要があります。preset が指定されない場合、CLI は `ui.shadcn.com/create` の custom preset builder を開く提案をします。

### `add` — components を追加する

> **IMPORTANT:** local components を upstream と比較する場合や変更を preview する場合は、必ず `npx shadcn@latest add <component> --dry-run`、`--diff`、または `--view` を使います。GitHub などから raw files を手動取得してはいけません。CLI が registry resolution、file paths、CSS diffing を自動的に処理します。

```bash
npx shadcn@latest add [components...] [options]
```

component names、registry-prefixed names（`@magicui/shimmer-button`）、GitHub item addresses（`owner/repo/item`）、URLs、local paths を受け付けます。

| Flag            | Short | 説明 | Default |
| --------------- | ----- | ---- | ------- |
| `--yes`         | `-y`  | confirmation prompt を skip | `false` |
| `--overwrite`   | `-o`  | 既存 files を overwrite | `false` |
| `--cwd <cwd>`   | `-c`  | Working directory | current |
| `--all`         | `-a`  | 利用可能なすべての components を追加 | `false` |
| `--path <path>` | `-p`  | component の target path | — |
| `--silent`      | `-s`  | output を抑制 | `false` |
| `--dry-run`     |       | file を書き込まず全変更を preview | `false` |
| `--diff [path]` |       | diffs を表示。path なしでは先頭5 files、path ありではその file のみ（`--dry-run` を含意） | — |
| `--view [path]` |       | file contents を表示。path なしでは先頭5 files、path ありではその file のみ（`--dry-run` を含意） | — |

#### Dry-Run Mode

`add` が行う内容を file 書き込みなしで preview するには `--dry-run` を使います。`--diff` と `--view` はどちらも `--dry-run` を含意します。

```bash
# すべての変更を preview。
npx shadcn@latest add button --dry-run

# すべての files の diff を表示（上位5件）。
npx shadcn@latest add button --diff

# 特定 file の diff を表示。
npx shadcn@latest add button --diff button.tsx

# すべての files の内容を表示（上位5件）。
npx shadcn@latest add button --view

# 特定 file の full content を表示。
npx shadcn@latest add button --view button.tsx

# URLs でも動作する。
npx shadcn@latest add https://api.npoint.io/abc123 --dry-run

# public GitHub registries でも動作する。
npx shadcn@latest add owner/repo/item --dry-run

# CSS diffs。
npx shadcn@latest add button --diff globals.css
```

**dry-run を使う場面:**

- ユーザーが「どの files が追加されるか」「何が変わるか」を尋ねた場合は `--dry-run` を使います。
- 既存 components を overwrite する前に、まず `--diff` で変更を preview します。
- install せず component source code を確認したい場合は `--view` を使います。
- `globals.css` への CSS 変更内容を確認する場合は `--diff globals.css` を使います。
- third-party registry code を install 前に review / audit する場合は、`--view` で source を確認します。

> **`npx shadcn@latest add --dry-run` と `npx shadcn@latest view`:** ユーザーが自分のプロジェクトへの変更を preview したい場合は、`npx shadcn@latest view` より `npx shadcn@latest add --dry-run/--diff/--view` を優先します。`npx shadcn@latest view` は raw registry metadata だけを表示します。`npx shadcn@latest add --dry-run` は、resolved file paths、既存 files との差分、CSS updates など、ユーザーのプロジェクトで実際に起きる内容を表示します。project context なしで registry info を閲覧したい場合にのみ `npx shadcn@latest view` を使います。

#### Upstream からの Smart Merge

完全な workflow は [SKILL.md の コンポーネント更新](./SKILL.md#コンポーネント更新) を参照してください。

### `search` — registries を検索する

```bash
npx shadcn@latest search [registries...] [options]
```

Registry 横断の fuzzy search です。`npx shadcn@latest list` としても使えます。namespace（`@acme`）、public GitHub registry sources（`owner/repo`）、registry catalog URLs をサポートします。`-q` なしではすべての items を一覧表示します。registries が渡されない場合は、`components.json` に設定されたすべての registry を検索します。

| Flag                | Short | 説明 | Default |
| ------------------- | ----- | ---- | ------- |
| `--query <query>`   | `-q`  | Search query | — |
| `--type <type>`     | `-t`  | item type で filter（例: `ui`, `block`, `hook`）。comma-separated | — |
| `--limit <number>`  | `-l`  | 表示する最大 items 数 | `100` |
| `--offset <number>` | `-o`  | skip する items 数 | `0` |
| `--json`            |       | JSON として output | `false` |
| `--cwd <cwd>`       | `-c`  | Working directory | current |

### `view` — item details を表示する

```bash
npx shadcn@latest view <items...> [options]
```

file contents を含む item info を表示します。例: `npx shadcn@latest view @shadcn/button`、`npx shadcn@latest view owner/repo/item`。

### `docs` — component documentation URLs を取得する

```bash
npx shadcn@latest docs <components...> [options]
```

component documentation、examples、API references の resolved URLs を出力します。1つ以上の component names を受け付けます。実際の content を得るには URLs を fetch します。

`npx shadcn@latest docs input button` の example output:

```txt
base  radix

input
  docs      https://ui.shadcn.com/docs/components/radix/input
  examples  https://raw.githubusercontent.com/.../examples/input-example.tsx

button
  docs      https://ui.shadcn.com/docs/components/radix/button
  examples  https://raw.githubusercontent.com/.../examples/button-example.tsx
```

一部 components は underlying library への `api` link を含みます（例: command component の `cmdk`）。

### `diff` — updates を確認する

この command は使いません。代わりに `npx shadcn@latest add --diff` を使います。

### `info` — project information

```bash
npx shadcn@latest info [options]
```

project info と `components.json` configuration を表示します。project の framework、aliases、Tailwind version、resolved paths を見つけるため、最初にこれを実行します。

| Flag          | Short | 説明 | Default |
| ------------- | ----- | ---- | ------- |
| `--cwd <cwd>` | `-c`  | Working directory | current |

**Project Info fields:**

| Field                | Type      | 意味 |
| -------------------- | --------- | ---- |
| `framework`          | `string`  | 検出された framework（`next`, `vite`, `react-router`, `start` など） |
| `frameworkVersion`   | `string`  | Framework version（例: `15.2.4`） |
| `isSrcDir`           | `boolean` | project が `src/` directory を使うか |
| `isRSC`              | `boolean` | React Server Components が有効か |
| `isTsx`              | `boolean` | TypeScript を使うか |
| `tailwindVersion`    | `string`  | `"v3"` または `"v4"` |
| `tailwindConfigFile` | `string`  | Tailwind config file path |
| `tailwindCssFile`    | `string`  | global CSS file path |
| `aliasPrefix`        | `string`  | Import alias prefix（例: `@`, `~`, `@/`） |
| `packageManager`     | `string`  | 検出された package manager（`npm`, `pnpm`, `yarn`, `bun`） |

**Components.json fields:**

| Field                | Type      | 意味 |
| -------------------- | --------- | ---- |
| `base`               | `string`  | Primitive library（`radix` または `base`）。component APIs と available props を決める |
| `style`              | `string`  | Visual style（例: `nova`, `vega`） |
| `rsc`                | `boolean` | config の RSC flag |
| `tsx`                | `boolean` | TypeScript flag |
| `tailwind.config`    | `string`  | Tailwind config path |
| `tailwind.css`       | `string`  | Global CSS path。custom CSS variables はここに置く |
| `iconLibrary`        | `string`  | Icon library。icon import package を決める（例: `lucide-react`, `@tabler/icons-react`） |
| `aliases.components` | `string`  | Component import alias（例: `@/components`） |
| `aliases.utils`      | `string`  | Utils import alias（例: `@/lib/utils`） |
| `aliases.ui`         | `string`  | UI component alias（例: `@/components/ui`） |
| `aliases.lib`        | `string`  | Lib alias（例: `@/lib`） |
| `aliases.hooks`      | `string`  | Hooks alias（例: `@/hooks`） |
| `resolvedPaths`      | `object`  | 各 alias の absolute file-system paths |
| `registries`         | `object`  | 設定済み custom registries |

**Links fields:**

`info` output には component docs、source、examples 用の templated URLs を含む **Links** section があります。resolved URLs が必要な場合は、代わりに `npx shadcn@latest docs <component>` を使います。

### `build` — custom registry を build する

```bash
npx shadcn@latest build [registry] [options]
```

配布用に `registry.json` を個別 JSON files へ build します。default input は `./registry.json`、default output は `./public/r` です。

authoring rules、`include`、item definitions、`registryDependencies`、GitHub registry behavior については [registry.md](./registry.md) を参照してください。

| Flag              | Short | 説明 | Default |
| ----------------- | ----- | ---- | ------- |
| `--output <path>` | `-o`  | Output directory | `./public/r` |
| `--cwd <cwd>`     | `-c`  | Working directory | current |

---

## テンプレート

| Value          | Framework      | Monorepo support |
| -------------- | -------------- | ---------------- |
| `next`         | Next.js        | Yes              |
| `vite`         | Vite           | Yes              |
| `start`        | TanStack Start | Yes              |
| `react-router` | React Router   | Yes              |
| `astro`        | Astro          | Yes              |
| `laravel`      | Laravel        | No               |

すべての templates は `--monorepo` flag による monorepo scaffolding をサポートします。渡された場合、CLI は monorepo-specific template directory（例: `next-monorepo`, `vite-monorepo`）を使います。`--monorepo` と `--no-monorepo` のどちらも渡されない場合、CLI は interactive prompt を表示します。Laravel は monorepo scaffolding をサポートしません。

---

## プリセット

`--preset` で preset を指定する方法は3つです。

1. **Named:** `--preset nova` または `--preset lyra`
2. **Code:** `--preset a2r6bw`（version-prefixed base62 string、例: `a2r6bw` または `b0`）
3. **URL:** `--preset "https://ui.shadcn.com/init?base=radix&style=nova&..."`

> **IMPORTANT:** preset codes を手動で decode、fetch、resolve しようとしてはいけません。preset codes は opaque です。`npx shadcn@latest init --preset <code>` に直接渡し、解決は CLI に任せます。
> 既存 project の preset を overwrite する場合は `npx shadcn@latest apply --preset <code>` を使います。

## プリセット切り替え

まずユーザーに確認します: 既存 components を **overwrite**、**merge**、または **skip** しますか？

- **Overwrite / Re-install** → `npx shadcn@latest apply --preset <code>`。検出されたすべての component files を新しい preset styles で overwrite します。ユーザーが components をカスタマイズしていない場合に使います。
- **Merge** → `npx shadcn@latest init --preset <code> --force --no-reinstall` を実行し、続けて `npx shadcn@latest info` で installed components の list を取得し、[smart merge workflow](./SKILL.md#updating-components) で1つずつ local changes を保ちながら更新します。ユーザーが components をカスタマイズしている場合に使います。
- **Skip** → `npx shadcn@latest init --preset <code> --force --no-reinstall`。config と CSS variables だけを更新し、既存 components はそのままにします。

preset commands は必ずユーザーの project directory 内で実行します。`apply` は `components.json` file を持つ既存 project でのみ動作します。CLI は `components.json` から現在の base（`base` vs `radix`）を自動的に保持します。scratch/temp directory を使う必要がある場合（例: `--dry-run` comparisons）、`--base <current-base>` を明示的に渡します。preset codes は base を encode していません。
