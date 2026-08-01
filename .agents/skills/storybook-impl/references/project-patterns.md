# DB-checker 向け Storybook 実装規約

## 目次

- [作業前の再確認](#作業前の再確認)
- [配置と命名](#配置と命名)
- [global 設定](#global-設定)
- [Wails 境界のモック](#wails-境界のモック)
- [コンポーネント別の方針](#コンポーネント別の方針)
- [コマンドと CI](#コマンドと-ci)

## 作業前の再確認

この資料は 2026-07-28 時点の構成を基にする。作業時には必ず実ファイルを再確認する。

- frontend: Wails v2 標準構成、React、TypeScript、Vite
- styling: Tailwind CSS、shadcn/ui、`frontend/src/index.css`
- alias: `@` は `frontend/src`、`@wails` は生成 binding
- Wails 呼び出し: `frontend/src/features/*/services` へ集約
- frontend 検証: Biome、Vite production build
- 定型コマンド: ルートの `Taskfile.yml`

Storybook 初期導入時点で現行版が Vite 5 以上を要求し、プロジェクトが Vite 3 系のままであれば、先に Vite 更新の影響を調査して利用者の承認を得る。古い Storybook の暗黙固定で回避しない。更新先は最小互換ラインと現行サポートラインを比較し、Node.js 20 の細かな version、`@vitejs/plugin-react`、CI への影響を示す。

## 配置と命名

Story は対象実装と同じディレクトリへ置く。

| 対象 | Story の例 | `title` の例 |
| --- | --- | --- |
| 共通 UI | `frontend/src/components/ui/Button.stories.tsx` | `Components/UI/Button` |
| feature component | `frontend/src/features/connection-profiles/components/ProfileList.stories.tsx` | `Features/Connection Profiles/ProfileList` |
| page | `frontend/src/pages/connection-profiles/ConnectionProfilesPage.stories.tsx` | `Pages/Connection Profiles` |

- Story export 名は `Default`、`Loading`、`Empty`、`LoadFailure`、`Saving` のように状態を UpperCamelCase で示す。
- Story の階層は物理配置と機能境界を反映する。
- CSF 3 と `Meta`、`StoryObj`、`satisfies` を既定とする。
- プロジェクトが明示的に CSF Next を採用した場合だけ、その方針へ統一する。

## global 設定

`frontend/.storybook/preview.ts` では次を構成する。

- `../src/index.css` を読み込む。
- document の言語を日本語として扱う。
- a11y test を原則 `error` にする。
- 部品の既定 layout は `padded` にする。
- ページ Story は meta または Story 単位で `fullscreen` にする。
- 全 Story に不要な Provider やアプリ初期化処理を追加しない。

Vite 設定の alias と plugin を再利用する。Storybook 側に同じ alias を別値で複製しない。

## Wails 境界のモック

Wails 生成 binding ではなく、feature の service module をモックする。現在の代表境界は次である。

```text
frontend/src/features/connection-profiles/services/connectionProfiles.ts
```

module mock は preview で静的に登録する。

```ts
import { sb } from "storybook/test";

sb.mock(
  import(
    "../src/features/connection-profiles/services/connectionProfiles.ts"
  ),
);
```

`sb.mock` の import は preview からの相対パス、拡張子付きとし、`@` alias を使わない。

Story ごとの結果は `beforeEach` で設定する。

```tsx
import type { Meta, StoryObj } from "@storybook/react-vite";
import { mocked } from "storybook/test";

import { listConnectionProfiles } from "@/features/connection-profiles/services/connectionProfiles";
import { ConnectionProfilesPage } from "./ConnectionProfilesPage";

const meta = {
  component: ConnectionProfilesPage,
  parameters: {
    layout: "fullscreen",
  },
  beforeEach: async () => {
    mocked(listConnectionProfiles).mockResolvedValue({
      profiles: [],
    });
  },
} satisfies Meta<typeof ConnectionProfilesPage>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Empty: Story = {};
```

実際の関数名、戻り値型、export 形式は作業時の service 実装へ合わせる。成功、空、失敗、未完了を各 Story の `beforeEach` で決定的に設定し、Story 間で mock 実装を共有しない。

次を禁止する。

- `frontend/wailsjs` の編集
- `window.go` の直接 stub
- Story から実際の DB 接続や Wails runtime を起動
- import 時副作用を持つ module の無検討な自動 mock

## コンポーネント別の方針

### Presentational component

- Props は `args` で渡す。
- callback は `fn()` を使う。
- variant、disabled、長い文言、狭い viewport を必要に応じて分ける。

### Feature component

- feature の domain type と service interface をそのまま利用する。
- loading、empty、success、failure、操作中の代表状態を作る。
- fetch や mutation の結果は service 境界で制御する。
- toast が結果確認に必要な Story だけ Toaster を decorator で追加する。

### Dialog

- 利用者が trigger から開く経路を優先して Story 化する。
- 初期フォーカス、Tab 移動、Escape、閉じた後のフォーカス復帰を確認する。
- 保存中の多重送信防止と、保存失敗後の再操作を確認する。
- DOM node や React ref を Story から直接操作して通常経路を飛ばさない。

### Page

- `layout: "fullscreen"` を指定する。
- 初期読み込み、成功、空、失敗、再試行を分ける。
- Router や Provider はページが実際に必要とする最小構成だけ追加する。
- ページ配下の feature service を境界として Wails を隔離する。

## コマンドと CI

初期導入では、導入バージョンが生成する script 名を確認した上で、少なくとも次の目的を package scripts に持たせる。

- Storybook 開発サーバー
- Storybook static build
- Storybook browser test

ルート `Taskfile.yml` には次の入口を用意する。

```text
frontend:storybook
frontend:storybook:build
frontend:storybook:test
```

CI では以下を維持・追加する。

- Biome check
- frontend production build
- Storybook static build
- Storybook browser test
- `storybook-static` を Git と Biome の対象から除外
- 導入バージョンに合った Playwright browser の準備

workflow が `paths` を使用する場合、CI で参照する `Taskfile.yml` や Storybook 設定の変更でも起動することを確認する。

外部 visual regression は既定に含めない。Chromatic 等を追加する場合は、送信対象、secret、費用、PR check の扱いについて利用者の承認を得る。
