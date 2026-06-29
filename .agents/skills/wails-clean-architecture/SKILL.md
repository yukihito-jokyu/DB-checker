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

## Wails Binding 命名と契約

- Binding メソッドは操作種別が分かる `CreateX` / `GetX` / `ListX` / `UpdateX` / `DeleteX` を基本にする。
- 単一リソースや単一状態の読み取りは `GetX`、複数件の読み取りは `ListX` にする。
- Wails の成功・失敗レスポンスは原則として `Response[T]` の `data` / `error` 形式に統一する。
- frontend へ返す Error DTO は `code` / `message` を持ち、`apperr` から `handler/wails` で変換する。
- 短時間で完了する Binding に cancellation 用の仕組みを先取りで追加しない。
- DB 接続、スキーマ取得、データ取得など時間が読めない処理では、`usecase` / `repository` の I/O 系メソッドに `context.Context` を渡す。
- `context.Canceled` は `OPERATION_CANCELED`、`context.DeadlineExceeded` は `OPERATION_TIMEOUT` に変換する。

## 統計取得のキャンセルとタイムアウト

- 統計取得は Issue #1 の要件に従い、10秒の固定タイムアウトを持つ。
- 統計取得中に別テーブルの統計取得を開始した場合、`handler/wails` が同じ統計取得カテゴリの前回処理を自動キャンセルする。
- 統計取得のキャンセル関数、最新リクエスト管理、前回処理の破棄判定は `handler/wails` が所有する。
- `usecase` は `context.Context` を受け取り、DB処理や集計処理の各段階でキャンセルに協力するだけにする。
- 統計取得の明示的な `CancelX` Binding は、ユーザー操作としてのキャンセルが要件化されるまで追加しない。
- 統計取得タイムアウトは完全失敗ではなく部分成功として扱い、取得済み結果と `status: "timeout"` を `data` で返す。
- 統計取得キャンセルは部分結果表示ではなく制御フローとして扱い、`OPERATION_CANCELED` の `error` を返す。ただし frontend では通知対象外にする。
- frontend は統計取得 hook または統計画面内で sequence id を管理し、最新 sequence 以外の結果を状態へ反映しない。
- frontend 共通 API 層や統計 DTO に request id / sequence id を先取りで追加しない。
- 通常のテーブル選択による統計取得中はテーブル切り替えをブロックしない。UI編集後のデータ・統計再取得中だけ、追加編集、テーブル切り替え、フィルタ変更、並び替え変更などをブロックする。
- 実DB統計取得が未実装の段階では、production Wails Binding を半端に公開しない。まず `handler/wails` 内の未公開ヘルパーと Go テストでキャンセル境界を検証する。

## Wails Events の利用方針

- 画面操作に対して結果が1回返れば足りる処理は、Wails Events ではなく Binding の戻り値で扱う。
- Wails Events は、長時間処理の進捗、backend 起点の状態変化、複数画面や複数 component へ同時通知したい変更に限定する。
- 単なる成功結果、画面内だけで完結する UI 状態、通常の再取得で済むデータ更新には Events を使わない。
- イベント名は `<domain>:<action>:<phase>` を基本形にする。例: `schema:reload:started`、`schema:reload:completed`、`config:changed`。
- payload は `EventsEmit(name, payload)` のようにオブジェクト1個で渡す。複数引数のイベント payload は使わない。
- payload なしで意味が通るイベントだけ、payload なしを許可する。
- 長時間処理や再取得競合があり得る通知には、`requestId` または対象 ID を payload に含める。
- Error を通知する場合は、`code` / `message` を持つ軽い DTO にする。
- frontend では Wails runtime の `EventsOn` を component から直接呼ばず、`src/lib/wails/events.ts` の薄い adapter 経由で購読する。
- React component / hook では `useEffect` の cleanup で購読解除関数を必ず呼ぶ。
- `EventsOffAll` は他機能の購読も解除し得るため、原則として使わない。
- frontend は、現在注目している `requestId` または対象 ID と一致しないイベントを無視する。
- Events は通知として扱い、最終的な正のデータは必要に応じて Binding で再取得する。
- backend 側に先取りでグローバルな重複排除機構を作らない。

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
- frontend 返却データは `XData`、Request が必要な場合は `CreateXRequest` / `UpdateXRequest` のように操作名を含める。
- Request DTO は必要になった時だけ追加し、空の将来用 DTO は作らない。

## 検証

- Go パッケージが存在する場合は `go test ./...` を実行する。
- `.go` ファイルがまだ存在せず `./... matched no packages` になる場合は、実装ファイルを作らない方針と整合する既知制約として報告する。
- 作成禁止ファイルが増えていないことを確認する。
