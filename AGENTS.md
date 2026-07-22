# AGENTS.md

## 応答方針

- 日本語で簡潔かつ丁寧に回答する。
- 実在するローカルファイルを案内する場合は、絶対パスを指定したクリック可能な Markdown リンクで記載する。

## プロジェクト方針

- Wails v2 を前提にする。
- 将来の Wails v3 移行に備え、Wails 依存は `internal/handler/wails` と起動部分へ寄せる。
- frontend は当面 Wails 標準構成のままとする。

## Go 側アーキテクチャ

- `internal/domain`: エンティティ、値オブジェクト、ドメインルールを置く。
- `internal/usecase`: ユースケースと Repository interface を置く。
- `internal/repository`: `usecase` に定義された Repository interface の具体実装を置く。
- `internal/handler/wails`: Wails binding、Request / Response、frontend との契約を置く。
- `internal/config`: アプリ設定の読み込み、検証、設定値型を置く。
- `internal/logger`: ログ出力の抽象化と初期化を置く。
- `internal/errors`: アプリ共通のエラー分類、判定、ラップ補助を置く。

## 依存ルール

- `domain` は Wails 依存を持たない。
- `usecase` は Wails 依存、Request / Response、Input DTO、Output DTO を持たない。
- Repository interface は `internal/usecase` に置く。
- `internal/repository` は Repository interface の具体実装だけを持つ。
- Wails 固有の DTO と frontend との契約は `internal/handler/wails` に閉じ込める。
- `internal/config` に Wails Request / Response や UI 表示用 DTO を置かない。
- `internal/logger` は Wails runtime に直接依存せず、起動部分から注入できる形にする。
- `internal/errors` に HTTP 的なステータスや frontend 表示文言を置かない。

## コメント方針

- コードコメントは基本的に日本語で記載する。
- 自明な処理説明コメントは追加しない。
- Go の全関数・メソッドは、直前に責務を示す一行コメントを記載する。
- 関数前コメントは短い名詞句とし、「〜は…する」、実装経緯、複数文を使わない。
- 複雑な関数では、分岐や変換、外部境界など理解の補助が必要な箇所へ関数内コメントを記載する。
- Go の関数内では、ログ、外部呼び出し、分岐、返却など責務が切り替わる論理ブロックを空行で区切る。
- 同じ目的の連続処理には不要な空行を入れない。
- TSX のコンポーネント関数では、関数名の前にコメントを置かず、取得処理、表示分岐、イベント処理など意図の補助が必要な関数内部へコメントを記載する。
- Go の `//go:embed` など、言語仕様やツールが要求するディレクティブコメントは英字表記のまま使用する。

## app.go / main.go

- `app.go` は Wails ライフサイクル管理用として残す。
- `app.go` に binding メソッド、Request / Response、ビジネスロジックを追加しない。
- `main.go` で DI を組み立て、`wails.Run` を呼ぶ。
- `Bind` には `internal/handler/wails` 配下の handler を登録する。

## UseCase 入出力

- UseCase のメソッド引数はプリミティブ値または domain 型を直接受け取る。
- UseCase の戻り値は domain 型、または必要最小限のプリミティブ値にする。
- 引数が増えて可読性や不変条件の表現が崩れる場合は、Request DTO ではなく domain の値オブジェクト導入を優先する。

## 初期基盤タスクの制約

- 空ディレクトリを Git 管理する場合は `.gitkeep` を置く。
- 初期段階では実装ファイルとテストファイルを作成しない。
- `internal/config/`、`internal/logger/`、`internal/errors/` は責務境界を先に固定するため、空ディレクトリとして追加してよい。
- `cmd/`、`pkg/` は必要になった時点で追加する。
- 設計方針の詳細ドキュメントや ADR は、必要になった時点で別タスクとして追加する。

## 検証

- Go / frontend / Wails 関連の定型操作は、原則として `Taskfile.yml` に定義された `task` コマンドを使用する。
- Go パッケージが存在する場合は `task test` を実行する。
- `.go` ファイルが存在せず `no packages to test` になる場合は、その理由を報告する。

| コマンド | 説明 |
| --- | --- |
| `task install` | `frontend` 配下で `npm install` を実行し、frontend の依存関係をインストールする。 |
| `task generate` | Wails の frontend binding を生成し、Go と frontend の連携コードを更新する。 |
| `task dev` | Wails 開発サーバーを起動し、アプリを開発モードで実行する。 |
| `task test` | Go のテストを `go test ./...` で実行する。 |
| `task format` | Go ファイルを `gofmt` で整形する。 |
| `task format:check` | Go ファイルが `gofmt` 済みか確認し、未整形ファイルがあれば失敗する。 |
| `task lint` | `golangci-lint` で Go コードの lint を実行する。 |
| `task backend:check` | `format:check`、`lint`、`test` をまとめて実行する。 |
| `task frontend:build` | `frontend` 配下で production build を実行する。 |
| `task frontend:check` | `frontend` 配下で Biome check を実行する。 |
| `task frontend:lint` | `frontend` 配下で lint を実行する。 |
| `task frontend:format` | `frontend` 配下で format を実行する。 |
