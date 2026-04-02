//go:build !linux

package core

import (
	"fyne.io/systray"
	"os"
	"time"
)

var (
	menuQuit  *systray.MenuItem
	menuProxy *systray.MenuItem
	menuOpen  *systray.MenuItem
)

func (a *App) Startup() {
	systray.Run(a.onReady, a.shutdown)
}

func (a *App) onReady() {
	systray.SetIcon(a.getIcon())
	systray.SetTitle("")
	systray.SetTooltip(a.Description)

	menuProxy = systray.AddMenuItem("Open proxy", "Set up system proxy")
	menuOpen = systray.AddMenuItem("Open panel", "Open the management panel")
	menuQuit = systray.AddMenuItem("Exit", "Exit the application")

	go httpServerOnce.run()

	time.AfterFunc(200*time.Millisecond, func() {
		_ = OpenBrowser("http://127.0.0.1:" + globalConfig.Port)
	})

	go func() {
		for {
			select {
			case <-menuOpen.ClickedCh:
				_ = OpenBrowser("http://127.0.0.1:" + globalConfig.Port)
			case <-menuProxy.ClickedCh:
				if appOnce.IsProxy {
					_ = a.UnsetSystemProxy()
				} else {
					_ = a.OpenSystemProxy()
				}
				httpServerOnce.send("updateProxyStatus", map[string]interface{}{
					"value": appOnce.IsProxy,
				})
			case <-menuQuit.ClickedCh:
				systray.Quit()
				os.Exit(0)
			}
		}
	}()
}

func (a *App) updateProxyMenuTitle() {
	if menuProxy == nil {
		return
	}
	if a.IsProxy {
		menuProxy.SetTitle("Close proxy")
		return
	}
	menuProxy.SetTitle("Open proxy")
}

func (a *App) shutdown() {
	_ = a.UnsetSystemProxy()
	globalLogger.Close()
}
