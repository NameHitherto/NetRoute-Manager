package main

import (
	"embed"
	"log"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"NetRoute-Manager/internal/dns"
	"NetRoute-Manager/internal/models"
	"NetRoute-Manager/internal/route"
	"NetRoute-Manager/internal/service"
	"NetRoute-Manager/internal/store"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var trayIcon []byte

func init() {
	// 注册自定义事件,binding 生成器据此产出强类型 TS API。
	application.RegisterEvent[models.LogEntry]("service:log")
}

func main() {
	// 领域层:持久化存储与核心路由引擎(与 wails 版本无关,原样沿用)。
	s := store.NewOrPanic()
	engine := service.New(dns.New(), route.NewManager(), s.RootDir())

	// quitting 托盘"退出"触发的真正退出标记,用于放行窗口关闭拦截。
	var quitting atomic.Bool

	// window 在 application.New 之后创建,单实例回调通过闭包引用。
	var window *application.WebviewWindow

	app := application.New(application.Options{
		Name:        "NetRoute-Manager",
		Description: "路由规则管理工具:将域名解析结果以临时主机路由绑定到指定网卡",
		Services: []application.Service{
			application.NewService(NewRoutesService(s)),
			application.NewService(NewNicsService()),
			application.NewService(NewEngineService(s, engine)),
			application.NewService(NewSettingsService(s)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.namehitherto.netroute-manager",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				if window != nil {
					window.Restore()
					window.Focus()
				}
			},
		},
		// 应用退出前:停止路由服务,清理全部临时路由,恢复系统默认。
		OnShutdown: func() {
			_ = engine.Stop()
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "NetRoute-Manager",
		Width:            1024,
		Height:           768,
		MinWidth:         800,
		MinHeight:        600,
		Frameless:        true,
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	// 关闭拦截:minToTray 开启(默认)时拦截关闭并隐藏到托盘;
	// 关闭时直接退出;托盘"退出"置位 quitting 后始终放行。
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if quitting.Load() {
			return
		}
		settings, err := s.LoadSettings()
		if err != nil || settings.MinToTray {
			window.Hide()
			e.Cancel()
		}
	})

	// 引擎日志通过事件推送到前端,由日志面板订阅展示。
	var logSeq atomic.Int32
	engine.SetLogFunc(func(level models.LogLevel, text string) {
		app.Event.Emit("service:log", models.LogEntry{
			ID:   int(logSeq.Add(1)),
			Time: time.Now().Format("15:04:05"),
			Type: level,
			Text: text,
		})
	})

	// 系统托盘(wails3 原生 API,替代第三方 getlantern/systray)。
	menu := app.NewMenu()
	menu.Add("显示主窗口").OnClick(func(ctx *application.Context) {
		window.Show()
		window.UnMinimise()
		window.Focus()
	})
	menu.Add("退出").OnClick(func(ctx *application.Context) {
		quitting.Store(true)
		app.Quit()
	})
	tray := app.SystemTray.New()
	tray.SetIcon(trayIcon)
	tray.SetTooltip("NetRoute-Manager")
	tray.SetMenu(menu)
	// 与旧版 getlantern 在 Windows 上"左键也弹菜单"的行为对齐。
	tray.OnClick(func() {
		tray.OpenMenu()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
