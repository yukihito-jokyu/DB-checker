# Backend Spec

Go backendとWails境界を、実装前に具体化する。必要に応じて `wails-clean-architecture` skill も読む。

## 出力

- ユースケース目的
- ユースケース単位のバックエンド・フローチャート
- Wails handlerの責務
- Request/Response契約
- usecaseメソッドと入出力
- domain型/値オブジェクト
- repository interfaceと具体実装の境界
- validationとerror分類
- frontendへ返す状態
- 互換性と移行観点
- 実装対象と対象外

## ルール

- Wails依存は `internal/handler/wails` と起動部分へ寄せる。
- usecaseにRequest/ResponseやUI DTOを持ち込まない。
- frontend契約が必要なら `ui-spec.md` と接続する。
- バックエンドのフローチャートは、画面操作やWails handlerではなく、対象usecaseの開始を起点にする。
- フローチャートには、usecaseが呼び出すrepository、domainの判定、成功時の戻り値、失敗時のエラー分類だけを記載する。画面から外部依存までの責務境界は、別途シーケンス図で表す。
- 図が肥大化する場合は縮小せず、「全体の段階」「複雑な判断」「責務境界」「状態遷移」のどの問いに答えるかで分割する。各図の直前に答える問いを一文で記載する。
- ユースケースごとに、全体フローチャート、必要な判断の補助フローと決定表、シナリオ別シーケンス図、エラー対応表、事前条件・事後条件・不変条件を揃える。不変条件には、そのユースケースが変更してはならない状態を記載する。
