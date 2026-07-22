---
name: design-o
description: "Issue起点の詳細設計オーケストレーション。調査結果をユーザー承認済みの実装可能な設計へ進める。使う場面: 画面詳細設計、API/backend詳細設計、設計論点整理、認識合わせ、検証方針作成、実装前承認。"
---

# design-o

Issueを「実装してよい設計済み状態」にする司令塔として使う。設計対象に応じて必要なreferenceを読み、ユーザー承認まで進行する。

## References

必要な場面だけ以下のreferenceを読む。画面設計なら `ui-spec.md`、backend設計なら `backend-spec.md` を優先する。

| Reference | 読む場面 |
| --- | --- |
| [brief.md](references/brief.md) | Issue、調査結果、既存制約を設計前提へ圧縮するとき。 |
| [criteria.md](references/criteria.md) | 設計承認条件と実装受け入れ条件を固定するとき。 |
| [unknown-map.md](references/unknown-map.md) | 認識齟齬、暗黙前提、ユーザーの未知領域を表面化するとき。 |
| [ask.md](references/ask.md) | ユーザー確認を作るとき。 |
| [ui-spec.md](references/ui-spec.md) | 画面、状態、操作、React境界を詳細化するとき。 |
| [backend-spec.md](references/backend-spec.md) | Wails/Go backend/API契約を詳細化するとき。 |
| [decide.md](references/decide.md) | 採用案、却下案、理由、影響を記録するとき。 |
| [verify.md](references/verify.md) | 実装後の検証方針を固定するとき。 |
| [note.md](references/note.md) | `.local/<issue>/design.md` とIssueコメント要約を作るとき。 |

## 実行計画

開始時に必ず以下を出力する。

- 目的: 今回の設計で決めること。
- 前提: Issue、調査結果、制約、既存設計。
- 使用reference: 必要なreferenceだけ選び、選んだ理由を短く書く。
- サブエージェント: 使う/使わない、使う場合の分担、入力、期待成果物、統合方法。
- 記録: `.local/<issue>/design.md` と `.local/<issue>/approvals.md` に残す内容。
- 承認: ユーザーに承認を求める単位。

## 手順

1. `references/brief.md` で設計対象と前提を短く再整理する。
2. 必要なら `references/criteria.md` で設計承認条件と実装受け入れ条件を固定する。
3. `references/unknown-map.md` で設計上の未決事項、暗黙前提、破綻条件を洗う。
4. 必要なら `references/ask.md` でユーザー確認を作る。
5. 画面・UX・React中心なら `references/ui-spec.md` を読む。
6. Wails handler、usecase、repository、domain中心なら `references/backend-spec.md` を読む。
7. 両方にまたがる場合は `ui-spec.md` と `backend-spec.md` の結果を `references/decide.md` で接続する。
8. `references/verify.md` で実装後の検証方針を決める。
9. `references/note.md` で `.local/<issue>/design.md` とIssueコメント用要約を作る。
10. ユーザー承認を得た単位ごとに、`.local/<issue>/approvals.md` へ日付、承認内容、対象範囲、未承認事項、根拠となるユーザー発言を追記する。
11. 承認なしに実装工程へ進めない。`impl-o` または実装スキルを開始するには、`approvals.md` に「実装着手を承認」と明記された記録が必要である。

## 完了条件

- 採用案、却下案、理由、検証方針、除外範囲が `.local/<issue>/design.md` にある。
- ユーザー承認の単位と根拠が `.local/<issue>/approvals.md` にある。
- ユーザー承認済みの要約がIssueに残っている。

## モック・プロトタイプ・壁打ち

モック、プロトタイプ、壁打ちは、明示的な実装依頼または設計承認がない限り、`.local/<issue>/` 配下の成果物として作成する。プロダクトコードは変更しない。

画面構成・画面状態・ダイアログの壁打ちは、HTMLスナップショットを `.local/<issue>/html/` に連番付きのファイル名（例: `01-overview.html`）で保存する。スナップショットには、承認済みの内容と検討中の内容を区別して表示し、検討中の提案を決定事項として扱わない。

backendの処理、ユースケースのフローチャート、責務境界のシーケンス図は、`.local/<issue>/mmd/` に連番付きのMarkdown/Mermaid成果物として保存する。画面確認を目的としないbackend設計の内容を、HTMLスナップショットとして重複出力しない。
