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

特定の feature に依存しない共通コンポーネントを配置する。

例:

- `Loading`
- `EmptyState`
- `ErrorMessage`

## pages

画面単位のコンポーネントを配置する。React Router の route に対応する単位として扱う。

page 専用の hook が必要になった場合は、対象 page の近くに置く。

```txt
pages/
  home/
    HomePage.tsx
    useHomePage.ts
```

## features

機能単位の UI、hook、型、service を配置する。土台作りの段階では具体的な feature が未確定のため、空ディレクトリとして管理する。

将来的に機能を追加する場合の例:

```txt
features/
  user/
    components/
    hooks/
    types.ts
    service.ts
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

## 状態管理

初期段階では Zustand store を作成しない。状態管理は以下の優先順位で行う。

1. component 内で閉じる状態は `useState`
2. 複雑な component 状態は `useReducer`
3. 複数 component / page で共有する状態が出た場合のみ Zustand を導入する

## Wails binding

Wails binding の扱いは Wails 公式推奨に従う。外部 HTTP 通信は行わないため、`src/api` は作成しない。

Wails binding を扱うためだけに `api` という名前のディレクトリを作らない。
