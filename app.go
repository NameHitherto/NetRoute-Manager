package main

import (
	"context"

	"NetRoute-Manager/internal/store"
)

// App 应用主结构,持有领域数据持久化层。
type App struct {
	ctx   context.Context
	store *store.Store
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{store: store.NewOrPanic()}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}
