---
name: wails-clean-architecture
description: Wails v2 を前提にした Go 側クリーンアーキテクチャ規約。internal/domain、internal/usecase、internal/repository、internal/handler/wails の責務分離、Wails binding 配置、UseCase 入出力、Repository interface、作成禁止ファイル、go test 検証を伴う Go backend 実装・レビュー・リファクタ時に使用する。
---

# Wails Clean Architecture

## 基本方針

Wails 標準構成を維持し、Wails 依存は `handler/wails` と起動部分へ寄せる。
将来 Wails v3 へ移行する場合も、まず `internal/handler/wails` を Wails 依存境界として維持する。

## ディレクトリ責務

- `internal/domain`: エンティティ、値オブジェクト、ドメインルールを置く。
- `internal/usecase`: ユースケースと Repository interface を置く。
- `internal/repository`: `usecase` に定義された Repository interface の具体実装を置く。
- `internal/handler/wails`: Wails binding、Request / Response、frontend との契約を置く。

## 依存方向

- `domain` は Wails、usecase、repository、handler を知らない。
- `usecase` は domain を使い、Repository interface を所有する。
- `repository` は usecase の Repository interface を実装する。
- `handler/wails` は Request から値を取り出し、usecase を呼び、戻り値を Response に変換する。
- Wails 固有の型、DTO、frontend 契約は `handler/wails` の外へ漏らさない。

## app.go と main.go

- `app.go` は Wails ライフサイクル管理専用にする。
- `app.go` に binding メソッド、Request / Response、ビジネスロジックを置かない。
- `main.go` は DI を組み立て、`wails.Run` を呼ぶ。
- `Bind` には `internal/handler/wails` 配下の handler インスタンスを登録する。

## handler 層のログ位置

- `handler/wails` の binding メソッドでは、関数の先頭で呼び出しログを出す。
- `Fail(...)` を返す失敗経路では、失敗の `return` 直前に失敗ログを出す。
- `OK(...)` を返す成功経路では、成功の `return` 直前ログを原則出さない。
- 長時間処理、非同期処理、キャンセル可能処理では、必要に応じて終了ログを追加してよい。
- 高頻度に呼ばれる binding メソッドの入口ログは `Debug`、通常のユーザー操作は `Info` を基本にする。
- `domain` は原則としてログを出さない。業務上意味のある分岐や外部 I/O の失敗は、必要に応じて `usecase`、`repository`、`config` 側で記録する。

## UseCase 入出力

- UseCase のメソッド引数はプリミティブ値または domain 型を直接受け取る。
- UseCase の戻り値は domain 型、または必要最小限のプリミティブ値にする。
- `internal/usecase` に Request / Response / Input DTO / Output DTO を置かない。
- 引数が増えて可読性や不変条件の表現が崩れる場合は、DTO ではなく domain の値オブジェクト導入を優先する。

## 初期基盤タスクの制約

初期ディレクトリ構造のみを作るタスクでは、以下だけを作成する。

- `internal/domain/.gitkeep`
- `internal/usecase/.gitkeep`
- `internal/repository/.gitkeep`
- `internal/handler/wails/.gitkeep`

必要が明示された場合だけ、最小構成の `go.mod` を作成してよい。
実装ファイル、テストファイル、README、ADR、config、logger、errors、`cmd/`、`pkg/` は作成しない。

## 実装ファイル追加時の命名

- repository 実装名には保存方式を含める。
- 例: `FileUserRepository`
- 例: `SQLiteUserRepository`
- handler の Request / Response は `internal/handler/wails` に閉じ込める。

## 検証

- Go パッケージが存在する場合は `go test ./...` を実行する。
- `.go` ファイルがまだ存在せず `./... matched no packages` になる場合は、実装ファイルを作らない方針と整合する既知制約として報告する。
- 作成禁止ファイルが増えていないことを確認する。
