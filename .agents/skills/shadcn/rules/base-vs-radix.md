# Base と Radix

`base` と `radix` の API 差分です。`npx shadcn@latest info` の `base` フィールドを確認してください。

## 目次

- Composition: asChild と render
- Button / trigger を button 以外の要素として使う
- Select（items prop、placeholder、positioning、multiple、object values）
- ToggleGroup（type と multiple）
- Slider（scalar と array）
- Accordion（type と defaultValue）

---

## Composition: asChild (radix) と render (base)

Radix はデフォルト要素を置き換えるために `asChild` を使います。Base は `render` を使います。trigger を余分な要素でラップしないでください。

**誤り:**

```tsx
<DialogTrigger>
  <div>
    <Button>Open</Button>
  </div>
</DialogTrigger>
```

**正しい (radix):**

```tsx
<DialogTrigger asChild>
  <Button>Open</Button>
</DialogTrigger>
```

**正しい (base):**

```tsx
<DialogTrigger render={<Button />}>Open</DialogTrigger>
```

これはすべての trigger / close コンポーネントに適用されます。`DialogTrigger`、`SheetTrigger`、`AlertDialogTrigger`、`DropdownMenuTrigger`、`PopoverTrigger`、`TooltipTrigger`、`CollapsibleTrigger`、`DialogClose`、`SheetClose`、`NavigationMenuLink`、`BreadcrumbLink`、`SidebarMenuButton`、`Badge`、`Item`。

---

## Button / trigger を button 以外の要素として使う（base のみ）

`render` が要素を button 以外（`<a>`、`<span>`）に変える場合は、`nativeButton={false}` を追加します。

**誤り (base):** `nativeButton={false}` がありません。

```tsx
<Button render={<a href="/docs" />}>Read the docs</Button>
```

**正しい (base):**

```tsx
<Button render={<a href="/docs" />} nativeButton={false}>
  Read the docs
</Button>
```

**正しい (radix):**

```tsx
<Button asChild>
  <a href="/docs">Read the docs</a>
</Button>
```

`render` が `Button` ではない trigger でも同様です。

```tsx
// base.
<PopoverTrigger render={<InputGroupAddon />} nativeButton={false}>
  Pick date
</PopoverTrigger>
```

---

## Select

**items prop (base のみ)。** Base では root に `items` prop が必要です。Radix は inline JSX のみを使います。

**誤り (base):**

```tsx
<Select>
  <SelectTrigger><SelectValue placeholder="Select a fruit" /></SelectTrigger>
</Select>
```

**正しい (base):**

```tsx
const items = [
  { label: "Select a fruit", value: null },
  { label: "Apple", value: "apple" },
  { label: "Banana", value: "banana" },
]

<Select items={items}>
  <SelectTrigger>
    <SelectValue />
  </SelectTrigger>
  <SelectContent>
    <SelectGroup>
      {items.map((item) => (
        <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>
      ))}
    </SelectGroup>
  </SelectContent>
</Select>
```

**正しい (radix):**

```tsx
<Select>
  <SelectTrigger>
    <SelectValue placeholder="Select a fruit" />
  </SelectTrigger>
  <SelectContent>
    <SelectGroup>
      <SelectItem value="apple">Apple</SelectItem>
      <SelectItem value="banana">Banana</SelectItem>
    </SelectGroup>
  </SelectContent>
</Select>
```

**Placeholder。** Base は items array 内の `{ value: null }` item を使います。Radix は `<SelectValue placeholder="...">` を使います。

**Content positioning。** Base は `alignItemWithTrigger` を使います。Radix は `position` を使います。

```tsx
// base.
<SelectContent alignItemWithTrigger={false} side="bottom">

// radix.
<SelectContent position="popper">
```

---

## Select — multiple selection と object values（base のみ）

Base は `multiple`、`SelectValue` の render-function children、`itemToStringValue` を使った object values をサポートします。Radix は string values の単一選択のみです。

**正しい (base — multiple selection):**

```tsx
<Select items={items} multiple defaultValue={[]}>
  <SelectTrigger>
    <SelectValue>
      {(value: string[]) => value.length === 0 ? "Select fruits" : `${value.length} selected`}
    </SelectValue>
  </SelectTrigger>
  ...
</Select>
```

**正しい (base — object values):**

```tsx
<Select defaultValue={plans[0]} itemToStringValue={(plan) => plan.name}>
  <SelectTrigger>
    <SelectValue>{(value) => value.name}</SelectValue>
  </SelectTrigger>
  ...
</Select>
```

---

## ToggleGroup

Base は `multiple` boolean prop を使います。Radix は `type="single"` または `type="multiple"` を使います。

**誤り (base):**

```tsx
<ToggleGroup type="single" defaultValue="daily">
  <ToggleGroupItem value="daily">Daily</ToggleGroupItem>
</ToggleGroup>
```

**正しい (base):**

```tsx
// Single（prop 不要）、defaultValue は常に array。
<ToggleGroup defaultValue={["daily"]} spacing={2}>
  <ToggleGroupItem value="daily">Daily</ToggleGroupItem>
  <ToggleGroupItem value="weekly">Weekly</ToggleGroupItem>
</ToggleGroup>

// Multi-selection.
<ToggleGroup multiple>
  <ToggleGroupItem value="bold">Bold</ToggleGroupItem>
  <ToggleGroupItem value="italic">Italic</ToggleGroupItem>
</ToggleGroup>
```

**正しい (radix):**

```tsx
// Single、defaultValue は string。
<ToggleGroup type="single" defaultValue="daily" spacing={2}>
  <ToggleGroupItem value="daily">Daily</ToggleGroupItem>
  <ToggleGroupItem value="weekly">Weekly</ToggleGroupItem>
</ToggleGroup>

// Multi-selection.
<ToggleGroup type="multiple">
  <ToggleGroupItem value="bold">Bold</ToggleGroupItem>
  <ToggleGroupItem value="italic">Italic</ToggleGroupItem>
</ToggleGroup>
```

**Controlled single value:**

```tsx
// base — array に包む / array から取り出す。
const [value, setValue] = React.useState("normal")
<ToggleGroup value={[value]} onValueChange={(v) => setValue(v[0])}>

// radix — plain string。
const [value, setValue] = React.useState("normal")
<ToggleGroup type="single" value={value} onValueChange={setValue}>
```

---

## Slider

Base は single thumb に plain number を受け取ります。Radix は常に array が必要です。

**誤り (base):**

```tsx
<Slider defaultValue={[50]} max={100} step={1} />
```

**正しい (base):**

```tsx
<Slider defaultValue={50} max={100} step={1} />
```

**正しい (radix):**

```tsx
<Slider defaultValue={[50]} max={100} step={1} />
```

range slider ではどちらも array を使います。base の controlled `onValueChange` では cast が必要になる場合があります。

```tsx
// base.
const [value, setValue] = React.useState([0.3, 0.7])
<Slider value={value} onValueChange={(v) => setValue(v as number[])} />

// radix.
const [value, setValue] = React.useState([0.3, 0.7])
<Slider value={value} onValueChange={setValue} />
```

---

## Accordion

Radix では `type="single"` または `type="multiple"` が必要で、`collapsible` をサポートします。`defaultValue` は string です。Base は `type` prop を使わず、`multiple` boolean を使い、`defaultValue` は常に array です。

**誤り (base):**

```tsx
<Accordion type="single" collapsible defaultValue="item-1">
  <AccordionItem value="item-1">...</AccordionItem>
</Accordion>
```

**正しい (base):**

```tsx
<Accordion defaultValue={["item-1"]}>
  <AccordionItem value="item-1">...</AccordionItem>
</Accordion>

// Multi-select.
<Accordion multiple defaultValue={["item-1", "item-2"]}>
  <AccordionItem value="item-1">...</AccordionItem>
  <AccordionItem value="item-2">...</AccordionItem>
</Accordion>
```

**正しい (radix):**

```tsx
<Accordion type="single" collapsible defaultValue="item-1">
  <AccordionItem value="item-1">...</AccordionItem>
</Accordion>
```
