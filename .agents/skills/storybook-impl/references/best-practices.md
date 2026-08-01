# Storybook ベストプラクティス

## 目次

- [前提](#前提)
- [導入と構成](#導入と構成)
- [Story の設計](#story-の設計)
- [テスト戦略](#テスト戦略)
- [モック](#モック)
- [アクセシビリティと viewport](#アクセシビリティと-viewport)
- [CI と外部サービス](#ci-と外部サービス)
- [避けること](#避けること)
- [公式資料](#公式資料)

## 前提

この資料は 2026-07-28 に Storybook 公式ドキュメントを調査して整理した。バージョン条件、CLI、addon、設定 API は変化するため、導入・更新時は必ず公式資料を再確認する。

調査時点の現行 Storybook は Node.js 20 以上、npm 10 以上、TypeScript 4.9 以上、Vite 5 以上、Vitest 3 以上などを要件としている。プロジェクトが満たさない場合は、主要依存関係を更新する案と Storybook 導入を分け、利用者の承認を得る。更新先は、最小要件を満たすラインと現行サポートラインの保守期間、Node.js 要件、plugin 互換性を比較して選び、最大のメジャーバージョンを自動的に選ばない。

## 導入と構成

- 公式 CLI で既存の React/Vite プロジェクトを検出・初期化する。
- React/Vite framework と、導入バージョンが推奨する test addon を使用する。
- preview でアプリの global CSS、テーマ、locale、必要最小限の Provider を再現する。
- Vite alias や plugin を重複定義せず、アプリの設定を再利用する。
- 生成されたサンプル Story と asset は、プロジェクトの実例へ置き換えた後に削除する。
- Storybook の static build を成功条件に含める。開発サーバーが起動するだけでは完了としない。
- `storybook-static` などの生成物を Git、formatter、lint の対象から除外する。

## Story の設計

Story はコンポーネントの利用者から見える代表状態を、一つずつ独立して再現する。

- 対象コンポーネントの近くに `*.stories.tsx` を配置する。
- 既存方針がない場合は、安定した CSF 3 を使う。
- `Meta<typeof Component>`、`StoryObj<typeof meta>`、`satisfies` で型安全性を保つ。
- データと callback は `args` で渡す。
- callback は `fn()` にして interaction test から観測する。
- decorator は Provider、レイアウト、テーマなど複数 Story に共通の環境へ限定する。
- `title` はプロジェクトの機能階層を示し、Story 名は `Loading`、`Empty`、`SaveFailure` のように状態を示す。
- Autodocs と controls は Props の理解を助ける場合に使い、手書きの説明と実装を重複させない。

代表状態の目安:

- 基本 UI: variant、size、disabled、長い文言、狭い幅
- 入力 UI: 初期値、validation、送信中、成功、失敗
- 一覧 UI: 通常、空、多数項目、読み込み中、取得失敗
- dialog: open、保存中、保存失敗、Escape、フォーカス復帰
- page: 初期読み込み、成功、空、失敗、再試行、画面遷移

状態の直積をすべて作らない。見た目、操作、アクセシビリティ、回帰リスクが同じ組み合わせは代表例へまとめる。

## テスト戦略

Story をテストケースの共通形式として扱い、段階的に検証する。

1. すべての Story が例外なく描画されることを確認する。
2. 重要な操作を Story の `play` で再現する。
3. test addon から browser mode で Story と `play` を実行する。
4. a11y addon で各 Story の自動検査を実行する。
5. 必要な場合だけ visual regression を追加する。

interaction test では次を守る。

- role、accessible name、label、表示テキストで要素を探す。
- Story の描画範囲には `canvas` を使う。
- popup や portal など範囲外の要素だけ必要に応じて `screen` を使う。
- 非同期表示には `findBy*` または `waitFor` を使う。
- `userEvent` と step を `await` する。
- callback 呼び出し、表示の変化、disabled、focus など観測可能な結果を検証する。
- class 名、DOM の深さ、React の内部 state、実装関数の呼び出し順を検証しない。

## モック

依存の近い境界から順に選ぶ。

1. Props と `args`
2. callback の `fn()`
3. module mock
4. HTTP 通信に対する MSW

module mock は `sb.mock(import("..."))` を `.storybook/preview` で静的に登録する。登録パスは preview からの相対パス、完全な拡張子付きとし、alias を使用しない。Story または meta の `beforeEach` で `mocked()` を使い、Story ごとの挙動を設定する。

完全に mock する module でも元 module と依存 module は評価され得る。import 時副作用を持つ module や server-only module は直接 mock せず、専用の mock file または subpath import を検討する。

## アクセシビリティと viewport

- a11y 検査は原則 `parameters.a11y.test = "error"` とする。
- 一時的な既知違反だけ `todo` とし、理由と解消条件を管理する。
- 意図的に対象外の Story だけ `off` とする。
- 自動検査だけで十分と考えず、キーボード操作、フォーカス順、accessible name、dialog のフォーカス管理を interaction test でも確認する。
- 狭い viewport、長い文言、一覧の増加を代表 Story で確認する。
- ページは `layout: "fullscreen"`、部品は原則 `padded` を使い分ける。

## CI と外部サービス

- Storybook static build と browser test を CI で実行する。
- Playwright browser の install/cache は、導入バージョンの公式手順へ合わせる。
- frontend production build、型・lint・format 検査を Storybook 導入後も維持する。
- Chromatic などの visual regression サービスは有用だが、外部送信、アカウント、token、費用が関係するため明示的な承認後に導入する。

## 避けること

- 最新要件を確認せずに Storybook のバージョンを固定する。
- Storybook 導入に便乗して Vite などを無断でメジャー更新する。
- 互換性と保守期間を比較せず、関連依存を一律に最新メジャーへ更新する。
- 本番 API や Wails binding を Story から実行する。
- 生成された Wails binding を Storybook 用に編集する。
- 一つの Story 内で多数の状態を切り替えてテストケースを曖昧にする。
- decorator へ Story 固有ロジックを詰め込む。
- `setTimeout` や任意の sleep で非同期処理を待つ。
- CSS class や DOM 構造を assertion に使う。
- a11y 違反を理由なしで `todo` または `off` にする。
- 承認なしに UI を外部サービスへ公開する。

## 公式資料

- [Install Storybook](https://storybook.js.org/docs/get-started/install)
- [Storybook for React with Vite](https://storybook.js.org/docs/get-started/frameworks/react-vite)
- [How to write stories](https://storybook.js.org/docs/writing-stories)
- [TypeScript stories](https://storybook.js.org/docs/writing-stories/typescript)
- [Args](https://storybook.js.org/docs/writing-stories/args)
- [Decorators](https://storybook.js.org/docs/writing-stories/decorators)
- [Naming and hierarchy](https://storybook.js.org/docs/writing-stories/naming-components-and-hierarchy)
- [Tags](https://storybook.js.org/docs/writing-stories/tags)
- [Mocking modules](https://storybook.js.org/docs/writing-stories/mocking-data-and-modules/mocking-modules)
- [Mocking network requests](https://storybook.js.org/docs/writing-stories/mocking-data-and-modules/mocking-network-requests)
- [UI testing overview](https://storybook.js.org/docs/writing-tests/index)
- [Interaction tests](https://storybook.js.org/docs/writing-tests/interaction-testing)
- [Vitest addon](https://storybook.js.org/docs/writing-tests/integrations/vitest-addon/index)
- [Accessibility tests](https://storybook.js.org/docs/writing-tests/accessibility-testing)
- [Visual tests](https://storybook.js.org/docs/writing-tests/visual-testing)
- [Viewport](https://storybook.js.org/docs/essentials/viewport)
- [Parameters](https://storybook.js.org/docs/api/parameters)
