package gui

import (
	"errors"
	"fmt"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (g *AppGUI) showSettingsScreen() {
	proxyTypes := []string{
		string(common.ProxyHTTP),
		string(common.ProxyHTTPS),
		string(common.ProxySOCKS4),
		string(common.ProxySOCKS5),
		i18n.T("gui.single.type_all"),
	}

	http2Check := widget.NewCheck("", func(checked bool) {
		g.cfg.CheckHTTP2 = checked
	})
	http2Check.SetChecked(g.cfg.CheckHTTP2)
	http2Box := container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.check_http2")), http2Check)

	logPathEntry := widget.NewEntry()
	logPathEntry.SetPlaceHolder(common.DefaultLogPath())
	logPathEntry.SetText(g.cfg.LogPath)
	logPathEntry.OnChanged = func(s string) { g.cfg.LogPath = s }

	allValue := i18n.T("gui.single.type_all")
	radioType := widget.NewRadioGroup(proxyTypes, func(pt string) {
		proxyType := common.ProxyType(pt)
		if proxyType == common.ProxyType(allValue) {
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

	sources := []string{"proxymania", "thespeedx"}
	selectSource := widget.NewSelect(sources, func(s string) {
		g.cfg.Source = common.Source(s)
	})
	selectSource.SetSelected(string(g.cfg.Source))

	rttOptions := []string{}
	for i := 50; i <= 500; i += 50 {
		rttOptions = append(rttOptions, strconv.Itoa(i))
	}
	selectRTT := widget.NewSelect(rttOptions, func(s string) {
		val, _ := strconv.Atoi(s)
		g.cfg.RTT = val
	})
	selectRTT.SetSelected(strconv.Itoa(g.cfg.RTT))
	rttLabel := widget.NewLabel(i18n.T("gui.settings.max_rtt"))

	workerOptions := []string{"2", "8", "16", "32", "64", "128", "256", "512"}
	selectWorkers := widget.NewSelect(workerOptions, func(s string) {
		val, _ := strconv.Atoi(s)
		g.cfg.Workers = val
	})
	selectWorkers.SetSelected(strconv.Itoa(g.cfg.Workers))

	pageOptions := []string{"1", "2", "3", "4", "5"}
	selectPages := widget.NewSelect(pageOptions, func(s string) {
		val, _ := strconv.Atoi(s)
		g.cfg.Pages = val
	})
	selectPages.SetSelected(strconv.Itoa(g.cfg.Pages))
	pagesLabel := widget.NewLabel(i18n.T("gui.settings.pages"))

	timeoutOptions := []string{"1s", "3s", "5s", "10s", "20s", "30s"}
	selectTimeout := widget.NewSelect(timeoutOptions, func(s string) {
		d, _ := time.ParseDuration(s)
		g.cfg.Timeout = d
	})
	currentTimeoutStr := fmt.Sprintf("%ds", int(g.cfg.Timeout.Seconds()))
	selectTimeout.SetSelected(currentTimeoutStr)

	targetSites := []string{
		"google.com",
		"youtube.com",
		"chatgpt.com",
		"web.telegram.org",
		i18n.T("gui.single.custom_site"),
	}

	customEntry := widget.NewEntry()
	customEntry.SetPlaceHolder(i18n.T("gui.single.custom_placeholder"))
	customEntry.OnChanged = func(s string) { g.customTargetURL = s }

	customBox := container.NewVBox(widget.NewLabel(i18n.T("gui.single.enter_addr")), customEntry)
	customBox.Hide()

	selectTarget := widget.NewSelect(targetSites, func(s string) {
		if s == i18n.T("gui.single.custom_site") {
			g.isCustomTarget = true
			customBox.Show()
		} else {
			g.isCustomTarget = false
			g.cfg.DestAddr = s
			g.customTargetURL = ""
			customBox.Hide()
		}
	})
	selectTarget.PlaceHolder = i18n.T("gui.settings.target_placeholder")

	if g.isCustomTarget {
		selectTarget.SetSelected(i18n.T("gui.single.custom_site"))
		customBox.Show()
	} else if g.cfg.DestAddr != "" {
		selectTarget.SetSelected(g.cfg.DestAddr)
	}

	themeLabels := []string{i18n.T("gui.settings.theme_system"), i18n.T("gui.settings.theme_light"), i18n.T("gui.settings.theme_dark")}
	selectTheme := widget.NewSelect(themeLabels, func(s string) {
		var val string
		switch {
		case s == i18n.T("gui.settings.theme_light"):
			val = "light"
		case s == i18n.T("gui.settings.theme_dark"):
			val = "dark"
		default:
			val = "system"
		}
		g.cfg.Theme = val
		g.applyTheme(val)
	})

	currentThemeLabel := i18n.T("gui.settings.theme_system")
	switch strings.ToLower(g.cfg.Theme) {
	case "light":
		currentThemeLabel = i18n.T("gui.settings.theme_light")
	case "dark":
		currentThemeLabel = i18n.T("gui.settings.theme_dark")
	}
	selectTheme.SetSelected(currentThemeLabel)

	availableLangs := i18n.AvailableLanguages()
	selectLang := widget.NewSelect(availableLangs, func(s string) {
		if s == "" {
			return
		}
		g.cfg.Lang = s
		_ = i18n.Init(s)
	})
	selectLang.SetSelected(g.cfg.Lang)

	langBox := container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.lang")), selectLang)
	rttBox := container.NewGridWithColumns(2, rttLabel, selectRTT)
	pagesBox := container.NewGridWithColumns(2, pagesLabel, selectPages)

	geoipEntry := widget.NewEntry()
	geoipEntry.SetPlaceHolder("/path/to/GeoLite2-Country.mmdb")
	geoipEntry.SetText(g.cfg.GeoIPDBPath)

	geoipBox := container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.geoip_db")), geoipEntry)
	dynamicBox := container.NewVBox(rttBox, pagesBox)

	toggleDynamicFields := func(source string) {
		if source == string(common.SourceTheSpeedX) {
			dynamicBox.Hide()
		} else {
			dynamicBox.Show()
		}
	}

	toggleDynamicFields(string(g.cfg.Source))

	selectSource.OnChanged = func(s string) {
		g.cfg.Source = common.Source(s)
		toggleDynamicFields(s)
	}

	ignoreHostsEntry := widget.NewEntry()
	ignoreHostsEntry.MultiLine = true
	ignoreHostsEntry.SetPlaceHolder("localhost\n127.0.0.0/8")
	ignoreHostsEntry.SetMinRowsVisible(5)

	if g.sysProxyManager.IsSupported() {
		ignoreHosts, err := g.sysProxyManager.GetIgnoreHosts()
		if err != nil {
			g.appendLog(fmt.Sprintf("%s: %v", i18n.T("sysproxy.err_get_ignore_hosts"), err))
		} else {
			ignoreHostsEntry.SetText(ignoreHosts)
		}
	} else {
		ignoreHostsEntry.Disable()
		ignoreHostsEntry.SetText(i18n.T("common.na"))
	}

	ignoreHostsBox := container.NewGridWithColumns(2,
		widget.NewLabel(i18n.T("gui.settings.ignore_hosts")),
		ignoreHostsEntry,
	)

	settingsContent := container.NewVBox(
		widget.NewLabel(i18n.T("gui.settings.title")),
		widget.NewSeparator(),
		langBox,
		container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.type")), radioType),
		http2Box,
		container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.source")), selectSource),
		dynamicBox,
		container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.workers")), selectWorkers),
		container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.timeout")), selectTimeout),
		container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.target")), selectTarget),
		customBox,
		widget.NewSeparator(),
		container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.log_path")), logPathEntry),
		container.NewGridWithColumns(2, widget.NewLabel(i18n.T("gui.settings.theme")), selectTheme),
		geoipBox,
		ignoreHostsBox,
	)

	btnSave := widget.NewButton(i18n.T("gui.btn_save"), func() {
		g.cfg.GeoIPDBPath = geoipEntry.Text

		g.initGeoIP(g.cfg.GeoIPDBPath)
		if g.isGeoIPAvailable {
			g.appendLog(i18n.T("gui.settings.geoip_loaded") + "\n")
		} else if g.cfg.GeoIPDBPath != "" {
			g.appendLog(fmt.Sprintf("%s: %v\n", i18n.T("gui.settings.geoip_error"), errors.New("file not found or invalid format")))
		}

		if g.sysProxyManager.IsSupported() {
			if err := g.sysProxyManager.SetIgnoreHosts(ignoreHostsEntry.Text); err != nil {
				g.appendLog(fmt.Sprintf("%s: %v", i18n.T("sysproxy.err_set_ignore_hosts"), err))
			} else {
				g.appendLog(i18n.T("gui.settings.ignore_hosts_saved") + "\n")
			}
		}

		if err := g.cfg.SaveToFile(); err != nil {
			g.appendLog(fmt.Sprintf("%s: %v\n", i18n.T("gui.settings.save_error"), err))
		} else {
			g.appendLog(i18n.T("gui.settings.saved") + "\n")
		}
		g.showMainScreen()
	})

	btnBack := widget.NewButton(i18n.T("gui.btn_back"), func() {
		g.showMainScreen()
	})

	buttonsBox := container.NewHBox(btnBack, layout.NewSpacer(), btnSave)

	content := container.NewBorder(
		nil, buttonsBox, nil, nil,
		settingsContent,
	)

	g.window.SetContent(content)
}
