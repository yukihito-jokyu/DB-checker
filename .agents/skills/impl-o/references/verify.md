# Verify

タスク種別に応じた検証を実行し、結果または実行不能理由を記録する。

## 検証候補

- Go: `task test`, `task format:check`, `task lint`, `task backend:check`
- frontend: `task frontend:check`, `task frontend:build`
- Wails連携: `task generate`, 必要に応じて起動確認
- ドキュメント/skill: `quick_validate.py <skill-folder>`、リンク/参照整合性確認
- 手動確認: 画面操作、エラー表示、空状態、既存動作

## 出力

- 検証計画
- 実行した検証
- 結果
- 未実行の検証と理由
- 残リスク

## ルール

- Taskfileにある操作を優先する。
- 実行できない検証を黙って省略しない。
