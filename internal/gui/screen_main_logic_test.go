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
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestApplySystemProxy_Error(t *testing.T) {
	testApp := test.NewTempApp(t)
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())

	ctrl := gomock.NewController(t)
	mockRunner := mocks.NewMockPipelineRunner(ctrl)
	mockSysProxy := mocks.NewMockSystemProxyManager(ctrl)

	// Defaults for the construction and screen paths: proxy management is not
	// supported on this platform, every mode lookup reports no active mode.
	mockSysProxy.EXPECT().IsSupported().Return(false).AnyTimes()
	mockSysProxy.EXPECT().GetMode().Return("", nil).AnyTimes()

	expectedErr := errors.New("gsettings error")
	mockSysProxy.EXPECT().
		SetProxy("1.1.1.1", "8080", "socks5").
		Return(expectedErr).
		Times(1)

	app := NewAppGUI(testApp, cfg, logger, "dev", mockRunner, mockSysProxy)
	app.ShowMainScreen()

	app.applySystemProxy("1.1.1.1", "8080", "socks5")

	assert.False(t, app.switchProxy.Checked, "System proxy switch must remain unchecked on error")
	assert.Equal(t, -1, app.highlightedRow, "No row must be highlighted on error")
}
