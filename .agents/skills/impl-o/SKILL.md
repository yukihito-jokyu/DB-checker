---
name: impl-o
description: "Issue起点の実装オーケストレーション。承認済み設計を実装、自己レビュー、検証、記録まで進める。使う場面: 実装着手、設計に沿った変更管理、サブエージェント分担、検証実行、Issueへの実装結果報告。"
---

# impl-o

承認済み設計を「レビュー・検証済みの実装」に変える司令塔として使う。実装中の判断、レビュー、検証、記録を管理する。

## References

必要な場面だけ以下のreferenceを読む。frontendやGo/Wails固有の実装規約は、既存の専門skillを必要に応じて追加で読む。

| Reference | 読む場面 |
| --- | --- |
| [brief.md](references/brief.md) | 承認済み設計、受け入れ条件、触る範囲を実装前提へ圧縮するとき。 |
| [dispatch.md](references/dispatch.md) | 調査、レビュー、検証観点を分担した方が速いとき。 |
| [decide.md](references/decide.md) | 実装中に出た設計外判断を記録するとき。 |
| [review.md](references/review.md) | 差分、受け入れ条件、回帰リスク、テスト不足を確認するとき。 |
| [verify.md](references/verify.md) | 検証を実行し、結果または不能理由を残すとき。 |
| [note.md](references/note.md) | `.local/<issue>/impl.md` とIssueコメント要約を作るとき。 |

## 実行計画

開始時に必ず以下を出力する。

- 目的: 今回実装すること。
- 前提: 承認済み設計、受け入れ条件、触る範囲。
- 使用reference: 必要なreferenceだけ選び、選んだ理由を短く書く。
- サブエージェント: 使う/使わない、使う場合の分担、入力、期待成果物、統合方法。実装編集は競合リスクを考えて慎重に分担する。
- 記録: `.local/<issue>/impl.md` に残す内容。
- 検証: 実行予定のテスト、lint、build、手動確認。

## 手順

1. `references/brief.md` で実装対象、受け入れ条件、設計上の制約を再確認する。
2. 必要なら `references/dispatch.md` で調査、テスト観点、レビュー観点を分担する。
3. 実装中に設計外の判断が出たら `references/decide.md` で記録する。重要判断はユーザーに戻す。
4. 実装を行う。frontend中心なら `frontend-implementation`、Go/Wails中心なら `wails-clean-architecture`、Goテスト作成時は `go-test-style` を必要に応じて参照する。
5. `references/review.md` で差分、リスク、受け入れ条件、回帰可能性を確認する。
6. `references/verify.md` でタスク種別に応じた検証を実行または実行不能理由を記録する。
7. `references/note.md` で `.local/<issue>/impl.md` とIssueコメント用要約を作る。

## 完了条件

- 実装、自己レビュー、検証が完了している。
- 作業内容、判断、検証結果が `.local/<issue>/impl.md` にある。
- Issueに実装結果と検証結果の要約が残っている。
