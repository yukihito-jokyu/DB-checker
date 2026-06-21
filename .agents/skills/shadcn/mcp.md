# shadcn MCP Server

CLI には、AI アシスタントが registry の item を検索、閲覧、表示、install できる MCP server が含まれています。

---

## セットアップ

```bash
shadcn mcp        # MCP server を起動する（stdio）
shadcn mcp init   # editor 用 config を書き込む
```

Editor config files:

| Editor      | Config file                     |
| ----------- | ------------------------------- |
| Claude Code | `.mcp.json`                     |
| Cursor      | `.cursor/mcp.json`              |
| VS Code     | `.vscode/mcp.json`              |
| OpenCode    | `opencode.json`                 |
| Codex       | `~/.codex/config.toml` (manual) |

---

## Tools

> **Tip:** MCP tools は registry 操作（search、view、install）を扱います。プロジェクト設定（aliases、framework、Tailwind version）には `npx shadcn@latest info` を使います。MCP に相当機能はありません。

### `shadcn:get_project_registries`

`components.json` から registry 名を返します。`components.json` が存在しない場合はエラーになります。

**Input:** none

### `shadcn:list_items_in_registries`

1つ以上の registry からすべての item を一覧表示します。Registry には `@acme` のような設定済み namespace、`owner/repo` のような public GitHub source、または registry catalog URL を指定できます。`components.json` に設定されたすべての registry から一覧表示する場合は、`registries` を省略します。

**Input:** `registries` (string[], optional — すべての設定済み registry では省略), `types` (string[], optional — 例: `["ui", "block"]`), `limit` (number, optional, default は 100), `offset` (number, optional)

### `shadcn:search_items_in_registries`

Registry 横断の fuzzy search です。Registry には設定済み namespace、public GitHub source、または registry catalog URL を指定できます。`components.json` に設定されたすべての registry を検索する場合は `registries` を省略します。例: すべての設定済み registry を対象に "find me a hero"。

**Input:** `registries` (string[], optional — すべての設定済み registry では省略), `query` (string), `types` (string[], optional — 例: `["ui", "block"]`), `limit` (number, optional, default は 100), `offset` (number, optional)

### `shadcn:view_items_in_registries`

完全なファイル内容を含む item 詳細を表示します。

**Input:** `items` (string[]) — 例:
`["@shadcn/button", "@shadcn/card", "owner/repo/item"]`

### `shadcn:get_item_examples_from_registries`

source code 付きの usage example と demo を探します。`components.json` に設定されたすべての registry を検索する場合は、`registries` を省略します。

**Input:** `registries` (string[], optional — すべての設定済み registry では省略), `query` (string) — 例: `"accordion-demo"`, `"button example"`

### `shadcn:get_add_command_for_items`

CLI install command を返します。

**Input:** `items` (string[]) — 例: `["@shadcn/button"]`

### `shadcn:get_audit_checklist`

コンポーネント検証用 checklist（imports、deps、lint、TypeScript）を返します。

**Input:** none

---

## Registry の設定

Namespaced registry と authenticated registry は `components.json` に設定します。`@shadcn` registry は常に built-in です。root `registry.json` を持つ repository であれば、public GitHub registry も `owner/repo` registry source として直接使えます。`components.json` の設定は不要です。

```json
{
  "registries": {
    "@acme": "https://acme.com/r/{name}.json",
    "@private": {
      "url": "https://private.com/r/{name}.json",
      "headers": { "Authorization": "Bearer ${MY_TOKEN}" }
    }
  }
}
```

- 名前は `@` で始める必要があります。
- URL には `{name}` が必要です。
- `${VAR}` 参照は環境変数から解決されます。

Community registry index: `https://ui.shadcn.com/r/registries.json`
