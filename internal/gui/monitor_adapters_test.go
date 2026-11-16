// nolint testpackage
package gui

import (
	"errors"
	"image/color"
	"testing"

	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"proxy-checker/internal/mocks"
	"proxy-checker/internal/services"
	"proxy-checker/internal/sysproxy"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

func TestGuiMonitorNotifier_ProxyDownMarksUnavailableRow(t *testing.T) {
	app, _ := newMonitorAdapterFixture(t)
	app.proxyItems = []*ProxyItemWrapper{
		{Host: "1.1.1.1", Port: "1080"},
		{Host: "2.2.2.2", Port: "1080"},
	}
	notifier := newMonitorNotifierAdapter(app)

	notifier.ProxyDown(services.ProxyItem{Host: "1.1.1.1", Port: "1080"})

	assert.True(t, app.proxyItems[0].IsUnavailable,
		"the failed proxy's row must be flagged unavailable")
	assert.False(t, app.proxyItems[1].IsUnavailable,
		"rows of other proxies must stay unflagged")
}

func TestMarkProxyUnavailable_UnknownProxyIsNoop(t *testing.T) {
	app, _ := newMonitorAdapterFixture(t)
	app.proxyItems = []*ProxyItemWrapper{{Host: "1.1.1.1", Port: "1080"}}

	app.markProxyUnavailable("9.9.9.9", "1080")

	assert.False(t, app.proxyItems[0].IsUnavailable,
		"no row must be flagged when the failed proxy is not in the table")
}

func TestTableCell_SetUnavailable(t *testing.T) {
	darkCell := newTableCell(nil)
	darkCell.setUnavailable(true)

	assert.Equal(t, color.NRGBA{R: 95, G: 45, B: 45, A: 160}, darkCell.bg.FillColor,
		"dark theme must use the muted dark red tint")
	assert.Equal(t, float32(0), darkCell.bg.CornerRadius,
		"unavailable background must have no corner radius")

	lightCell := newTableCell(nil)
	lightCell.setUnavailable(false)

	assert.Equal(t, color.NRGBA{R: 255, G: 176, B: 176, A: 160}, lightCell.bg.FillColor,
		"light theme must use the muted light red tint")
	assert.Equal(t, float32(0), lightCell.bg.CornerRadius,
		"unavailable background must have no corner radius")
}

// visibleTableCells collects the result-table cells the widget.Table renders
// right now. Fyne pools cells internally, so the traversal walks the renderer
// object tree (table -> content scroll -> cells) instead of private fields.
func visibleTableCells(tbl *widget.Table) []*tableCell {
	var cells []*tableCell
	for _, obj := range test.WidgetRenderer(tbl).Objects() {
		scroll, ok := obj.(*container.Scroll)
		if !ok {
			continue
		}
		cellsWidget, ok := scroll.Content.(fyne.Widget)
		if !ok {
			continue
		}
		for _, cellObj := range test.WidgetRenderer(cellsWidget).Objects() {
			if cell, ok := cellObj.(*tableCell); ok {
				cells = append(cells, cell)
			}
		}
	}
	return cells
}

func TestResultTable_UnavailableRowRendering(t *testing.T) {
	app, _ := newMonitorAdapterFixture(t)
	app.proxyItems = []*ProxyItemWrapper{
		{Host: "1.1.1.1", Port: "1080", Source: "proxymania", Country: "NL"},
		{Host: "2.2.2.2", Port: "1080", Source: "proxymania", Country: "DE"},
	}
	app.proxyItems[1].IsUnavailable = true
	app.applyCombinedFilters()
	require.Len(t, app.filteredProxyItems, 2)

	// Force the table to lay out its cells for both rows.
	app.table.Resize(fyne.NewSize(2000, 300))
	app.table.Refresh()

	cells := visibleTableCells(app.table)
	require.Len(t, cells, 16, "both rows with all columns must be rendered")

	// Rows are identified by position: cells of one row share the Y offset.
	var unavailableRowY float32
	for _, cell := range cells {
		if cell.label.Visible() && cell.label.Text == "2.2.2.2" {
			unavailableRowY = cell.Position().Y
		}
	}
	require.NotEqual(t, float32(0), unavailableRowY,
		"the unavailable row must not be the first rendered row")

	var applyButtons []*tableCell
	for _, cell := range cells {
		if cell.btn.Visible() {
			applyButtons = append(applyButtons, cell)
		}
	}
	require.Len(t, applyButtons, 2, "each row must render its Apply button")

	for _, cell := range applyButtons {
		if cell.Position().Y == unavailableRowY {
			assert.True(t, cell.btn.Disabled(),
				"the unavailable row's Apply button must be disabled")
		} else {
			assert.False(t, cell.btn.Disabled(),
				"available rows must keep an enabled Apply button")
		}
	}

	unavailableTint := color.NRGBA{R: 255, G: 176, B: 176, A: 160}
	for _, cell := range cells {
		if !cell.label.Visible() {
			continue
		}
		if cell.Position().Y == unavailableRowY {
			assert.Equal(t, unavailableTint, cell.bg.FillColor,
				"unavailable row must be tinted red")
		} else {
			assert.Equal(t, color.Transparent, cell.bg.FillColor,
				"healthy rows must stay untinted without an active highlight")
		}
	}
}
