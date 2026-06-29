---
name: backend-integration-test
description: DB-checker の Go backend 結合テストを設計・実装・レビューするときに使用する。UseCase public method を API 境界とし、Docker Compose で起動した MySQL/PostgreSQL 実DBへ Go 製 seed を投入して、呼び出し元から見える全パターンを検証するための配置、build tag、Taskfile、テストデータ、DB差分ルールを扱う。
---

# Backend Integration Test

## 基本方針

- 結合テストの API 境界は `internal/usecase` の public method とする。
- Wails handler は DB 実データを使う結合テストに原則含めない。frontend 契約テストまたは handler 単体テストで扱う。
- 呼び出し元から見える成功、失敗、境界値、DB差分を検証する。
- repository 起点の結合テストは原則作らない。UseCase から観測できないが DB 対応の保証に必須な場合だけ、理由をコメントで明示して例外扱いにする。
- すでに単体テストで検証済みの純粋ロジックを結合テストで重複して細かく検証しない。

## 実データと DB

- 実データとは、本番DBや開発者個人のDBではなく、テスト用 MySQL/PostgreSQL に投入する seed データを指す。
- seed はリポジトリで管理し、実際の利用パターンに近いスキーマ、制約、レコードを含める。
- スキーマ作成、seed 投入、クリーンアップは Go コードで実装する。場当たり的な SQL ファイル追加を避ける。
- 結合テスト用 DB は Docker Compose で起動する。
- 通常の `task test` には結合テストを含めない。明示的な結合テスト用 task で実行する。

## 配置

結合テスト本体は対象 UseCase と同じ package に置く。

```text
internal/usecase/
  inspect_schema.go
  inspect_schema_test.go
  inspect_schema_integration_test.go
```

共通セットアップは `test/integration` 配下に置く。

```text
test/
  integration/
    db/
      mysql.go
      postgres.go
      setup.go
    seed/
      schema.go
      data.go
      scenario.go
```

- `test/integration` には DB 接続、スキーマ作成、seed 投入、クリーンアップ、シナリオ定義を置く。
- 共通シナリオは DB 非依存にする。
- DDL、接続文字列、方言差分だけ `db/mysql.go`、`db/postgres.go` などに分ける。
- 本番コードから `test/integration` を import しない。

## Build Tag と実行

結合テスト本体には build tag を付ける。

```go
//go:build integration
```

Taskfile には少なくとも次の責務を分けて用意する。

```text
integration:up    Docker Compose で MySQL/PostgreSQL を起動する
integration:test  go test -tags=integration ./... を実行する
integration:down  Docker Compose の DB を停止する
```

接続先は環境変数で受け取る。テストコードにローカル個人環境の接続情報を埋め込まない。

## MySQL/PostgreSQL の扱い

- 原則として、すべての UseCase 結合テストは MySQL/PostgreSQL の両方で同じケースを実行する。
- 入力 DDL や内部 SQL は DB ごとに分かれてよい。
- UseCase の戻り値、エラー分類、観測可能な振る舞いは同じ期待値に揃える。
- DB 固有差分を仕様として認める場合だけ、テスト名とコメントで明示する。

## パターン定義

「全パターン」は、内部 SQL 分岐ではなく API 利用者から見える振る舞いで定義する。

検討軸:

- DB 種別: MySQL / PostgreSQL
- UseCase public method
- 正常系
- 異常系
- 境界値
- 代表的なデータ型
- 制約
- リレーション
- タイムアウトやキャンセルなど、呼び出し元が観測できる実行制御

新しい UseCase や DB 対応を追加するときは、結合テストのパターン表も更新する。詳細方針をドキュメント化する場合は `docs/testing/backend-integration.md` を優先し、issue には要約を残す。

## 実装時の注意

- 先に対象 UseCase の public method と呼び出し元から見える期待値を確認する。
- テストデータ追加はシナリオ名、目的、期待する観測結果が分かる単位で行う。
- DB 初期化は各テストまたは各サブテストが独立して再実行できるようにする。
- 失敗時に DB 種別、シナリオ名、UseCase 名が分かるテスト名にする。
- 単体テスト、結合テスト、handler 契約テストの責務を混ぜない。
