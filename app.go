package main

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"NetRoute-Manager/internal/dns"
	"NetRoute-Manager/internal/models"
	"NetRoute-Manager/internal/route"
	"NetRoute-Manager/internal/service"
	"NetRoute-Manager/internal/store"
)

// App 应用主结构,持有领域数据持久化层与核心路由服务引擎。
type App struct {
	ctx      context.Context
	store    *store.Store
	engine   *service.Engine
	logSeq   atomic.Int32 // 运行日志自增 ID,保证事件日志唯一
	quitting atomic.Bool  // 托盘"退出"触发的真正退出标记,放行 OnBeforeClose
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

// beforeClose 覆写原生标题栏关闭事件:非主动退出时拦截关闭,将窗口隐藏到系统托盘;
// 仅当托盘菜单"退出"置位 quitting 后才放行,由 runtime.Quit 走正常退出流程。
// 注意:关闭行为当前固定为隐藏到托盘,AppSettings.MinToTray 配置字段尚未接入本逻辑。
func (a *App) beforeClose(ctx context.Context) bool {
	if a.quitting.Load() {
		return false // 放行:托盘"退出"触发的真正退出
	}
	runtime.WindowHide(ctx)
	return true // 拦截关闭,转入托盘驻留
}

// showFromTray 从系统托盘恢复主窗口并取消最小化(托盘菜单"显示主窗口")。
// 私有方法:仅作为托盘回调,不暴露给前端绑定。
func (a *App) showFromTray() {
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
}

// quitFromTray 从系统托盘退出应用(托盘菜单"退出"):先移除托盘图标,
// 再置退出标记并调用 runtime.Quit,确保 OnShutdown(engine.Stop 清理路由)仍被执行。
func (a *App) quitFromTray() {
	systray.Quit()
	a.quitting.Store(true)
	runtime.Quit(a.ctx)
}
