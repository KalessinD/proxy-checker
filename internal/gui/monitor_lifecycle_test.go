// nolint testpackage
package gui

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"proxy-checker/internal/mocks"
	"proxy-checker/internal/services"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func newMonitorLifecycleFixture(t *testing.T) (*AppGUI, *mocks.MockSystemProxyManager) {
	t.Helper()
	testApp := test.NewTempApp(t)
	t.Cleanup(testApp.Quit)

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())

	ctrl := gomock.NewController(t)
	mockSysProxy := mocks.NewMockSystemProxyManager(ctrl)
	mockSysProxy.EXPECT().IsSupported().Return(true).AnyTimes()
	mockSysProxy.EXPECT().GetMode().Return("", nil).AnyTimes()
	mockSysProxy.EXPECT().GetActiveProxy().Return("", "", nil).AnyTimes()
	mockSysProxy.EXPECT().SetMode(gomock.Any()).Return(nil).AnyTimes()

	deps := newTestDependencies(testApp, cfg, logger)
	deps.SysProxyManager = mockSysProxy
	app := NewAppGUI(deps)
	app.ShowMainScreen()
	return app, mockSysProxy
}

// mockProxyChecker routes to the two mock methods through one object.
func newMockChecker(ctrl *gomock.Controller) *mocks.MockProxyChecker {
	return mocks.NewMockProxyChecker(ctrl)
}

func TestMonitorLifecycle_StartStopIsIdempotent(t *testing.T) {
	app, _ := newMonitorLifecycleFixture(t)
	ctrl := gomock.NewController(t)

	app.startProxyMonitorWith(newMockChecker(ctrl), &recordingNotifier{})
	require.True(t, app.monitorRunning(), "monitor must be running after start")

	app.startProxyMonitorWith(newMockChecker(ctrl), &recordingNotifier{}) // second start is a no-op

	app.stopProxyMonitor()
	require.False(t, app.monitorRunning())

	app.stopProxyMonitor() // second stop is a no-op
	assert.False(t, app.monitorRunning())
}

func TestMonitorLifecycle_UpdatePoolSnapshotAndFreshness(t *testing.T) {
	app, _ := newMonitorLifecycleFixture(t)
	ctrl := gomock.NewController(t)
	app.startProxyMonitorWith(newMockChecker(ctrl), &recordingNotifier{})
	t.Cleanup(app.stopProxyMonitor)

	app.proxyItems = []*ProxyItemWrapper{
		{Host: "1.1.1.1", Port: "1080", Country: "NL", Type: common.ProxySOCKS5, Source: string(common.SourceProxyMania)},
		{Host: "2.2.2.2", Port: "8080", Country: "DE", Type: common.ProxySOCKS4, Source: string(common.SourceProxyMania)},
	}
	app.cacheExpiryMutex.Lock()
	app.cacheExpiry = map[common.Source]time.Time{
		common.SourceProxyMania: time.Now().Add(time.Hour),
	}
	app.cacheExpiryMutex.Unlock()

	app.updateMonitorPool()

	// Snapshot contents must match the table.
	snapshot := app.monitorPoolSnapshot()
	require.Len(t, snapshot, 2)
	assert.Equal(t, "1.1.1.1", snapshot[0].Host)
	assert.Equal(t, "NL", snapshot[0].Country)
	assert.Equal(t, common.SourceProxyMania, snapshot[1].Source)

	// Unexpired cache -> fresh pool.
	assert.True(t, app.monitorPoolFresh())

	// Expired cache -> stale pool (never used for switching).
	app.cacheExpiryMutex.Lock()
	app.cacheExpiry = map[common.Source]time.Time{
		common.SourceProxyMania: time.Now().Add(-time.Minute),
	}
	app.cacheExpiryMutex.Unlock()
	assert.False(t, app.monitorPoolFresh())

	// Unknown source in the table -> not fresh either.
	app.cacheExpiryMutex.Lock()
	app.cacheExpiry = map[common.Source]time.Time{}
	app.cacheExpiryMutex.Unlock()
	assert.False(t, app.monitorPoolFresh())
}

// recordingNotifier is a race-safe services.MonitorNotifier fake for tests.
type recordingNotifier struct {
	mu       sync.Mutex
	healthy  []services.ProxyItem
	down     []services.ProxyItem
	switched []services.ProxyItem
	failed   []string
}

func (n *recordingNotifier) Healthy(item services.ProxyItem) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.healthy = append(n.healthy, item)
}

func (n *recordingNotifier) ProxyDown(item services.ProxyItem) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.down = append(n.down, item)
}

func (n *recordingNotifier) SwitchedTo(item services.ProxyItem) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.switched = append(n.switched, item)
}

func (n *recordingNotifier) SwitchFailed(reason string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.failed = append(n.failed, reason)
}

// waitUntilNotifier polls the predicate against the recording notifier until
// it passes or the timeout expires. The notifier mutex establishes a
// happens-before edge, making the subsequent AppGUI state reads race-free.
func waitUntilNotifier(t *testing.T, notifier *recordingNotifier, predicate func(*recordingNotifier) bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		notifier.mu.Lock()
		ok := predicate(notifier)
		notifier.mu.Unlock()
		if ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("monitor condition not met within timeout")
}

func TestMonitorLifecycle_FullTickSwitchesActiveProxy(t *testing.T) {
	app, mockSysProxy := newMonitorLifecycleFixture(t)
	ctrl := gomock.NewController(t)
	checker := newMockChecker(ctrl)
	notifier := &recordingNotifier{}

	// The active proxy fails the health check.
	checker.EXPECT().
		CheckProxy(gomock.Any(), "9.9.9.9:1080", gomock.Any()).
		Return(services.Result{Error: errors.New("connection refused")}).
		Times(1)

	// The batch check passes the only candidate.
	checker.EXPECT().
		CheckBatch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ []*services.ProxyItem, _ *services.CheckParams, _ services.ProxyVerifier, _ func(int, int)) []*services.ProxyItemFull {
			return []*services.ProxyItemFull{
				{ProxyItem: services.ProxyItem{
					Host:    "8.8.8.8",
					Port:    "1080",
					Country: "NL",
					Type:    common.ProxySOCKS5,
				}},
			}
		}).
		Times(1)

	// Applying the winner writes the system proxy.
	mockSysProxy.EXPECT().
		SetProxy("8.8.8.8", "1080", string(common.ProxySOCKS5)).
		Return(nil).
		Times(1)

	app.startProxyMonitorWith(checker, notifier)
	t.Cleanup(app.stopProxyMonitor)

	app.proxyItems = []*ProxyItemWrapper{
		{Host: "8.8.8.8", Port: "1080", Country: "NL", Type: common.ProxySOCKS5, Source: string(common.SourceProxyMania)},
	}
	app.cacheExpiryMutex.Lock()
	app.cacheExpiry = map[common.Source]time.Time{
		common.SourceProxyMania: time.Now().Add(time.Hour),
	}
	app.cacheExpiryMutex.Unlock()
	app.updateMonitorPool()

	app.activeProxyHost = "9.9.9.9"
	app.activeProxyPort = "1080"

	// One health-check tick on the real monitor instance.
	app.monitorMu.Lock()
	monitor := app.proxyMonitor
	app.monitorMu.Unlock()
	require.NotNil(t, monitor)

	// monitor.tick is unexported in services; run the loop with a short
	// interval instead and wait for the switch to happen.
	monitor.SetInterval(10 * time.Millisecond)

	waitUntilNotifier(t, notifier, func(n *recordingNotifier) bool {
		return len(n.switched) > 0
	})

	// The notifier signal's mutex edge makes this read race-free.
	assert.Equal(t, "8.8.8.8", app.activeProxyHost, "the winner must become the active proxy")
}

func TestMonitorLifecycle_TickSkippedWithoutActiveProxy(t *testing.T) {
	app, _ := newMonitorLifecycleFixture(t)
	ctrl := gomock.NewController(t)
	checker := newMockChecker(ctrl)
	// No active proxy: the store reports not-found, so the checker is never called.

	app.startProxyMonitorWith(checker, &recordingNotifier{})
	t.Cleanup(app.stopProxyMonitor)

	app.proxyItems = []*ProxyItemWrapper{
		{Host: "1.1.1.1", Port: "1080", Source: string(common.SourceProxyMania)},
	}
	app.cacheExpiryMutex.Lock()
	app.cacheExpiry = map[common.Source]time.Time{
		common.SourceProxyMania: time.Now().Add(time.Hour),
	}
	app.cacheExpiryMutex.Unlock()
	app.updateMonitorPool()

	time.Sleep(50 * time.Millisecond) // no tick expectations — gomock fails on unexpected calls
}

// waitForMonitorSwitch polls until the active proxy becomes the expected one.
func waitForMonitorSwitch(t *testing.T, app *AppGUI, host string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if app.activeProxyHost == host {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("monitor did not switch to %s within timeout", host)
}
