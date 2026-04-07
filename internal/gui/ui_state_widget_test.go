// nolint testpackage
package gui

import (
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/config"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestSyncSourcesFromChecks_DefaultsToProxyMania(t *testing.T) {
	testApp := test.NewTempApp(t)
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	g := NewAppGUI(testApp, cfg, logger, "dev")

	checkProxymania := widget.NewCheck(string(common.SourceProxyMania), nil)
	checkThespeedx := widget.NewCheck(string(common.SourceTheSpeedX), nil)

	checkProxymania.SetChecked(false)
	checkThespeedx.SetChecked(false)

	g.syncSourcesFromChecks([]*widget.Check{checkProxymania, checkThespeedx})

	assert.Equal(t, []common.Source{common.SourceProxyMania}, g.cfg.Sources)
	assert.True(t, checkProxymania.Checked, "Must force-check ProxyMania if nothing is selected")
}

func TestSetUIState_RunningDisablesButtons(t *testing.T) {
	testApp := test.NewTempApp(t)
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	g := NewAppGUI(testApp, cfg, logger, "dev")

	g.initUIComponents()
	g.setUIState(true)

	assert.True(t, g.btnCheckList.Disabled(), "Check list button must be disabled while running")
	assert.True(t, g.btnCheckSingle.Disabled(), "Check single button must be disabled while running")
	assert.False(t, g.btnCancel.Disabled(), "Cancel button must be enabled while running")
}

func TestSetUIState_StoppedEnablesButtons(t *testing.T) {
	testApp := test.NewTempApp(t)
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	g := NewAppGUI(testApp, cfg, logger, "dev")

	g.initUIComponents()
	g.setUIState(false)

	assert.False(t, g.btnCheckList.Disabled(), "Check list button must be enabled when stopped")
	assert.False(t, g.btnCheckSingle.Disabled(), "Check single button must be enabled when stopped")
	assert.True(t, g.btnCancel.Disabled(), "Cancel button must be disabled when stopped")
	assert.Nil(t, g.cancelFunc, "Cancel func must be cleared when stopped")
}

func TestBuildTargetSelector_CustomSiteState(t *testing.T) {
	testApp := test.NewTempApp(t)
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	g := NewAppGUI(testApp, cfg, logger, "dev")

	targetSelect, _, customBox := g.buildTargetSelector()
	targetSelect.SetSelected(i18n.T("gui.single.custom_site"))

	assert.True(t, g.isCustomTarget, "isCustomTarget flag must be true when custom site is selected")
	assert.True(t, customBox.Visible(), "Custom entry box must be visible")
}

func TestBuildTargetSelector_NormalSiteState(t *testing.T) {
	testApp := test.NewTempApp(t)
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	g := NewAppGUI(testApp, cfg, logger, "dev")

	targetSelect, _, customBox := g.buildTargetSelector()
	targetSelect.SetSelected("youtube.com")

	assert.False(t, g.isCustomTarget, "isCustomTarget flag must be false for standard sites")
	assert.Equal(t, "youtube.com", g.cfg.DestAddr, "DestAddr in config must be updated")
	assert.False(t, customBox.Visible(), "Custom entry box must be hidden")
}

func TestCreateProxyTypeSelector_SetsAllType(t *testing.T) {
	testApp := test.NewTempApp(t)
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	g := NewAppGUI(testApp, cfg, logger, "dev")

	radioType, _ := g.createProxyTypeSelector()
	radioType.SetSelected(i18n.T("gui.single.type_all"))

	assert.Equal(t, common.ProxyAll, g.cfg.Type, "Config type must be set to ProxyAll")
}

func TestTableCell_UpdateText(t *testing.T) {
	cell := newTableCell(nil)
	cell.updateText("192.168.1.1")

	assert.True(t, cell.label.Visible(), "Label must be visible on text update")
	assert.False(t, cell.btn.Visible(), "Button must be hidden on text update")
}

func TestTableCell_UpdateButton(t *testing.T) {
	cell := newTableCell(nil)
	var buttonClicked bool
	cell.updateButton(func() { buttonClicked = true })

	assert.False(t, cell.label.Visible(), "Label must be hidden on button update")
	assert.True(t, cell.btn.Visible(), "Button must be visible on button update")

	cell.btn.OnTapped()
	assert.True(t, buttonClicked, "Button callback must be executable")
}

func TestResizableTable_MinSize(t *testing.T) {
	testApp := test.NewTempApp(t)
	defer testApp.Quit()

	table := widget.NewTable(
		func() (int, int) { return 0, 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(widget.TableCellID, fyne.CanvasObject) {},
	)

	minW := float32(800)
	minH := float32(600)
	scalableTable := newResizableTable(table, nil, false, minW, minH)

	actualSize := scalableTable.MinSize()
	assert.Equal(t, minW, actualSize.Width, "MinSize width must match config")
	assert.Equal(t, minH, actualSize.Height, "MinSize height must match config")
}
