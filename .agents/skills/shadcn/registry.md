# Registry Authoring と Addresses

ユーザーが shadcn registry を作成、修正、公開、または理解したい場合にこの reference を使います。

## Mental Model

registry には2つの形式があります。

- **Source registry**: project または repository 内で authoring された `registry.json`。source files を指す `include` や file paths を使えます。
- **Built registry**: CLI consumer に提供される生成済み JSON files。通常は `public/r` から配信されます。この形式を作るには `npx shadcn@latest build` を使います。

CLI installer は registry item payloads を消費します。source registry は実ファイルからそれらの payloads を authoring する方法です。

Registry items は React components に限定されません。components、hooks、utilities、design tokens、pages、config files、docs、rules、workflows、templates、MCP files、その他 project files を配布できます。

## Root `registry.json`

root registry file では registry metadata と、`items` または `include` を定義します。

```json
{
  "$schema": "https://ui.shadcn.com/schema/registry.json",
  "name": "acme",
  "homepage": "https://acme.com",
  "items": [
    {
      "name": "absolute-url",
      "type": "registry:lib",
      "title": "Absolute URL",
      "description": "A utility to turn any path into an absolute URL.",
      "files": [
        {
          "path": "lib/absolute-url.ts",
          "type": "registry:lib"
        }
      ]
    }
  ]
}
```

Root registry rules:

- Root `registry.json` には `name` と `homepage` が必要です。
- `items` は registry item definitions の array です。
- source registry を複数 files に分割するために `include` を使えます。
- included registry files では `name` と `homepage` を省略できます。

## Include

大きな registries を modular に保つには `include` を使います。

```json
{
  "$schema": "https://ui.shadcn.com/schema/registry.json",
  "name": "acme",
  "homepage": "https://acme.com",
  "include": ["registry/ui/registry.json", "registry/blocks/registry.json"]
}
```

Include rules:

- Include paths は、それを宣言する `registry.json` からの相対パスです。
- Include paths は `registry.json` file を明示的に指す必要があります。
- remote URLs、absolute paths、parent traversal（`..`）は使いません。
- Item file paths は、item を宣言する registry file からの相対パスです。
- 解決済み registry 全体で item names が重複すると失敗します。

included file の例:

```json
{
  "items": [
    {
      "name": "button",
      "type": "registry:ui",
      "files": [
        {
          "path": "button.tsx",
          "type": "registry:ui"
        }
      ]
    }
  ]
}
```

この file が `registry/ui/registry.json` にある場合、`button.tsx` は `registry/ui/button.tsx` から読み取られ、built item path は root registry からの相対パスとして出力されます。

## Item Definitions

一般的な item fields:

```json
{
  "name": "login-form",
  "type": "registry:block",
  "title": "Login Form",
  "description": "A login form with email and password fields.",
  "dependencies": ["zod"],
  "registryDependencies": ["button", "input", "label"],
  "files": [
    {
      "path": "blocks/login-form.tsx",
      "type": "registry:block"
    }
  ],
  "cssVars": {
    "light": {
      "brand": "oklch(0.62 0.18 250)"
    },
    "dark": {
      "brand": "oklch(0.72 0.16 250)"
    }
  }
}
```

重要 fields:

- `name`: install 可能な item name。必ずしも file path ではありません。
- `type`: registry item type のいずれかです。例: `registry:ui`、`registry:block`、`registry:lib`、`registry:hook`、`registry:file`、`registry:page`、`registry:theme`、`registry:style`、`registry:font`、`registry:item`。
- `files`: item によって copy または generate される source files。
- `dependencies`: npm runtime dependencies。
- `devDependencies`: npm development dependencies。
- `registryDependencies`: この item が必要とする他の registry items。
- `cssVars`、`css`、`tailwind`、`envVars`、`docs`: install 時の optional additions。

File rules:

- File paths は宣言元の `registry.json` からの相対パスです。
- `registry:file` と `registry:page` files には `target` が必要です。
- source registry file paths で remote file URLs を使ってはいけません。
- source files は copy-paste 可能に保ちます。隠れた app-only imports を入れないでください。

## Registry Dependencies

`registryDependencies` entries は item addresses であり、file paths ではありません。

```json
{
  "name": "login-form",
  "type": "registry:block",
  "registryDependencies": ["button", "@acme/input", "acme/ui/card#v1.2.0"],
  "files": [
    {
      "path": "blocks/login-form.tsx",
      "type": "registry:block"
    }
  ]
}
```

Dependency rules:

- `"button"` のような bare names は official shadcn items を意味します。
- Bare names は same-registry または same-repository items を意味しません。
- Namespaced dependencies は `@namespace/item-name` を使います。
- GitHub dependencies は `owner/repo/item-name` を使います。
- 必要に応じて `owner/repo/item-name#ref` で GitHub dependencies を pin します。
- Refs は継承されません。`owner/repo/foo#v2` が同じ repo の `v2` にある `bar` に依存する場合は、`owner/repo/bar#v2` と書きます。
- `"./bar"` のような relative dependencies は使いません。

## Address Schemes

registry item string を扱うときは、まず分類します。

| Address                             | Scheme    | 意味 |
| ----------------------------------- | --------- | ---- |
| `button`                            | shadcn    | `button` という official shadcn item。 |
| `@acme/button`                      | namespace | configured registry `@acme` の item `button`。 |
| `@acme/ui/button`                   | namespace | configured registry `@acme` の item `ui/button`。 |
| `https://example.com/r/button.json` | url       | その URL にある built registry item JSON。 |
| `./button.json`                     | file      | disk 上の built registry item JSON。 |
| `acme/ui/button`                    | github    | GitHub repo `acme/ui` の item `button`。 |
| `acme/ui/forms/login#main`          | github    | GitHub repo `acme/ui` の ref `main` にある item `forms/login`。 |

namespace と GitHub addresses では slash を含む item names が許可されます。これは file paths ではなく item names です。`.json` で終わる addresses は file-address precedence を保つため、`acme/ui/data/schema.json` は GitHub item address ではなく file path として扱われます。

## GitHub Registries

root `registry.json` を持つ public GitHub repository は source registry として機能できます。

```txt
owner/repo/item-name[#ref]
```

Rules:

- 最初の2つの path segments は GitHub owner と repo です。
- 残りすべての path segments が registry item name です。
- source entrypoint は常に root `registry.json` です。
- GitHub registries は CLI が直接消費する source registries です。`shadcn build` や generated item JSON files は不要です。
- `include` は local registries と同じ source-registry rules に従います。
- 現時点で GitHub addresses がサポートするのは public `github.com` repositories のみです。
- Private repos と GitHub Enterprise には明示的な product decision が必要です。

GitHub registry fetching を実装する場合、source files を読む前に refs を commit SHA に解決します。branch-like refs は数分間 cache されることがあるため、`raw.githubusercontent.com` から moving refs を直接読まないでください。

推奨 flow:

```txt
owner/repo[#ref]
  -> git ls-remote で ref を解決
  -> commit SHA
  -> https://raw.githubusercontent.com/{owner}/{repo}/{sha}/registry.json を読む
  -> 同じ SHA から includes と item files を読む
```

これにより、1つの一貫した repository snapshot 上で command を保てます。

Full 40-character commit SHAs はすでに stable であり、そのまま使えます。Branches、tags、short refs は CLI が commit SHA に解決するため Git を必要とします。

## Build and Verify

source registries を build するには CLI を使います。

```bash
npx shadcn@latest build
npx shadcn@latest build registry.json --output public/r
```

結果を inspect するには CLI commands を使います。

```bash
npx shadcn@latest list @acme
npx shadcn@latest search @acme -q "login"
npx shadcn@latest view @acme/login-form
npx shadcn@latest add @acme/login-form --dry-run
npx shadcn@latest registry validate ./registry.json
```

public GitHub registries には GitHub addresses を直接使います。

```bash
npx shadcn@latest list owner/repo
npx shadcn@latest search owner/repo -q "login"
npx shadcn@latest view owner/repo/item
npx shadcn@latest add owner/repo/item --dry-run
npx shadcn@latest registry validate owner/repo
```

shadcn/ui codebase で registry implementation に取り組む場合:

- address parsing は pure かつ testable に保ちます。
- validators に side effects を追加しません。
- official shadcn、namespace、URL、file schemes の既存 behavior を維持します。
- address parsing、source loading、dependency resolution、list、search、view、add paths の tests を追加します。
- 複数の実 provider が存在するまでは、plugin system より小さな source-reader abstractions を優先します。
