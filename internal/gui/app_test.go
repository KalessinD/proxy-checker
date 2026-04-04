package gui_test

import (
	"os"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/config"
	"proxy-checker/internal/gui"
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

	appGUI := gui.NewAppGUI(testApp, cfg, logger, "dev")

	assert.NotNil(t, appGUI)
}

func TestNewAppGUI_MainScreen(_ *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())

	appGUI := gui.NewAppGUI(testApp, cfg, logger, "dev")

	appGUI.Run()
}

func TestNewAppGUI_SingleScreen(_ *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())

	gui.NewAppGUI(testApp, cfg, logger, "dev").ShowSettingsScreen()
}

func TestNewAppGUI_SettingseScreen(_ *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())

	gui.NewAppGUI(testApp, cfg, logger, "dev").ShowSettingsScreen()
}

func TestNewAppGUI_AboutScreen(_ *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())

	gui.NewAppGUI(testApp, cfg, logger, "dev").ShowAboutDialog()
}
