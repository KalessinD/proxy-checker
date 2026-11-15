// nolint testpackage
package gui

import (
	"errors"
	"testing"

	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"proxy-checker/internal/mocks"
	"proxy-checker/internal/services"
	"proxy-checker/internal/sysproxy"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

// newMonitorAdapterFixture builds an AppGUI with a mock system proxy manager
// and the adapter under test, mirroring screen_main_logic_test.go setup.
func newMonitorAdapterFixture(t *testing.T) (*AppGUI, *mocks.MockSystemProxyManager) {
	t.Helper()
	testApp := test.NewTempApp(t)
	t.Cleanup(testApp.Quit)

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())

	ctrl := gomock.NewController(t)
	mockSysProxy := mocks.NewMockSystemProxyManager(ctrl)
	// Supported platform: the switchProxy OnChanged callback otherwise
	// resets the checkbox, which masks the apply path under test.
	mockSysProxy.EXPECT().IsSupported().Return(true).AnyTimes()
	mockSysProxy.EXPECT().GetMode().Return("", nil).AnyTimes()
	mockSysProxy.EXPECT().GetActiveProxy().Return("", "", nil).AnyTimes()
	mockSysProxy.EXPECT().SetMode(sysproxy.ProxyModeManual).Return(nil).AnyTimes()

	deps := newTestDependencies(testApp, cfg, logger)
	deps.SysProxyManager = mockSysProxy
	app := NewAppGUI(deps)
	app.ShowMainScreen()
	return app, mockSysProxy
}

func TestGuiActiveProxyStore_GetActiveProxy(t *testing.T) {
	app, _ := newMonitorAdapterFixture(t)
	app.proxyItems = []*ProxyItemWrapper{
		{Host: "1.1.1.1", Port: "1080", Country: "NL", Type: common.ProxySOCKS5},
	}
	store := newActiveProxyStoreAdapter(app)

	// Nothing applied at the OS level yet.
	_, found := store.GetActiveProxy(t.Context())
	assert.False(t, found, "no active proxy must be reported before apply")

	app.activeProxyHost = "1.1.1.1"
	app.activeProxyPort = "1080"

	item, found := store.GetActiveProxy(t.Context())
	require.True(t, found)
	assert.Equal(t, "1.1.1.1", item.Host)
	assert.Equal(t, "1080", item.Port)
	assert.Equal(t, "NL", item.Country, "country must come from the result table")
	assert.Equal(t, common.ProxySOCKS5, item.Type)
}

func TestGuiActiveProxyStore_ApplyProxy(t *testing.T) {
	app, mockSysProxy := newMonitorAdapterFixture(t)
	store := newActiveProxyStoreAdapter(app)

	mockSysProxy.EXPECT().
		SetProxy("2.2.2.2", "1080", "socks5").
		Return(nil).
		Times(1)

	err := store.ApplyProxy(t.Context(), services.ProxyItem{
		Host: "2.2.2.2",
		Port: "1080",
		Type: common.ProxySOCKS5,
	})
	require.NoError(t, err)
	assert.Equal(t, "2.2.2.2", app.activeProxyHost, "apply must update the cached active proxy")
	assert.True(t, app.switchProxy.Checked)
}

func TestGuiActiveProxyStore_ApplyProxyError(t *testing.T) {
	app, mockSysProxy := newMonitorAdapterFixture(t)
	store := newActiveProxyStoreAdapter(app)

	expectedErr := errors.New("gsettings error")
	mockSysProxy.EXPECT().
		SetProxy("2.2.2.2", "1080", "socks5").
		Return(expectedErr).
		Times(1)

	err := store.ApplyProxy(t.Context(), services.ProxyItem{
		Host: "2.2.2.2",
		Port: "1080",
		Type: common.ProxySOCKS5,
	})
	require.ErrorIs(t, err, expectedErr, "apply error must reach the monitor")
	assert.Empty(t, app.activeProxyHost, "active proxy must not change on error")
}

func TestGuiMonitorNotifier_LogsAndNotifies(t *testing.T) {
	app, _ := newMonitorAdapterFixture(t)
	notifier := newMonitorNotifierAdapter(app)

	notifier.ProxyDown(services.ProxyItem{Host: "1.1.1.1", Port: "1080"})
	notifier.SwitchedTo(services.ProxyItem{Host: "2.2.2.2", Port: "1080", Type: common.ProxySOCKS5})
	notifier.SwitchFailed("candidate pool is empty")

	require.NotEmpty(t, app.logRichText.Segments)
	last := app.logRichText.Segments[len(app.logRichText.Segments)-1].(*widget.TextSegment)
	assert.Contains(t, last.Text, "candidate pool is empty")
}
