// Package app contains the composition root: it assembles all application
// dependencies from configuration and launches the configured entry points.
package app

import (
	"proxy-checker/internal/cache"
	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"proxy-checker/internal/gui"
	"proxy-checker/internal/services"
	"proxy-checker/internal/sysproxy"

	fyneapp "fyne.io/fyne/v2/app"
)

// RunGUI assembles the GUI dependencies and launches the GUI application entry point.
func RunGUI(cfg *config.Config, logger common.LoggerInterface, version string) {
	deps := gui.AppDependencies{
		FyneApp:         fyneapp.NewWithID(common.AppName),
		Cfg:             cfg,
		Logger:          logger,
		Version:         version,
		PipelineRunner:  services.NewDefaultPipelineRunner(),
		SysProxyManager: sysproxy.NewSystemProxyManager(),
		Cache:           cache.NewFileStorage(logger),
	}

	gui.Run(deps)
}
