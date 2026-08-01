# DB-checker E2E Project Patterns

## 現在の境界

- Wails v2の起動とDIはルートの `main.go` で組み立てる。
- Wails bindingとfrontend契約は `internal/handler/wails` に閉じ込める。
- frontendは `frontend/src/features/*/services` から生成bindingを呼ぶ。
- 接続プロファイルは `internal/config.NewDefaultStore` により実ユーザー設定領域へ保存される。
- 資格情報は `internal/repository` からOSキーチェーンへ保存される。
- MySQL/PostgreSQLのテスト用Docker ComposeとGo seedは既存のbackend結合テストが所有する。
- Storybookはservice境界をモックした表示・操作・アクセシビリティテストを所有する。

## テスト責務

| 種類 | 主な境界 | 対象 |
| --- | --- | --- |
| Storybook | React component / page + mocked service | 表示状態、入力、操作、a11y |
| Go単体 | domain / usecase / handler / repository | 分岐、変換、エラー契約 |
| Go結合 | UseCase → repository → 実テストDB | MySQL/PostgreSQL差分、DB制約 |
| E2E | Playwright → React → Wails binding → Go →制御された依存 | 重要な利用者フローと層間接続 |

E2Eへ追加する前に、より狭いテストで十分かを判定する。E2Eでは層間の接続が壊れたときに検出できる価値を優先する。

## 標準ディレクトリ

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
    └── <feature>/
        ├── <user-flow>.spec.ts
        ├── <feature>.page.ts
        └── <feature>.data.ts
```

- `<feature>` は利用者から見た機能領域とする。
- `frontend/src/features` と必ずしも1対1にしない。
- 複数featureをまたぐフローは `database-inspection` のように利用者の目的で命名する。
- 1ファイルで十分な間は不要な `*.page.ts`、`*.data.ts` を作らない。
- 共通処理へ昇格するまでfeature近傍に置く。
- setup/teardownが不要な段階では空ディレクトリを作らない。

## ファイル責務

- `playwright.config.ts`: test収集、browser、server、CI、artifact設定だけを持つ。
- `fixtures/test.ts`: 全specがimportする `test` / `expect` と共通fixtureを公開する。
- `fixtures/app-environment.ts`: E2E専用設定、資格情報、DB名前空間などの環境を生成・破棄する。
- `setup/*.setup.ts`: suite全体の高コストな準備だけを行う。
- `*.spec.ts`: 利用者フローと主要assertionを持つ。
- `*.page.ts`: 複数specで再利用するlocatorと利用者操作だけを持つ。
- `*.data.ts`: 秘密でない入力シナリオを持ち、実資格情報を埋め込まない。

## 状態と秘密情報

- `main.go` の既定DIをそのままE2Eで使用しない。
- E2E実行ごとに一時設定領域を作り、明示的な引数またはDIで渡す。
- OSキーチェーンへE2E資格情報を書かない。テスト用adapterを注入する。
- 本番DB、個人DB、共有開発DBへ接続しない。
- DB名、schema名、profile IDなどをworker/test単位で一意にする。
- パスワードやDSNをテストタイトル、step、attachment、consoleへ含めない。
- cleanup対象を明示的に解決し、広いパスや未解決変数を削除対象にしない。

## Wails v2の保証範囲

Wails dev serverのHTTP入口から開いた画面では、Wailsが注入するIPC/runtimeと生成bindingを通してGoメソッドを呼ぶ。これによりfrontendとGo backendの接続を検証する。

ブラウザとnative WebViewの差、native dialog、menu、window制御、OS統合は同じ保証に含めない。必要な場合は対象OSのplatform smoke testまたは手動検証を別に定義する。

## 既存入口との統合

- frontendの定型操作は `Taskfile.yml` の `frontend:*` を優先する。
- E2E用DBがbackend結合テストと同じ構成なら、既存ComposeとGo seedを再利用する。
- StorybookのPlaywright依存とPlaywright Test runnerを混同しない。
- 生成bindingを編集しない。Go契約変更時は `task generate` で更新する。
