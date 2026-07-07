# Backend Spec

Go backendとWails境界を、実装前に具体化する。必要に応じて `wails-clean-architecture` skill も読む。

## 出力

- ユースケース目的
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
