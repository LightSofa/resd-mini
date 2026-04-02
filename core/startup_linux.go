//go:build linux

package core

import (
	"os"
	"os/signal"
	"syscall"
)

func (a *App) Startup() {
	go httpServerOnce.run()

	if globalConfig.AutoProxy {
		if err := a.OpenSystemProxy(); err != nil {
			globalLogger.Esg(err, "auto open proxy failed")
		}
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	a.shutdown()
}

func (a *App) updateProxyMenuTitle() {}

func (a *App) shutdown() {
	_ = a.UnsetSystemProxy()
	globalLogger.Close()
}
