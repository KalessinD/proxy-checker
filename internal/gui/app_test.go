package gui_test

import (
	"os"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/config"
	"proxy-checker/internal/gui"
	"proxy-checker/internal/services"
	"proxy-checker/internal/sysproxy"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	_ = i18n.Init("en")
	os.Exit(m.Run())
}

func TestNewAppGUI_Base(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())

	appGUI := gui.NewAppGUI(testApp, cfg, logger, "dev", services.NewDefaultPipelineRunner(), sysproxy.NewSystemProxyManager())

	assert.NotNil(t, appGUI)
}

func TestNewAppGUI_Run(_ *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())

	appGUI := gui.NewAppGUI(testApp, cfg, logger, "dev", services.NewDefaultPipelineRunner(), sysproxy.NewSystemProxyManager())

	appGUI.Run()
}

func TestNewAppGUI_SingleScreen(_ *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())

	app := gui.NewAppGUI(testApp, cfg, logger, "dev", services.NewDefaultPipelineRunner(), sysproxy.NewSystemProxyManager())

	app.ShowSingleCheckScreen()
}

func TestNewAppGUI_SettingseScreen(_ *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())

	app := gui.NewAppGUI(testApp, cfg, logger, "dev", services.NewDefaultPipelineRunner(), sysproxy.NewSystemProxyManager())

	app.ShowSettingsScreen()
}

func TestNewAppGUI_AboutScreen(_ *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())

	app := gui.NewAppGUI(testApp, cfg, logger, "dev", services.NewDefaultPipelineRunner(), sysproxy.NewSystemProxyManager())

	app.ShowAboutDialog()
}

func TestNewAppGUI_MainScreen(_ *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())

	app := gui.NewAppGUI(testApp, cfg, logger, "dev", services.NewDefaultPipelineRunner(), sysproxy.NewSystemProxyManager())

	app.ShowMainScreen()
}
