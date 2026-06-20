---
name: github-issue-labeling
description: GitHub Issue の作成・更新時に、Wails アプリケーション向けのラベルを判断・付与・更新するためのルール。
---

# GitHub Issue Labeling Skill

この Skill は、Wails アプリケーション開発における GitHub Issue のラベル運用ルールを定義する。

目的は以下の3つ。

1. 人間が Issue 一覧から現在の状態を把握しやすくする。
2. AI 駆動開発で AI が Issue の状態を動的に更新できるようにする。
3. ラベルの意味を端的に保ち、複数ラベルの組み合わせで Issue を表現する。

## 基本方針

すべての Issue には、原則として以下の4軸のラベルを必ず付ける。

- `type:*`
- `status:*`
- `area:*`
- `priority:*`

例外 Issue は作らない。

管理用 Issue、調査 Issue、リリース準備 Issue、ドキュメント Issue であっても、必ず4軸すべてを付ける。

## 排他ルール

以下のラベル軸は、1つの Issue に対して必ず1つだけ付ける。

- `type:*`
- `status:*`
- `priority:*`

以下のラベル軸は、1つ以上を必ず付け、複数付与を許可する。

- `area:*`

正しい例:

type:feature
status:todo
priority:high
area:frontend
area:backend
