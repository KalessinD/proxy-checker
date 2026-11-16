package gui

import (
	"errors"
	"fmt"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (g *AppGUI) ShowSettingsScreen() {
	oldSources := g.cfg.Sources
	oldProxyType := g.cfg.Type

	selectSourceCheckAll, sourceChecks := g.createSourceCheckboxes()

	geoipEntry := widget.NewEntry()
	geoipEntry.SetPlaceHolder("/path/to/GeoLite2-Country.mmdb")
	geoipEntry.SetText(g.cfg.GeoIPDBPath)

	themeLabels := []string{i18n.T("gui.settings.theme_system"), i18n.T("gui.settings.theme_light"), i18n.T("gui.settings.theme_dark")}
	selectTheme := g.createThemeSelector(themeLabels)

	availableLangs := i18n.AvailableLanguages()
	selectLang := widget.NewSelect(availableLangs, func(s string) {
		if s == "" {
			return
		}
		g.cfg.Lang = s
		_ = i18n.Init(s)
	})
	selectLang.SetSelected(g.cfg.Lang)

	logPathEntry := widget.NewEntry()
	logPathEntry.SetPlaceHolder(common.DefaultLogPath())
	logPathEntry.SetText(g.cfg.LogPath)
	logPathEntry.OnChanged = func(s string) { g.cfg.LogPath = s }

	ignoreHostsBox, ignoreHostsEntry := g.createIgnoreHostsBox()
	proxyTypeSelector, http2Box := g.createProxyTypeSelector()
	dynamicBox := g.createDynamicFieldsBox()
	selectTarget, customEntry, customBox := g.buildTargetSelector()
	monitorBox, selectMonitorInterval := g.createMonitorSettingsBox()

	g.restoreTargetSelectorState(selectTarget, customEntry, customBox)

	settingsContent := container.NewVBox(
		widget.NewLabel(i18n.T("gui.settings.title")),
		widget.NewSeparator(),
		container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.lang")), selectLang),
		container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.type")), proxyTypeSelector),
		http2Box,
		container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.sources")), selectSourceCheckAll),
		container.NewGridWithColumns(2, container.NewHBox(), func() fyne.CanvasObject {
			objs := make([]fyne.CanvasObject, len(sourceChecks))
			for i, c := range sourceChecks {
				objs[i] = c
			}
			return container.NewGridWithColumns(2, objs...)
		}()),
		dynamicBox,
		container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.workers")), g.createWorkersSelector()),
		container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.timeout")), g.createTimeoutSelector()),
		container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.target")), selectTarget),
		customBox,
		widget.NewSeparator(),
		container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.log_path")), logPathEntry),
		container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.theme")), selectTheme),
		container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.geoip_db")), geoipEntry),
		ignoreHostsBox,
		widget.NewSeparator(),
		monitorBox,
	)

	btnSave := widget.NewButton(i18n.T("gui.btn_save"), func() {
		g.cfg.GeoIPDBPath = geoipEntry.Text

		g.syncSourcesFromChecks(sourceChecks)
		g.initGeoIP(g.cfg.GeoIPDBPath)

		if g.isGeoIPAvailable {
			g.appendLog(common.LogLevelInfo, i18n.T("gui.settings.geoip_loaded")+"\n")
		} else if g.cfg.GeoIPDBPath != "" {
			g.appendLog(common.LogLevelError, fmt.Sprintf("%s: %v\n", i18n.T("gui.settings.geoip_error"), errors.New("file not found or invalid format")))
		}

		if g.sysProxyManager.IsSupported() {
			if err := g.sysProxyManager.SetIgnoreHosts(ignoreHostsEntry.Text); err != nil {
				g.appendLog(common.LogLevelError, fmt.Sprintf("%s: %v", i18n.T("sysproxy.err_set_ignore_hosts"), err))
			} else {
				g.appendLog(common.LogLevelInfo, i18n.T("gui.settings.ignore_hosts_saved")+"\n")
			}
		}

		g.cfg.MonitorIntervalMin = parseMonitorInterval(selectMonitorInterval.Selected)
		g.applyMonitorSettings()

		if err := g.cfg.SaveToFile(); err != nil {
			g.appendLog(common.LogLevelError, fmt.Sprintf("%s: %v\n", i18n.T("gui.settings.save_error"), err))
		} else {
			g.appendLog(common.LogLevelInfo, i18n.T("gui.settings.saved")+"\n")
		}

		if !sourcesEqual(oldSources, g.cfg.Sources) || oldProxyType != g.cfg.Type {
			g.loadCacheForSources(g.cfg.Sources, g.cfg.Type)
		}

		g.setupMainMenu()
		g.ShowMainScreen()
	})

	buttonsBox := container.NewHBox(
		widget.NewButton(i18n.T("gui.btn_back"), func() {
			g.ShowMainScreen()
		}),
		layout.NewSpacer(),
		btnSave,
	)

	g.window.SetContent(container.NewBorder(nil, buttonsBox, nil, nil, container.NewScroll(settingsContent)))
}

// createSourceCheckboxes builds "Select all" and individual source checkboxes.
func (g *AppGUI) createSourceCheckboxes() (*widget.Check, []*widget.Check) {
	availableSources := []common.Source{common.SourceProxyMania, common.SourceTheSpeedX, common.SourceProxifly}
	checks := make([]*widget.Check, 0, len(availableSources))

	var updating bool

	for _, src := range availableSources {
		c := widget.NewCheck(string(src), nil)
		isSelected := false
		for _, s := range g.cfg.Sources {
			if s == src {
				isSelected = true
				break
			}
		}
		c.SetChecked(isSelected)

		checks = append(checks, c)
	}

	selectAllCheck := widget.NewCheck(i18n.T("gui.settings.sources_select_all"), nil)
	allChecked := len(g.cfg.Sources) == len(availableSources)
	selectAllCheck.SetChecked(allChecked)

	selectAllCheck.OnChanged = func(checked bool) {
		if updating {
			return
		}
		updating = true
		defer func() { updating = false }()

		for _, c := range checks {
			c.SetChecked(checked)
		}
	}

	for _, c := range checks {
		c.OnChanged = func(bool) {
			if updating {
				return
			}
			updating = true
			defer func() { updating = false }()

			isAllSelected := true
			for _, ch := range checks {
				if !ch.Checked {
					isAllSelected = false
					break
				}
			}
			selectAllCheck.SetChecked(isAllSelected)
		}
	}

	return selectAllCheck, checks
}

// syncSourcesFromChecks reads the state of checkboxes and updates the config.
func (g *AppGUI) syncSourcesFromChecks(checks []*widget.Check) {
	var sources []common.Source
	for _, c := range checks {
		if c.Checked {
			sources = append(sources, common.Source(c.Text))
		}
	}
	if len(sources) == 0 {
		sources = []common.Source{common.SourceProxyMania}
		for _, c := range checks {
			if c.Text == string(common.SourceProxyMania) {
				c.SetChecked(true)
				break
			}
		}
	}
	g.cfg.Sources = sources
}

// sourcesEqual is a helper to compare two slices of sources.
func sourcesEqual(a, b []common.Source) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (g *AppGUI) createIgnoreHostsBox() (*fyne.Container, *widget.Entry) {
	ignoreHostsEntry := widget.NewEntry()
	ignoreHostsEntry.MultiLine = true
	ignoreHostsEntry.SetPlaceHolder("localhost\n127.0.0.0/8")
	ignoreHostsEntry.SetMinRowsVisible(5)

	if g.sysProxyManager.IsSupported() {
		ignoreHosts, err := g.sysProxyManager.GetIgnoreHosts()
		if err != nil {
			g.appendLog(common.LogLevelError, fmt.Sprintf("%s: %v", i18n.T("sysproxy.err_get_ignore_hosts"), err))
		} else {
			ignoreHostsEntry.SetText(ignoreHosts)
		}
	} else {
		ignoreHostsEntry.Disable()
		ignoreHostsEntry.SetText(i18n.T("common.na"))
	}

	box := container.NewGridWithColumns(2,
		widget.NewLabel(i18n.T("gui.settings.ignore_hosts")),
		ignoreHostsEntry,
	)

	return box, ignoreHostsEntry
}

func (g *AppGUI) createProxyTypeSelector() (*widget.RadioGroup, *fyne.Container) {
	allTypeTranslation := i18n.T("gui.single.type_all")
	proxyTypes := append(common.AllowedProxyTypesStrings(), allTypeTranslation)

	http2Check := widget.NewCheck("", func(checked bool) { g.cfg.CheckHTTP2 = checked })
	http2Check.SetChecked(g.cfg.CheckHTTP2)
	http2Box := container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.check_http2")), http2Check)

	allValue := i18n.T("gui.single.type_all")
	radioType := widget.NewRadioGroup(proxyTypes, func(pt string) {
		proxyType := common.ProxyType(pt)
		if pt == allTypeTranslation {
			g.cfg.Type = common.ProxyAll
		} else {
			g.cfg.Type = proxyType
		}
		if proxyType == common.ProxyHTTPS || proxyType == common.ProxySOCKS5 || proxyType == common.ProxyType(allValue) {
			http2Box.Show()
		} else {
			http2Box.Hide()
			g.cfg.CheckHTTP2 = false
			http2Check.SetChecked(false)
		}
	})

	currentType := string(g.cfg.Type)
	if g.cfg.Type == common.ProxyAll {
		radioType.SetSelected(allValue)
	} else {
		radioType.SetSelected(currentType)
	}
	if currentType != string(common.ProxyHTTPS) && currentType != string(common.ProxySOCKS5) && currentType != allValue {
		http2Box.Hide()
	}
	return radioType, http2Box
}

func (g *AppGUI) createWorkersSelector() *widget.Select {
	workerOptions := []string{"2", "8", "16", "32", "64", "128", "256", "512"}
	selectWorkers := widget.NewSelect(workerOptions, func(s string) {
		val, err := strconv.Atoi(s)
		if err != nil {
			g.appendLog(common.LogLevelError, fmt.Sprintf("%s: %v", i18n.T("cli.err_invalid_type"), err))
			return
		}
		g.cfg.Workers = val
	})
	selectWorkers.SetSelected(strconv.Itoa(g.cfg.Workers))
	return selectWorkers
}

func (g *AppGUI) createTimeoutSelector() *widget.Select {
	timeoutOptions := []string{"1s", "3s", "5s", "10s", "20s", "30s"}
	selectTimeout := widget.NewSelect(timeoutOptions, func(s string) {
		d, err := time.ParseDuration(s)
		if err != nil {
			g.appendLog(common.LogLevelError, fmt.Sprintf("%s: %v", i18n.T("gui.settings.save_error"), err))
			return
		}
		g.cfg.Timeout = d
	})
	currentTimeoutStr := fmt.Sprintf("%ds", int(g.cfg.Timeout.Seconds()))
	selectTimeout.SetSelected(currentTimeoutStr)
	return selectTimeout
}

func (g *AppGUI) createDynamicFieldsBox() *fyne.Container {
	rttOptions := []string{}
	for i := 50; i <= 500; i += 50 {
		rttOptions = append(rttOptions, strconv.Itoa(i))
	}
	selectRTT := widget.NewSelect(rttOptions, func(s string) {
		val, err := strconv.Atoi(s)
		if err != nil {
			g.appendLog(common.LogLevelError, fmt.Sprintf("%s: %v", i18n.T("cli.err_invalid_type"), err))
			return
		}
		g.cfg.RTT = val
	})
	selectRTT.SetSelected(strconv.Itoa(g.cfg.RTT))
	rttLabel := widget.NewLabel(i18n.T("gui.settings.max_rtt"))

	pageOptions := []string{"1", "2", "3", "4", "5"}
	selectPages := widget.NewSelect(pageOptions, func(s string) {
		val, err := strconv.Atoi(s)
		if err != nil {
			return
		}
		g.cfg.Pages = val
	})
	selectPages.SetSelected(strconv.Itoa(g.cfg.Pages))
	pagesLabel := widget.NewLabel(i18n.T("gui.settings.pages"))

	dynamicBox := container.NewVBox(
		container.NewGridWithColumns(2, rttLabel, selectRTT),
		container.NewGridWithColumns(2, pagesLabel, selectPages),
	)

	needsDynamic := false
	for _, src := range g.cfg.Sources {
		if src == common.SourceProxyMania {
			needsDynamic = true
			break
		}
	}

	if !needsDynamic {
		dynamicBox.Hide()
	}
	return dynamicBox
}

func (g *AppGUI) createThemeSelector(themeLabels []string) *widget.Select {
	selectTheme := widget.NewSelect(themeLabels, func(s string) {
		var val string
		switch {
		case s == i18n.T("gui.settings.theme_light"):
			val = themeLight
		case s == i18n.T("gui.settings.theme_dark"):
			val = themeDark
		default:
			val = themeSystem
		}
		g.cfg.Theme = val
		g.applyTheme(val)
	})

	currentThemeLabel := i18n.T("gui.settings.theme_system")
	switch strings.ToLower(g.cfg.Theme) {
	case themeLight:
		currentThemeLabel = i18n.T("gui.settings.theme_light")
	case themeDark:
		currentThemeLabel = i18n.T("gui.settings.theme_dark")
	}
	selectTheme.SetSelected(currentThemeLabel)
	return selectTheme
}

// Monitor auto-switch modes persisted in cfg.MonitorMode (TODO item 1).
const (
	monitorModeAny         = "any"
	monitorModeSameCountry = "same_country"
	monitorModeNone        = "none"
)

// createMonitorSettingsBox builds the monitoring settings block (TODO item 1):
// the auto-switch mode selector, the "exclude proxies without country" flag
// and the check interval selector. Returns the composed container together
// with the interval selector: its selected value is read back in the save
// callback of the settings screen, while the mode selector and the checkbox
// update g.cfg directly through their change handlers.
func (g *AppGUI) createMonitorSettingsBox() (*fyne.Container, *widget.Select) {
	selectMode, requireCountryCheck := g.createMonitorModeSelector()
	selectInterval := g.createMonitorIntervalSelector()

	box := container.NewVBox(
		container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.monitor_mode")), selectMode),
		requireCountryCheck,
		container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.monitor_interval")), selectInterval),
	)
	return box, selectInterval
}

// createMonitorModeSelector builds the auto-switch mode selector and the
// "exclude proxies without country" checkbox. The checkbox is greyed out
// unless the "any available proxy" mode is selected (TODO item 1: the flag
// only applies there). The initial selection comes from cfg.MonitorMode,
// with unknown values falling back to the "any" mode.
func (g *AppGUI) createMonitorModeSelector() (*widget.Select, *widget.Check) {
	type modeOption struct {
		label string
		value string
	}
	modeOptions := []modeOption{
		{label: i18n.T("gui.settings.monitor_mode_any"), value: monitorModeAny},
		{label: i18n.T("gui.settings.monitor_mode_same_country"), value: monitorModeSameCountry},
		{label: i18n.T("gui.settings.monitor_mode_none"), value: monitorModeNone},
	}

	options := make([]string, 0, len(modeOptions))
	labelByValue := make(map[string]string, len(modeOptions))
	valueByLabel := make(map[string]string, len(modeOptions))
	for _, opt := range modeOptions {
		options = append(options, opt.label)
		labelByValue[opt.value] = opt.label
		valueByLabel[opt.label] = opt.value
	}

	requireCountryCheck := widget.NewCheck(i18n.T("gui.settings.monitor_ignore_no_country"), func(checked bool) {
		g.cfg.MonitorRequireCountry = checked
	})
	requireCountryCheck.SetChecked(g.cfg.MonitorRequireCountry)

	selectMode := widget.NewSelect(options, func(label string) {
		mode, ok := valueByLabel[label]
		if !ok {
			return
		}
		g.cfg.MonitorMode = mode
		if mode == monitorModeAny {
			requireCountryCheck.Enable()
		} else {
			requireCountryCheck.Disable()
		}
	})

	currentLabel, known := labelByValue[g.cfg.MonitorMode]
	if !known {
		currentLabel = labelByValue[monitorModeAny]
	}
	selectMode.SetSelected(currentLabel) // fires the change handler: syncs the checkbox enabled state

	return selectMode, requireCountryCheck
}

// createMonitorIntervalSelector builds the monitor tick-period selector,
// initialized from cfg.MonitorIntervalMin. The value is not applied here:
// it is parsed in the settings save callback so that an invalid selection
// can fall back to the default.
func (g *AppGUI) createMonitorIntervalSelector() *widget.Select {
	intervalOptions := []string{"1", "2", "5", "10"}
	selectInterval := widget.NewSelect(intervalOptions, nil)
	selectInterval.SetSelected(strconv.Itoa(g.cfg.MonitorIntervalMin))
	return selectInterval
}

// parseMonitorInterval converts the selected interval option into minutes.
// An empty or invalid selection falls back to the default of 1 minute: a
// MonitorIntervalMin of zero or less would break the monitor timer
// (Zero-Value rule).
func parseMonitorInterval(selected string) int {
	const defaultIntervalMinutes = 1
	minutes, err := strconv.Atoi(selected)
	if err != nil || minutes <= 0 {
		return defaultIntervalMinutes
	}
	return minutes
}
