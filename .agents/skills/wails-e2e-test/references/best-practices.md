# E2E Best Practices

調査日: 2026-08-01

## テスト戦略

- E2Eを少数の重要な利用者フローへ限定し、単体・結合テストとのピラミッドを維持する。
- 利用者から観測できる振る舞いを検証し、関数名、component state、CSS classなどの実装詳細を検証しない。
- 自分たちが制御できない第三者サービスへ依存せず、テストデータと外部状態を制御する。

参照:

- [Playwright Best Practices](https://playwright.dev/docs/best-practices)
- [Google Testing Blog: Just Say No to More End-to-End Tests](https://testing.googleblog.com/2015/04/just-say-no-to-more-end-to-end-tests.html)

## Locatorと待機

- role、label、text、test idの順に、利用者向け属性と明示的な契約を優先する。
- Locatorのstrictnessを `first()` や `nth()` で安易に回避せず、一意な意味を持つlocatorへ直す。
- auto-waitingと再試行可能なweb-first assertionを使う。
- `isVisible()` の即時値を通常のexpectへ渡したり、固定sleepで待ったりしない。

参照:

- [Playwright Locators](https://playwright.dev/docs/locators)
- [Playwright Assertions](https://playwright.dev/docs/test-assertions)

## 隔離とfixture

- 各テストを単独・任意順で実行可能にする。
- setupとteardownをfixtureの `use()` 前後へまとめ、必要なテストだけで遅延実行する。
- worker共有状態を変更する場合はworkerごとに一意なデータを割り当てる。
- suite全体の準備はPlaywright project dependenciesを優先し、各テストの可変状態をglobal setupへ置かない。

参照:

- [Playwright Fixtures](https://playwright.dev/docs/test-fixtures)
- [Playwright Global setup and teardown](https://playwright.dev/docs/test-global-setup-teardown)
- [Playwright Isolation](https://playwright.dev/docs/browser-contexts)

## CIとflaky test

- PRごとに実行し、CIでは最初は1 workerを使用して再現性を優先する。
- 必要なbrowserだけをinstallする。
- retryはCIだけで有効にし、最初のretryでtraceを収集する。
- `failOnFlakyTests` を使い、retryで成功したテストを安定した合格として扱わない。
- `forbidOnly` で `test.only` の混入を失敗させる。
- traceを常時有効にせず、失敗調査に必要な範囲で収集する。

参照:

- [Playwright Continuous Integration](https://playwright.dev/docs/ci)
- [Playwright Test configuration](https://playwright.dev/docs/test-configuration)
- [Playwright Retries](https://playwright.dev/docs/test-retries)
- [Playwright Trace viewer](https://playwright.dev/docs/trace-viewer-intro)

## Wails v2

- `wails dev` のdev serverはfrontendだけでなくアプリをHTTPで提供し、Wails IPC/runtimeを注入する。
- `devServer` の既定値に依存し続けず、E2E設定では衝突しない固定アドレスを明示する。
- `frontend:dev:serverUrl: auto` はVite出力からfrontend dev server URLを推論する設定であり、Playwrightが開くWails dev server URLと混同しない。
- ブラウザ実行がnative WebView・OS統合まで保証するとはみなさない。

参照:

- [Wails v2 CLI](https://wails.io/docs/reference/cli/)
- [Wails Project Config](https://wails.io/docs/reference/project-config)
- [Wails Application Development](https://wails.io/docs/guides/application-development/)
- [Wails Frontend Script Injection](https://wails.io/docs/guides/frontend/)
