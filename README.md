# DB-checker
ローカルDBにダミーデータを投入し、DB設計の有効性をUI上で検証できるようにするツール

## 開発コマンド

Go backend の検証には Taskfile を使用する。

```bash
task format        # gofmt で Go ファイルを整形する
task format:check  # gofmt 済みか確認する
task lint          # golangci-lint を実行する
task test          # go test ./... を実行する
task backend:check # format:check / lint / test をまとめて実行する
```

`task lint` には `golangci-lint` が必要。CI では `golangci-lint` を `v2.12.2` に固定して実行する。
