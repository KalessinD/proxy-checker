// nolint testpackage
package gui

import (
	"proxy-checker/internal/cache"
	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"proxy-checker/internal/services"
	"proxy-checker/internal/sysproxy"

	"fyne.io/fyne/v2"
)

// newTestDependencies assembles AppDependencies with default production
// implementations for GUI white-box tests. Callers override individual
// fields when mocks are required.
func newTestDependencies(
	fyneApp fyne.App,
	testCfg *config.Config,
	testLogger common.LoggerInterface,
) AppDependencies {
	return AppDependencies{
		FyneApp:         fyneApp,
		Cfg:             testCfg,
		Logger:          testLogger,
		Version:         "dev",
		PipelineRunner:  services.NewDefaultPipelineRunner(),
		SysProxyManager: sysproxy.NewSystemProxyManager(),
		Cache:           cache.NewFileStorage(testLogger),
	}
}
