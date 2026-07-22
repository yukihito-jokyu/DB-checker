package main

import "context"

type App struct {
	ctx context.Context
}

// アプリ生成
func NewApp() *App {
	return &App{}
}

// 起動コンテキスト保持
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}
