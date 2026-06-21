# コンポーネント構成

## 目次

- Item は必ず対応する Group コンポーネント内に置く
- コールアウトには Alert を使う
- 空状態には Empty コンポーネントを使う
- Toast 通知には sonner を使う
- オーバーレイコンポーネントの使い分け
- Dialog、Sheet、Drawer には必ず Title が必要
- Card の構造
- Button に isPending / isLoading prop はない
- TabsTrigger は必ず TabsList 内に置く
- Avatar には必ず AvatarFallback が必要
- 生の hr や border div ではなく Separator を使う
- 読み込みプレースホルダーには Skeleton を使う
- 独自スタイルの span ではなく Badge を使う

---

## Item は必ず対応する Group コンポーネント内に置く

Item を content コンテナ直下に直接レンダリングしてはいけません。

**誤り:**

```tsx
<SelectContent>
  <SelectItem value="apple">Apple</SelectItem>
  <SelectItem value="banana">Banana</SelectItem>
</SelectContent>
```

**正しい:**

```tsx
<SelectContent>
  <SelectGroup>
    <SelectItem value="apple">Apple</SelectItem>
    <SelectItem value="banana">Banana</SelectItem>
  </SelectGroup>
</SelectContent>
```

これは Group ベースのすべてのコンポーネントに適用されます。

| Item | Group |
|------|-------|
| `SelectItem`, `SelectLabel` | `SelectGroup` |
| `DropdownMenuItem`, `DropdownMenuLabel`, `DropdownMenuSub` | `DropdownMenuGroup` |
| `MenubarItem` | `MenubarGroup` |
| `ContextMenuItem` | `ContextMenuGroup` |
| `CommandItem` | `CommandGroup` |

---

## コールアウトには Alert を使う

```tsx
<Alert>
  <AlertTitle>Warning</AlertTitle>
  <AlertDescription>Something needs attention.</AlertDescription>
</Alert>
```

---

## 空状態には Empty コンポーネントを使う

```tsx
<Empty>
  <EmptyHeader>
    <EmptyMedia variant="icon"><FolderIcon /></EmptyMedia>
    <EmptyTitle>No projects yet</EmptyTitle>
    <EmptyDescription>Get started by creating a new project.</EmptyDescription>
  </EmptyHeader>
  <EmptyContent>
    <Button>Create Project</Button>
  </EmptyContent>
</Empty>
```

---

## Toast 通知には sonner を使う

```tsx
import { toast } from "sonner"

toast.success("Changes saved.")
toast.error("Something went wrong.")
toast("File deleted.", {
  action: { label: "Undo", onClick: () => undoDelete() },
})
```

---

## オーバーレイコンポーネントの使い分け

| 用途 | コンポーネント |
|----------|-----------|
| 入力が必要な集中タスク | `Dialog` |
| 破壊的操作の確認 | `AlertDialog` |
| 詳細やフィルター用のサイドパネル | `Sheet` |
| モバイル優先の下部パネル | `Drawer` |
| hover 時の簡易情報 | `HoverCard` |
| click 時の小さな文脈コンテンツ | `Popover` |

---

## Dialog、Sheet、Drawer には必ず Title が必要

`DialogTitle`、`SheetTitle`、`DrawerTitle` はアクセシビリティ上必須です。視覚的に隠す場合は `className="sr-only"` を使います。

```tsx
<DialogContent>
  <DialogHeader>
    <DialogTitle>Edit Profile</DialogTitle>
    <DialogDescription>Update your profile.</DialogDescription>
  </DialogHeader>
  ...
</DialogContent>
```

---

## Card の構造

完全な構成を使い、すべてを `CardContent` に押し込まないでください。

```tsx
<Card>
  <CardHeader>
    <CardTitle>Team Members</CardTitle>
    <CardDescription>Manage your team.</CardDescription>
  </CardHeader>
  <CardContent>...</CardContent>
  <CardFooter>
    <Button>Invite</Button>
  </CardFooter>
</Card>
```

---

## Button に isPending / isLoading prop はない

`Spinner` + `data-icon` + `disabled` で構成します。

```tsx
<Button disabled>
  <Spinner data-icon="inline-start" />
  Saving...
</Button>
```

---

## TabsTrigger は必ず TabsList 内に置く

`TabsTrigger` を `Tabs` の直下に直接レンダリングしてはいけません。必ず `TabsList` でラップします。

```tsx
<Tabs defaultValue="account">
  <TabsList>
    <TabsTrigger value="account">Account</TabsTrigger>
    <TabsTrigger value="password">Password</TabsTrigger>
  </TabsList>
  <TabsContent value="account">...</TabsContent>
</Tabs>
```

---

## Avatar には必ず AvatarFallback が必要

画像の読み込みに失敗した場合のため、必ず `AvatarFallback` を含めます。

```tsx
<Avatar>
  <AvatarImage src="/avatar.png" alt="User" />
  <AvatarFallback>JD</AvatarFallback>
</Avatar>
```

---

## 独自マークアップではなく既存コンポーネントを使う

| 代わりに避けるもの | 使うもの |
|---|---|
| `<hr>` または `<div className="border-t">` | `<Separator />` |
| スタイル付き div を使った `<div className="animate-pulse">` | `<Skeleton className="h-4 w-3/4" />` |
| `<span className="rounded-full bg-green-100 ...">` | `<Badge variant="secondary">` |
