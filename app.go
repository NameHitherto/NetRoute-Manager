package main

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"NetRoute-Manager/internal/dns"
	"NetRoute-Manager/internal/models"
	"NetRoute-Manager/internal/route"
	"NetRoute-Manager/internal/service"
	"NetRoute-Manager/internal/store"
)

// App 应用主结构,持有领域数据持久化层与核心路由服务引擎。
type App struct {
	ctx    context.Context
	store  *store.Store
	engine *service.Engine
	logSeq atomic.Int32 // 运行日志自增 ID,保证事件日志唯一
}

// NewApp creates a new App application struct
func NewApp() *App {
	s := store.NewOrPanic()
	return &App{
		store:  s,
		engine: service.New(dns.New(), route.NewManager(), s.RootDir()),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// 引擎日志通过 wails 事件推送到前端,由日志面板订阅展示
	a.engine.SetLogFunc(func(level models.LogLevel, text string) {
		entry := models.LogEntry{
			ID:   int(a.logSeq.Add(1)),
			Time: time.Now().Format("15:04:05"),
			Type: level,
			Text: text,
		}
		runtime.EventsEmit(ctx, "service:log", entry)
	})
}

// shutdown 应用退出前调用:停止路由服务,清理全部临时路由,恢复系统默认。
func (a *App) shutdown(ctx context.Context) {
	_ = a.engine.Stop()
}
