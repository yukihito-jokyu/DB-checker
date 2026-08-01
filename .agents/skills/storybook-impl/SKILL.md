---
name: storybook-impl
description: DB-checker の Wails v2 / React / TypeScript UI に Storybook を導入・構成し、型付き Story、描画・操作・アクセシビリティテスト、Wails 境界モック、Taskfile・CI 検証を実装・更新・診断する。Storybook 初期導入、`.storybook` や `*.stories.tsx` の作成・修正、Vitest addon・a11y 設定、Storybook build・test の失敗対応、コンポーネント状態の網羅を依頼されたときに使用する。
---

# Storybook Implementation

## 目的

DB-checker の既存設計と依存関係を尊重し、Storybook を UI の状態カタログ、開発環境、描画・操作・アクセシビリティテストの共通基盤として実装する。

## 必須リファレンス

- 作業前に必ず [references/project-patterns.md](references/project-patterns.md) を読む。
- 初期導入、設定変更、依存更新、テスト戦略の変更、または判断に迷う場合は [references/best-practices.md](references/best-practices.md) も読む。
- Storybook、Vite、Vitest、Playwright のバージョン条件は変化するため、依存関係を追加・更新する前に Storybook 公式ドキュメントの現行要件を再確認する。

## 作業手順

### 1. 現状を調査する

以下を確認し、既存変更を上書きしない。

- ルートと作業対象配下の `AGENTS.md`
- `git status --short`
- `frontend/package.json` と lockfile
- `frontend/vite.config.ts`、TypeScript 設定、Biome 設定
- `frontend/src/index.css` と UI ライブラリ設定
- `frontend/.storybook` と既存の `*.stories.*`
- `Taskfile.yml` と frontend CI
- 対象コンポーネント、その Props、状態、サービス依存、既存テスト

依頼を次のいずれかに分類する。

- Storybook の初期導入・基盤更新
- Story の追加・更新
- 描画・操作・アクセシビリティテストの追加・更新
- Storybook の不具合診断・修正

### 2. バージョン互換性を判定する

現在の Storybook 公式要件と、プロジェクトの Node.js、npm、React、TypeScript、Vite、Vitest の実バージョンを比較する。

- 最新安定版が既存の主要依存関係と互換であれば、最新安定版を選ぶ。
- 最新安定版の導入に Vite などのメジャー更新が必要な場合、影響範囲と選択肢を説明して明示的な承認を得る。
- 主要依存関係の更新先は、Storybook の最小要件を満たすラインと現行サポートラインを比較し、Node.js・CI・plugin・保守期間への影響を示す。単に最大のメジャーバージョンを選ばない。
- 承認なしに主要依存関係を更新しない。
- 互換性のためだけに古い Storybook を黙って固定しない。
- Chromatic など外部サービスへの公開・送信は、明示的な承認なしに追加・実行しない。

### 3. Storybook 基盤を構成する

初期導入では、承認済みのバージョン方針に従って Storybook 公式 CLI を `frontend` から実行する。生成差分を必ず確認し、不要なサンプル Story とサンプル asset を除く。

次を満たすよう構成する。

- React と Vite に対応する公式 framework を使用する。
- `frontend/src/index.css` を preview で読み込む。
- Vite の既存 alias と解決規則を再利用する。
- Storybook の docs、描画テスト、interaction test、アクセシビリティテストを利用可能にする。
- package scripts と `Taskfile.yml` に開発、static build、test の入口を用意する。
- CI で static build と自動テストを実行し、通常の frontend build と Biome check も維持する。
- Playwright browser の準備方法は、導入した Storybook/Vitest addon の公式手順に合わせる。
- `storybook-static` などの生成物を Git、formatter、lint の対象から除外する。

設定を推測で手書きせず、導入したバージョンが生成する設定と公式ドキュメントを基準にする。

### 4. Story の状態を設計する

実装前に、対象 UI の利用者から見える状態を列挙する。

- 通常状態
- 空状態
- 読み込み中
- 失敗と再試行
- 無効化・処理中
- 長い文言や多数項目
- 狭い viewport
- キーボード操作やフォーカス管理

すべてを機械的に作らず、見た目、振る舞い、アクセシビリティ、回帰リスクが異なる代表状態を選ぶ。

### 5. 型付き Story を実装する

- Story は対象コンポーネントの近くへ配置する。
- 既存方針がない限り、安定した CSF 3 と `Meta`、`StoryObj`、`satisfies` を使う。
- Story 固有の差分は `args` で表現する。
- 共通のレイアウトや Provider だけを decorator にする。
- callback は `fn()` で観測可能にする。
- Story 名は表示状態や利用シナリオを示す。
- DOM 構造、CSS class、内部 state に依存する assertion を避ける。
- ページ Story では必要な Provider だけを追加し、本番アプリ全体を無条件に起動しない。

### 6. 外部依存を境界でモックする

まず Props と `fn()` で依存を注入できるか確認する。Wails 呼び出しが必要な場合は、生成 binding ではなく `frontend/src/features/*/services` のサービス境界をモックする。

- `frontend/wailsjs` と `window.go` を直接モックしない。
- Wails 生成ファイルを編集しない。
- module mock は preview で静的に登録する。
- Story ごとの返却値は `beforeEach` と `mocked()` で設定する。
- 成功、空、失敗、未完了を決定的に再現する。
- 実際の HTTP 通信を再現する場合だけ MSW を検討する。
- Story 間で mock 状態を共有・漏洩させない。

Storybook 対応のために本番コンポーネントの責務や公開 API を大きく変える必要がある場合は、変更理由と影響を説明して承認を得る。

### 7. テストを実装する

すべての Story を描画テストの対象にする。重要な利用者操作は `play` 関数で検証する。

- `canvas` と role、accessible name、label を優先して要素を取得する。
- 非同期表示は `findBy*` または `waitFor` で待つ。
- `userEvent` を必ず `await` する。
- callback、表示内容、無効状態、フォーカスなど利用者から観測できる結果を検証する。
- 主要な成功経路に加え、失敗、再試行、キーボード操作をリスクに応じて検証する。
- a11y 検査は原則 `error` とする。
- 既知の一時的な違反だけ、理由と解消条件を残して `todo` とする。
- 対象外が明確な Story だけ `off` とする。

### 8. 検証する

変更範囲に応じ、Taskfile の入口を優先して次を実行する。

```sh
task frontend:check
task frontend:build
task frontend:storybook:build
task frontend:storybook:test
```

初期導入前など Taskfile の入口がまだ存在しない場合は、導入した package scripts を `frontend` で実行し、初期導入の一部として Taskfile の入口を追加する。

失敗した場合は、最初の根本原因を修正してから再実行する。環境上実行できない検証は、実行できなかった理由、未確認のリスク、利用者が実行する正確なコマンドを報告する。

## 完了条件

- 対象 UI の代表状態が Story として確認できる。
- 主要操作が利用者視点の interaction test で検証される。
- Wails 境界が決定的かつ Story 間で独立してモックされる。
- アクセシビリティ検査が有効である。
- Storybook static build とテストが成功する。
- 通常の frontend check と production build が成功する。
- 依存更新、外部サービス、未実施検証が明示される。
