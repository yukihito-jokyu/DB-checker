---
name: wails-e2e-test
description: DB-checker の Wails v2 / React / TypeScript アプリに対する Playwright E2E テストを設計・導入・実装・レビュー・診断する。`frontend/e2e`、`playwright.config.ts`、E2E fixture、feature別spec、Wails dev server、テスト用MySQL/PostgreSQL、Taskfile・CI、traceやflaky testを扱うときに使用する。
---

# Wails E2E Test

## 目的

DB-checker の重要な利用者フローを、ブラウザ操作から Wails binding、Go backend、テスト用永続化先まで通して、決定的かつ安全に検証する。

## 必須リファレンス

- 作業前に必ず [references/project-patterns.md](references/project-patterns.md) を読む。
- 初期導入、設定・依存・CI変更、テスト戦略の判断、レビュー、flaky test診断では [references/best-practices.md](references/best-practices.md) も読む。
- Playwright、Wails、Node.js の条件は変化するため、依存追加・更新や公式挙動に関する判断前に現行の公式ドキュメントを確認する。

## 作業手順

### 1. 現状と依頼を分類する

ルートと対象配下の `AGENTS.md`、`git status --short`、`Taskfile.yml`、frontend CI、`frontend/package.json`、`wails.json`、対象画面、Wails service、handler、usecase、repository、既存テストを確認する。既存変更を上書きしない。

依頼を次のいずれかへ分類する。

- Playwright E2E 基盤の初期導入・更新
- feature の利用者フロー追加・更新
- E2E テストまたは設定のレビュー
- 失敗、flaky、起動・cleanup不良の診断

### 2. E2Eに含める責務を決める

テスト対象を内部実装ではなく、利用者の目的と観測可能な結果で定義する。

- E2E: 重要な成功経路と、層をまたぐことで初めて保証できる重大な失敗経路
- Storybook: serviceをモックした表示状態、入力検証、操作、アクセシビリティ
- Go単体テスト: domain、usecase、handlerの分岐とエラー契約
- Go結合テスト: UseCaseから実MySQL/PostgreSQLまでのDB組み合わせ

StorybookやGoテストで十分なケースをE2Eへ重複追加しない。すべての入力値やエラー分岐をE2Eで網羅しない。

### 3. 安全ゲートを確認する

状態を変更するテストの実装・実行前に、次をすべて確認する。

- 開発者の設定ファイルではなく、E2E専用の一時設定領域を使う。
- OSキーチェーンではなく、隔離されたテスト用資格情報ストアを使う。
- Docker Composeで起動したテスト専用MySQL/PostgreSQLだけを使う。
- テストまたはworkerごとに一意なデータを作り、必ずcleanupする。
- パスワード、DSN、資格情報をログ、trace、スクリーンショット、reportへ残さない。

差し替え口がない場合、実ユーザー環境を使って続行しない。起動部分のDI、設定パス、資格情報adapterなど、最小のテスト境界を先に設計する。依頼範囲を超える変更が必要なら、理由と影響を説明して承認を得る。

### 4. ディレクトリを構成する

Playwright設定を `frontend/playwright.config.ts`、E2E本体を `frontend/e2e` に置く。

```text
frontend/
├── playwright.config.ts
└── e2e/
    ├── fixtures/
    │   ├── test.ts
    │   └── app-environment.ts
    ├── setup/
    │   ├── environment.setup.ts
    │   └── environment.teardown.ts
    └── connection-profiles/
        ├── connection-profiles.spec.ts
        ├── connection-profiles.page.ts
        └── connection-profiles.data.ts
```

- `fixtures/test.ts` を各specが使う `test` と `expect` の入口にする。
- suite全体で一度だけ必要な高コスト処理だけを `setup` に置く。
- featureディレクトリを利用者から見た機能領域ごとに増やす。
- `frontend/src/features` を機械的に複製せず、複数featureをまたぐ場合は利用者の目的で命名する。
- 1つのspec専用処理はspec内に置き、複数シナリオで再利用するときだけ `*.page.ts` や `*.data.ts` を作る。
- 共通 `pages`、`helpers`、`utils` へ無秩序に集約しない。
- Go側のE2E専用コードが本当に必要な場合だけ `test/e2e` を追加し、業務ロジックを置かず、本番コードからimportしない。

`test-results`、`playwright-report`、一時設定、資格情報をGit管理しない。visual regressionを明示的に導入する場合だけ、承認済みsnapshotをspec近傍で管理する。

### 5. Playwright基盤を構成する

公式手順と既存バージョンの互換性を確認して `@playwright/test` を導入する。Storybook経由の `playwright` が存在するだけでPlaywright Test導入済みと判断しない。

`playwright.config.ts` では少なくとも次を明示する。

- `testDir: "./e2e"`
- 固定したE2E用Wails dev server URL
- CIの `forbidOnly`、worker数、retry、`failOnFlakyTests`
- 最初のretryで収集するtrace
- test artifactとHTML reportの出力先
- 必要最小限のChromium project

アプリ起動はPlaywrightの `webServer` またはTaskfileの明示的な入口から行い、ready状態をURLで確認する。ローカルで既存serverを再利用する場合とCIで新規起動する場合を区別する。依存関係を無断でメジャー更新しない。

suite共通のDB準備・破棄には、単独の `globalSetup` よりPlaywright project dependenciesを優先する。各テストの可変状態はfixtureで作成・破棄し、suite setupへ混ぜない。

### 6. Wails境界を通す

Wails v2 dev server経由で画面を開き、生成bindingを通して実handlerを呼ぶ。E2Eでは `frontend/src/features/*/services`、`frontend/wailsjs`、`window.go` をモックしない。

テスト用repository adapterや制御された外部依存は起動時DIで差し替えてよいが、テスト対象の層をモックした場合はE2Eと呼ばず責務を明示する。ブラウザから保証できないnative dialog、menu、window、WebView固有差分は別のplatform smoke testまたは手動検証として扱う。

### 7. 利用者フローを実装する

- テスト名を利用者の行動と期待結果で記載する。
- role、label、accessible name、表示テキストを優先して取得する。
- semantic locatorで安定して表現できない契約だけ `data-testid` を使う。
- Locatorとweb-first assertionを使い、actionとassertionを必ず `await` する。
- `waitForTimeout`、固定sleep、CSS class、DOM階層、XPathへ依存しない。
- 操作後は利用者から見える表示、フォーカス、無効状態、保存結果などを検証する。
- `test.step` で長いフローの意味を分けるが、1テストへ多数の独立シナリオを詰め込まない。
- 条件分岐でassertionを回避せず、前提が異なるケースを別テストにする。

Page Objectは操作語彙とlocatorの再利用に限定する。業務上重要なassertionをPage Objectへ隠さずspecに残す。

### 8. 隔離と診断可能性を実装する

fixtureでsetupとteardownを同じ場所に保ち、テストを単独・任意順・繰り返し実行できるようにする。共有DB状態を変更する間はCIを1 workerから始め、分離が証明できてから並列化する。

retryは原因調査用artifactを得るためにCIだけで使用し、flaky成功を合格扱いしない。失敗時はtrace、console、Wails/Goログ、DB種別、シナリオ名から最初の根本原因を特定する。固定待機やretry増加だけで症状を隠さない。

### 9. TaskfileとCIを接続する

定型操作はpackage scriptだけで閉じず、Taskfileから実行できるようにする。

- Playwright単体入口として `frontend:e2e` を用意する。
- DBやWails起動を含む場合、`e2e:up`、`e2e:test`、`e2e:down` など状態変更とcleanupを明示する。
- 既存の `integration:up` / `integration:down` を安全に再利用できる場合、DB定義を複製しない。
- CIでは依存install、Chromium準備、E2E実行、失敗artifact uploadを明示する。
- 外部サービスや実ユーザー資格情報を要求しない。

### 10. 検証する

変更範囲に応じ、Taskfileの入口を優先して次を実行する。

```sh
task frontend:check
task frontend:build
task frontend:storybook:test
task test
task integration:test
task frontend:e2e
```

未導入の入口は実行せず、初期導入の一部として追加する。DB起動が必要なら事前に対応するup taskを実行し、終了時にdown taskでcleanupする。

環境上実行できない検証は、理由、未確認リスク、利用者が実行する正確なコマンドを報告する。テスト成功だけでなく、実ユーザー設定・資格情報・DBを変更していないことも確認する。

## 完了条件

- E2Eの責務がStorybook、Go単体・結合テストと分離されている。
- feature単位のディレクトリとfixture境界が規約に従っている。
- ブラウザ操作から対象のWails/Go境界まで実際に通っている。
- テストデータ、設定、資格情報が独立し、cleanupされる。
- semantic locatorとweb-first assertionを使用している。
- CIの失敗artifactから原因を追跡できる。
- 対象のTaskfile検証が成功し、未実施項目と残存リスクが報告されている。
