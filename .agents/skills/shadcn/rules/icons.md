# アイコン

**import には必ずプロジェクトで設定された `iconLibrary` を使います。** プロジェクトコンテキストの `iconLibrary` フィールドを確認してください。`lucide` → `lucide-react`、`tabler` → `@tabler/icons-react` などです。`lucide-react` だと決めつけてはいけません。

---

## Button 内のアイコンには data-icon 属性を使う

アイコンに `data-icon="inline-start"`（前置）または `data-icon="inline-end"`（後置）を追加します。アイコンにサイズ指定クラスは付けません。

**誤り:**

```tsx
<Button>
  <SearchIcon className="mr-2 size-4" />
  Search
</Button>
```

**正しい:**

```tsx
<Button>
  <SearchIcon data-icon="inline-start"/>
  Search
</Button>

<Button>
  Next
  <ArrowRightIcon data-icon="inline-end"/>
</Button>
```

---

## コンポーネント内のアイコンにサイズ指定クラスを付けない

コンポーネントは CSS でアイコンサイズを扱います。`Button`、`DropdownMenuItem`、`Alert`、`Sidebar*`、その他 shadcn コンポーネント内のアイコンに `size-4`、`w-4 h-4`、その他のサイズ指定クラスを追加しないでください。ただし、ユーザーが明示的にカスタムアイコンサイズを求めた場合は除きます。

**誤り:**

```tsx
<Button>
  <SearchIcon className="size-4" data-icon="inline-start" />
  Search
</Button>

<DropdownMenuItem>
  <SettingsIcon className="mr-2 size-4" />
  Settings
</DropdownMenuItem>
```

**正しい:**

```tsx
<Button>
  <SearchIcon data-icon="inline-start" />
  Search
</Button>

<DropdownMenuItem>
  <SettingsIcon />
  Settings
</DropdownMenuItem>
```

---

## アイコンは文字列キーではなくコンポーネントオブジェクトとして渡す

lookup map への文字列キーではなく、`icon={CheckIcon}` を使います。

**誤り:**

```tsx
const iconMap = {
  check: CheckIcon,
  alert: AlertIcon,
}

function StatusBadge({ icon }: { icon: string }) {
  const Icon = iconMap[icon]
  return <Icon />
}

<StatusBadge icon="check" />
```

**正しい:**

```tsx
// プロジェクトで設定された iconLibrary から import する（例: lucide-react, @tabler/icons-react）。
import { CheckIcon } from "lucide-react"

function StatusBadge({ icon: Icon }: { icon: React.ComponentType }) {
  return <Icon />
}

<StatusBadge icon={CheckIcon} />
```
