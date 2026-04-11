// nolint testpackage
package gui

import (
	"errors"
	"testing"

	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"proxy-checker/internal/mocks"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestApplySystemProxy_Error(t *testing.T) {
	testApp := test.NewTempApp(t)
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())

	mockRunner := &mocks.MockPipelineRunner{}
	mockSysProxy := &mocks.MockSystemProxyManager{}

	app := NewAppGUI(testApp, cfg, logger, "dev", mockRunner, mockSysProxy)
	app.ShowMainScreen()

	expectedErr := errors.New("gsettings error")
	mockSysProxy.MockSetProxy = func(_, _, _ string) error {
		return expectedErr
	}

	app.applySystemProxy("1.1.1.1", "8080", "socks5")

	assert.False(t, app.switchProxy.Checked, "System proxy switch must remain unchecked on error")
	assert.Equal(t, -1, app.highlightedRow, "No row must be highlighted on error")
}
