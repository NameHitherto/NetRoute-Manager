package main

import (
	_ "embed"

	"github.com/getlantern/systray"
)

//go:embed build/windows/icon.ico
var trayIconData []byte

// trayController 托盘菜单回调接口，由 App 实现，避免 tray 模块反向依赖具体类型。
type trayController interface {
	// showFromTray 从系统托盘恢复并聚焦主窗口。
	showFromTray()
	// quitFromTray 从系统托盘退出应用（走 wails 正常退出流程）。
	quitFromTray()
}

// runTray 启动系统托盘,阻塞直至进程退出,须在独立 goroutine 中调用。
// onReady 中注册图标、tooltip 与菜单项;菜单"显示主窗口"恢复窗口,
// "退出"经 App.quitFromTray 走 runtime.Quit 正常退出(保证 OnShutdown 清理路由)。
func runTray(c trayController) {
	systray.Run(func() {
		systray.SetIcon(trayIconData)
		systray.SetTooltip("NetRoute-Manager")
		systray.SetTitle("NetRoute-Manager")

		showItem := systray.AddMenuItem("显示主窗口", "显示 NetRoute-Manager 主窗口")
		quitItem := systray.AddMenuItem("退出", "退出 NetRoute-Manager")

		go func() {
			for {
				select {
				case <-showItem.ClickedCh:
					c.showFromTray()
				case <-quitItem.ClickedCh:
					c.quitFromTray()
					return
				}
			}
		}()
	}, func() {
		// 托盘已退出(进程即将结束),无需额外收尾
	})
}
