# Verify

何をもって正しいと判断するかを、設計時点で決める。

## 検証候補

- Go: `task test`, `task format:check`, `task lint`, `task backend:check`
- frontend: `task frontend:check`, `task frontend:build`
- Wails連携: `task generate`, 必要に応じて起動確認
- ドキュメント/skill: `quick_validate.py <skill-folder>`、リンク/参照整合性確認
- 手動確認: 画面操作、エラー表示、空状態、既存動作

## 出力

- 検証計画
- 実装後に実行する検証
- 未実行になり得る検証と理由
- 残リスク
