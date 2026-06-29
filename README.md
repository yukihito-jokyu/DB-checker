# DB-checker
ローカルDBにダミーデータを投入し、DB設計の有効性をUI上で検証できるようにするツール

## 開発コマンド

開発・検証には Taskfile を使用する。

| コマンド | 説明 |
| --- | --- |
| `task install` | `frontend` 配下で `npm install` を実行し、frontend の依存関係をインストールする。 |
| `task generate` | Wails の frontend binding を生成し、Go と frontend の連携コードを更新する。 |
| `task dev` | Wails 開発サーバーを起動し、アプリを開発モードで実行する。 |
| `task test` | Go のテストを `go test ./...` で実行する。 |
| `task format` | Go ファイルを `gofmt` で整形する。 |
| `task format:check` | Go ファイルが `gofmt` 済みか確認し、未整形ファイルがあれば失敗する。 |
| `task lint` | `golangci-lint` で Go コードの lint を実行する。 |
| `task backend:check` | `format:check`、`lint`、`test` をまとめて実行する。 |
| `task integration:up` | 結合テスト用の MySQL / PostgreSQL を Docker Compose で起動する。 |
| `task integration:test` | 実 DB を使う Go 結合テストを `go test -tags=integration ./...` で実行する。 |
| `task integration:down` | 結合テスト用 DB を停止する。 |
| `task frontend:build` | `frontend` 配下で production build を実行する。 |
| `task frontend:check` | `frontend` 配下で Biome check を実行する。 |
| `task frontend:lint` | `frontend` 配下で lint を実行する。 |
| `task frontend:format` | `frontend` 配下で format を実行する。 |

`task lint` には `golangci-lint` が必要。CI では `golangci-lint` を `v2.12.2` に固定して実行する。

## バックエンド結合テスト

バックエンド結合テストは、UseCase の public method を API 境界として、呼び出し元から見える振る舞いを MySQL / PostgreSQL の実 DB で検証する。Wails handler は原則として結合テストに含めず、frontend 契約または handler 単体テストで扱う。

結合テスト本体は対象 UseCase と同じ package に `*_integration_test.go` として置き、`//go:build integration` を付ける。DB 起動後の接続、スキーマ作成、seed 投入、クリーンアップなどの共通処理は `test/integration` 配下に置く。

実行手順:

```bash
task integration:up
task integration:test
task integration:down
```

通常の `task test` には実 DB を使う結合テストを含めない。
