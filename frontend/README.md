# Frontend

## 開発コマンド

frontend の依存関係をインストールする。

```bash
cd frontend
npm ci
```

Biome check を実行する。

```bash
npm run check
```

frontend build を実行する。

```bash
npm run build
```

Taskfile を使う場合は以下でも実行できる。

```bash
task frontend:check
task frontend:build
```

## CI

GitHub Actions の `Frontend CI` は Pull Request で `frontend/**` または `.github/workflows/frontend.yml` が変更された場合のみ実行される。

CI では `frontend` ディレクトリで `npm ci` を実行したあと、`npm run check` による Biome check と `npm run build` による frontend build を実行する。

## 構成方針

- Wails アプリの frontend は React と TypeScript で実装する。
- 画面遷移には React Router を使い、ルーティング定義は `src/app/router.tsx` に集約する。
- 初期状態では `createHashRouter` で `/` のみを定義する。
- shadcn/ui は必要になったコンポーネントだけを追加する。初期導入は `button` のみとする。
- `@` エイリアスは `frontend/src` を指す。Wails 生成物は `frontend/wailsjs` の公式生成パスから直接 import する。
- Wails 生成物の import は、疎通確認を除き service または runtime adapter の境界に閉じ込める。

## components/ui

`components/ui` には shadcn/ui が生成する低レベル UI コンポーネントを配置する。

このディレクトリには以下を入れない。

- Wails binding 呼び出し
- Zustand store 参照
- アプリ固有の業務ロジック
- page 固有の処理
- feature 固有の処理

`components/ui` は見た目と基本的な UI 振る舞いに責務を限定する。

## app

アプリ全体の組み立てに関係する定義を配置する。

- `router.tsx`: React Router の route 定義を集約する。`src/main.tsx` は `RouterProvider` へ router を渡すエントリポイントに限定する。

## components/layout

アプリ全体のレイアウトに関係するコンポーネントを配置する。

例:

- `AppShell`
- `Header`
- `Sidebar`
- `MainLayout`

## components/common

特定の feature に依存しない共通コンポーネントを配置する。業務ドメインの語彙や Wails service に近い UI は置かない。

例:

- `Loading`
- `EmptyState`
- `ErrorMessage`
- 汎用確認ダイアログ

## pages

画面単位のコンポーネントを配置する。React Router の route に対応する単位として扱う。

page は複数 feature の配置、URL、読み込み状態、画面固有 hook を扱う。page 専用の hook や service が必要になった場合は、対象 page の近くに置く。

```txt
pages/
  home/
    HomePage.tsx
    useHomePage.ts
    services/
```

## features

機能単位の UI、hook、型、service、store を配置する。DB 接続、スキーマ検査、検査結果など、業務語彙や Wails service に近い UI は `components/common` ではなく feature に置く。

将来的に機能を追加する場合の例:

```txt
features/
  user/
    components/
    hooks/
    services/
    stores/
    types.ts
```

## hooks

`src/hooks` にはアプリ全体で再利用する custom hook を配置する。

- 特定の component でしか使わない hook は対象 component の近くに置く。
- 特定の page でしか使わない hook は対象 page の近くに置く。
- 特定の feature 内で再利用する hook は対象 feature の `hooks` に置く。

配置例:

```txt
src/hooks/useKeyboardShortcut.ts
src/pages/home/useHomePage.ts
src/features/user/hooks/useUsers.ts
src/features/user/components/UserForm/useUserForm.ts
```

## lib

アプリ全体で使う utility を配置する。shadcn/ui で利用する `cn()` は `lib/utils.ts` に配置する。

Wails runtime API の薄い adapter は `src/lib/wails/*.ts` に置く。ここには業務 API 呼び出しを置かない。

例:

- `src/lib/wails/events.ts`: `EventsOn` / `EventsOff` の購読、解除 helper
- `src/lib/wails/window.ts`: window 操作 wrapper
- `src/lib/wails/clipboard.ts`: clipboard 操作 wrapper
- `src/lib/wails/browser.ts`: 外部 URL オープン wrapper
- `src/lib/wails/logger.ts`: runtime log wrapper

## 状態管理

Zustand store は `src/stores/appStore.ts` にアプリ共有状態の基盤だけを置く。画面固有の状態は store に寄せず、以下の優先順位で扱う。

1. component 内で閉じる状態は `useState`
2. 複雑だが局所的な状態は component / page / feature 近傍の `useReducer` または custom hook
3. feature 内で共有する状態は `src/features/<feature>/stores/*.ts`
4. 複数 feature / page をまたぐアプリ共有状態は `src/stores/*.ts`

Zustand の実データ管理は、複数 feature / page で共有する状態が出た時点で追加する。feature 内だけで共有する状態は `src/features/<feature>/stores/*.ts` に置き、`src/stores` へ集約しすぎない。

## Wails binding

Wails binding の扱いは Wails 公式推奨に従う。外部 HTTP 通信は行わないため、`src/api` は作成しない。

Wails binding を扱うためだけに `api` という名前のディレクトリを作らない。

Wails binding 呼び出しは薄い service 境界に閉じ込める。

- feature に属する Wails binding 呼び出し: `src/features/<feature>/services/*.ts`
- page 固有で feature 化しない呼び出し: `src/pages/<page>/services/*.ts`
- app 全体の初期化、疎通、バージョン取得: `src/app/services/*.ts`
- Wails runtime API の adapter: `src/lib/wails/*.ts`

`frontend/wailsjs` の生成物は原則として上記の境界からのみ import する。既存の初期疎通確認は例外として扱い、機能実装または対象 page 改修時に service 境界へ移す。

## test helper

test helper は利用範囲に近い場所へ置く。

- feature 固有の test helper: `src/features/<feature>/test-utils/*.ts`
- page 固有の test helper: `src/pages/<page>/test-utils/*.ts`
- 複数領域で再利用する test helper: `src/test-utils/*.ts`
- Playwright など E2E 専用 helper: `frontend/e2e` または将来作る E2E ルート配下

本番コードから `test-utils` を import しない。
