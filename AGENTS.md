# AGENTS.md

## 応答方針

- 日本語で簡潔かつ丁寧に回答する。

## プロジェクト方針

- Wails v2 を前提にする。
- 将来の Wails v3 移行に備え、Wails 依存は `internal/handler/wails` と起動部分へ寄せる。
- frontend は当面 Wails 標準構成のままとする。

## Go 側アーキテクチャ

- `internal/domain`: エンティティ、値オブジェクト、ドメインルールを置く。
- `internal/usecase`: ユースケースと Repository interface を置く。
- `internal/repository`: `usecase` に定義された Repository interface の具体実装を置く。
- `internal/handler/wails`: Wails binding、Request / Response、frontend との契約を置く。

## 依存ルール

- `domain` は Wails 依存を持たない。
- `usecase` は Wails 依存、Request / Response、Input DTO、Output DTO を持たない。
- Repository interface は `internal/usecase` に置く。
- `internal/repository` は Repository interface の具体実装だけを持つ。
- Wails 固有の DTO と frontend との契約は `internal/handler/wails` に閉じ込める。

## コメント方針

- コードコメントは基本的に日本語で記載する。
- 自明な処理説明コメントは追加しない。
- Go の `//go:embed` など、言語仕様やツールが要求するディレクティブコメントは英字表記のまま使用する。

## app.go / main.go

- `app.go` は Wails ライフサイクル管理用として残す。
- `app.go` に binding メソッド、Request / Response、ビジネスロジックを追加しない。
- `main.go` で DI を組み立て、`wails.Run` を呼ぶ。
- `Bind` には `internal/handler/wails` 配下の handler を登録する。

## UseCase 入出力

- UseCase のメソッド引数はプリミティブ値または domain 型を直接受け取る。
- UseCase の戻り値は domain 型、または必要最小限のプリミティブ値にする。
- 引数が増えて可読性や不変条件の表現が崩れる場合は、Request DTO ではなく domain の値オブジェクト導入を優先する。

## 初期基盤タスクの制約

- 空ディレクトリを Git 管理する場合は `.gitkeep` を置く。
- 初期段階では実装ファイルとテストファイルを作成しない。
- `cmd/`、`pkg/`、`internal/config/`、`internal/logger/`、`internal/errors/` は必要になった時点で追加する。
- 設計方針の詳細ドキュメントや ADR は、必要になった時点で別タスクとして追加する。

## 検証

- Go パッケージが存在する場合は `go test ./...` を実行する。
- `.go` ファイルが存在せず `no packages to test` になる場合は、その理由を報告する。
