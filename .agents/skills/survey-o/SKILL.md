---
name: survey-o
description: "Issue起点の調査オーケストレーション。調査目的、終了条件、未知領域、サブエージェント分担、結果統合、.local記録、Issueコメント化を計画・進行する。使う場面: Issueの調査開始、調査範囲の整理、設計前の材料収集、複数観点の並列調査。"
---

# survey-o

Issueを「設計に渡せる調査済み状態」にする司令塔として使う。調査そのものを抱え込まず、必要なreferenceとサブエージェントを選び、終了条件と記録を管理する。

## References

必要な場面だけ以下のreferenceを読む。調査が短縮できる場合は `brief.md`, `criteria.md`, `note.md` だけでよい。

| Reference | 読む場面 |
| --- | --- |
| [brief.md](references/brief.md) | Issueを作業前提へ圧縮するとき。 |
| [criteria.md](references/criteria.md) | 調査終了条件を固定するとき。 |
| [unknown-map.md](references/unknown-map.md) | 調査対象、ユーザー確認、設計判断を分けるとき。 |
| [dispatch.md](references/dispatch.md) | サブエージェント分担を設計するとき。 |
| [merge.md](references/merge.md) | 分担結果を統合するとき。 |
| [note.md](references/note.md) | `.local/<issue>/survey.md` とIssueコメント要約を作るとき。 |

## 実行計画

開始時に必ず以下を出力する。

- 目的: 今回の調査で明らかにすること。
- 終了条件: Issueから抽出する。なければユーザーと定義する。
- 使用reference: 必要なreferenceだけ選び、選んだ理由を短く書く。
- サブエージェント: 使う/使わない、使う場合の分担、入力、期待成果物、統合方法。
- 記録: `.local/<issue>/survey.md` に残す内容。
- Issue反映: 調査完了時にコメントする要約。

## 手順

1. `gh issue view <issue-number>` でIssueを読む。
2. `references/brief.md` で目的、背景、制約、決定済み事項を要約する。
3. `references/criteria.md` で調査終了条件を抽出する。ない場合はユーザーに確認する。
4. `references/unknown-map.md` で未知、曖昧さ、リスクを分類する。
5. 調査量が多い場合は `references/dispatch.md` でサブエージェント分担を設計する。単独で足りる場合は理由を明記する。
6. サブエージェントを使った場合は `references/merge.md` で結果を統合する。
7. `references/note.md` で `.local/<issue>/survey.md` とIssueコメント用要約を作る。

## 短縮ルート

Issueが十分具体的で追加調査が不要な場合は、`brief.md`, `criteria.md`, `note.md` だけで完了してよい。その場合も、調査を短縮した理由と設計へ渡す前提を記録する。

## 完了条件

- Issueの調査終了条件を満たしている。
- 根拠、未解決事項、設計へ渡す材料が `.local/<issue>/survey.md` にある。
- Issueに調査結果の要約が残っている。
