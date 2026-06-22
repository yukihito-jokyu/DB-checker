---
name: go-test-style
description: Go のテストコードを作成・修正・レビューするときに、失敗処理、期待値比較、サブテスト、エラーメッセージの書き方を統一する。特に testing.T の Fatalf/Fatal と Errorf/Error の使い分け、テーブル駆動テスト、errors パッケージ周辺の単体テスト、internal 配下の Go テストを扱う場面で使用する。
---

# Go Test Style

## 基本方針

Go のテストは、失敗時に原因を追いやすく、1 回の実行で有用な差分をできるだけ多く確認できる形にする。前提条件が崩れたときだけ即停止し、独立した期待値比較は継続して報告する。

## Fatal と Error の使い分け

- `t.Fatalf` / `t.Fatal` は、後続処理を続けると panic する、または検証の前提が崩れて意味がなくなる場合だけ使う。
- `t.Errorf` / `t.Error` は、失敗しても後続の検証が安全に続けられる独立した値比較に使う。
- nil 参照を防ぐチェック、型アサーションの成否、必須の戻り値が得られたかの確認は `Fatal` 系を使う。
- `Code`、`Message`、`Err`、`Error()`、`Unwrap()`、真偽値の結果など、互いに独立して確認できるものは `Error` 系を使う。

```go
appErr := As(err)
if appErr == nil {
	t.Fatal("As() = nil, want app error")
}
if appErr.Code != wantCode {
	t.Errorf("Code = %q, want %q", appErr.Code, wantCode)
}
```

## 期待値比較の書き方

- 失敗メッセージは原則として `got, want` の両方を出す。
- メソッドや関数の戻り値は `Function() = got, want want` の形に寄せる。
- フィールド比較は `Field = got, want want` の形に寄せる。
- 期待値の文字列が長くない場合は、テスト本文内で直接比較してよい。
- 同じ期待値を複数箇所で使う場合や可読性が落ちる場合は `want` 変数に分ける。
- 小さなエラー型のように、各フィールドの失敗理由を個別に見たい場合は、フィールドごとに比較してよい。
- 一方で、通常の struct / slice / map の戻り値は、期待値全体を作って `reflect.DeepEqual` や `cmp.Diff` で比較することも検討する。

```go
if got := string(err.Message); got != "処理がタイムアウトしました" {
	t.Errorf("Message = %q, want %q", got, "処理がタイムアウトしました")
}
```

## サブテストとテーブル駆動

- 分岐やパターンが複数ある場合は、`tests := []struct { ... }` と `t.Run(tt.name, ...)` を使う。
- 1 ケースだけの単純なコンストラクタ検証は、テーブル化せず素直に書いてよい。
- `wantFound == false` のように以降の検証が不要なケースでは、失敗ではなく早期 `return` で抜ける。
- `wantFound == false` のように、見つからないこと自体が期待結果であり、以降の詳細比較が不要なケースでは `return` で正常に抜ける。
- サブテスト名は、失敗ログで読んだときに意味が分かる名前にする。

```go
if gotFound := appErr != nil; gotFound != tt.wantFound {
	t.Fatalf("found = %v, want %v", gotFound, tt.wantFound)
}
if !tt.wantFound {
	return
}
```

## errors 周辺の検証

- 内部の保持状態を確認したい場合は、`Unwrap()` や保持フィールドを直接検証する。
- 外部利用者から見た振る舞いを確認したい場合は、`errors.Is` / `errors.As` を検証する。
- どちらを主検証にするかは、そのテストが「内部構造」を見たいのか「公開 API としての振る舞い」を見たいのかで決める。
- 補助検証も、後続に危険がなければ `t.Error` / `t.Errorf` にする。

## テストヘルパー

- テスト用ヘルパーを作る場合は、原則として先頭で `t.Helper()` を呼ぶ。
- テストの本質に関係しない抽象化は避け、同じ前提確認や変換が複数回出てくる場合だけヘルパー化する。

## 並行処理テスト

- goroutine 内で `t.Fatal` / `t.Fatalf` を直接呼ばない。
- goroutine 内の失敗は channel などで親 goroutine に返し、親側で `t.Fatal` / `t.Error` する。

## 避けること

- すべての不一致を機械的に `t.Fatalf` にしない。
- 後続で nil 参照する可能性があるのに `t.Errorf` で続行しない。
- 期待値を書かず、実際値だけを出す失敗メッセージにしない。
- テストの本質に関係しない抽象化やヘルパーを増やさない。

## 検証

Go テストを変更したら、プロジェクトの `Taskfile.yml` に従って `rtk task test` を実行する。対象パッケージだけではなく、原則として Go テスト全体の成功を確認する。
