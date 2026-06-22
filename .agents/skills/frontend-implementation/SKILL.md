---
name: frontend-implementation
description: DB-checker の Wails v2 / React / TypeScript フロントエンドを実装・改修するための手順。frontend/src 配下の画面、コンポーネント、ルーティング、Tailwind CSS、Wails 生成バインディング、UI状態管理、アクセシビリティ、ビルド検証を伴うフロントエンド機能追加・修正・リファクタ時に使用する。
---

# Frontend Implementation

## Overview

DB-checker のフロントエンド実装では、Wails 標準構成を維持し、React / TypeScript / Tailwind CSS の既存パターンに合わせる。
Go 側との契約は `internal/handler/wails` と `frontend/wailsjs` の境界を尊重し、画面側にバックエンド都合を漏らしすぎない。

## Workflow

1. 依頼内容から対象ユーザー、主要操作、表示状態、失敗状態を確認する。
2. `frontend/src`、`frontend/package.json`、`frontend/wailsjs`、関連する `internal/handler/wails` を読んで既存契約と命名を把握する。
3. 既存のルーティング、コンポーネント分割、CSS/Tailwind の使い方を優先して設計する。
4. 必要な UI を実装し、ローディング、空状態、エラー、無効化、再試行などの状態を同時に扱う。
5. `frontend/wailsjs/go` の生成コードを直接編集しない。Go 側 binding が変わった場合は Wails の生成手順で更新する。
6. ビルドと、対象機能に応じたテストまたは手動確認を実行する。

## Frontend Structure

- `frontend/src` に画面、コンポーネント、hooks、lib を置く。既存ディレクトリがない場合は、必要になった最小単位で追加する。
- React Router のルーティング定義は `frontend/src/app/router.tsx` に集約し、`frontend/src/main.tsx` は `RouterProvider` へ router を渡すエントリポイントに限定する。
- `frontend/wailsjs` は生成物として扱い、手編集しない。
- Wails 生成 binding の import は、疎通確認を除き service 境界に閉じ込める。
- Wails の runtime API は画面全体へ散らさず、`src/lib/wails/*.ts` の薄い adapter に閉じ込める。
- shadcn/ui の導入済み設定があるため、UI部品を追加する場合は `components.json` の alias と Tailwind 設定に合わせる。
- CSS はまず Tailwind utility と既存CSSで表現する。複雑な画面固有スタイルが必要な場合だけ、近い責務のファイルへ限定する。

## Implementation Guidelines

- React コンポーネントは props 境界を明確にし、ページコンポーネントへ過剰な表示ロジックを集めない。
- データ取得、変換、表示、操作ハンドラの責務を分ける。小さな画面では過度に抽象化しない。
- TypeScript 型は Wails 生成型を活用する。画面専用の派生型が必要な場合は局所化する。
- ユーザー操作はキーボード、フォーカス、aria 属性、disabled 状態を含めて実装する。
- テーブル、リスト、キャンバス、詳細パネルなどの情報密度が高いUIでは、装飾より読み取りやすさと操作効率を優先する。
- 長文、長いファイル名、狭いウィンドウでレイアウトが崩れないように、折り返し、min/max、overflow を明示する。
- 依存追加は既存の React / Tailwind / Radix 系で足りない場合だけにする。追加時は理由と影響範囲を確認する。

## Wails Boundary

- frontend から直接呼ぶ API は `frontend/wailsjs/go/...` の生成関数に限定する。
- Wails binding 呼び出しは薄い service 境界に閉じ込める。feature 所属なら `src/features/<feature>/services/*.ts`、page 固有なら `src/pages/<page>/services/*.ts`、app 全体の初期化・疎通・バージョン取得なら `src/app/services/*.ts` に置く。
- `src/api` は作成しない。Wails binding を扱うためだけに `api` という名前のディレクトリを作らない。
- Wails runtime API の adapter は `src/lib/wails/*.ts` に置き、ここには業務 API 呼び出しを置かない。
- Request / Response の形を変える必要がある場合は、Go 側では `internal/handler/wails` に閉じ込める。
- usecase / domain の型や都合を frontend のために崩さない。必要なら handler で frontend 向け Response へ変換する。
- Wails binding 変更後は生成コマンドを実行し、生成差分を確認してから frontend 実装へ反映する。

## UI Quality Bar

- 初期表示、読み込み中、成功、空、エラー、再試行、操作中、部分的失敗を必要に応じて用意する。
- 画面内説明文で機能を過剰に説明しない。ボタン、ラベル、状態表示で自然に理解できるUIにする。
- アイコンが使える場面では既存のアイコンライブラリを優先する。未導入なら無理に追加しない。
- カードの入れ子、過剰なヒーロー表現、単色に偏った配色を避ける。
- ボタンやパネル内のテキストがモバイル幅でもはみ出さないように確認する。
- 業務アプリとして、静かでスキャンしやすいレイアウトを優先する。

## Verification

- TypeScript / Vite の変更後は `rtk npm run build` を `frontend` で実行する。
- lint や formatter のスクリプトが追加されている場合は、該当スクリプトも実行する。
- Wails binding 変更を含む場合は、生成コード更新後に Go テストも実行する。
- 画面挙動が重要な変更では、可能なら dev server または Playwright で表示確認する。
- 検証できなかった項目は、理由と残リスクを最終報告に含める。

## Completion Checklist

- 既存の画面構成、命名、スタイルから逸脱していない。
- Wails 生成物を手編集していない。
- ローディング、空、エラー、長文、狭幅表示のいずれかが関係する場合は考慮済み。
- ビルドまたは対象テストを実行済み。
- 未検証項目があれば明示済み。
