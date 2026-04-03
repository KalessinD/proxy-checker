package gui

import (
	"context"
	"fmt"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/fetcher"
	"proxy-checker/internal/services"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (g *AppGUI) showMainScreen() {
	g.btnSettings = widget.NewButton(i18n.T("gui.btn_settings"), func() { g.showSettingsScreen() })
	g.btnCheckSingle = widget.NewButton(i18n.T("gui.btn_check_single"), func() { g.showSingleCheckScreen() })
	g.btnCheckList = widget.NewButton(i18n.T("gui.btn_check_list"), func() { go g.runBatchCheck() })
	g.btnCancel = widget.NewButton(i18n.T("gui.btn_cancel"), func() {
		if g.cancelFunc != nil {
			g.cancelFunc()
			g.appendLog(i18n.T("gui.log_stopped") + "\n")
		}
	})
	g.btnCancel.Importance = widget.DangerImportance

	if g.cancelFunc != nil {
		g.setUIState(true)
	} else {
		g.setUIState(false)
	}

	rightButtons := container.NewHBox(
		g.btnCancel,
		g.btnCheckSingle,
		g.btnCheckList,
	)

	var leftSide fyne.CanvasObject
	if g.sysProxyManager.IsSupported() {
		proxyLabel := widget.NewLabelWithStyle(i18n.T("gui.label_sys_proxy"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

		alignedLabel := newVPad(proxyLabel, 3)
		borderedSwitch := newBorderedBox(g.switchProxy)
		alignedSwitch := newVPad(borderedSwitch, 3)

		leftSide = container.NewHBox(
			g.btnSettings,
			alignedLabel,
			alignedSwitch,
		)
	} else {
		leftSide = g.btnSettings
	}

	buttonsBar := container.NewBorder(nil, nil, leftSide, rightButtons)
	buttonsContainer := container.NewVBox(
		widget.NewLabel(""),
		container.NewPadded(buttonsBar),
	)

	progressBar := widget.NewProgressBarWithData(g.progress)
	g.progressBar = progressBar

	topBox := container.NewVBox()

	if !g.isGeoIPAvailable {
		warnLabel := widget.NewLabel(i18n.T("gui.warn_no_geoip"))
		warnLabel.Wrapping = fyne.TextWrapWord
		topBox.Add(warnLabel)
		topBox.Add(widget.NewSeparator())
	}

	topBox.Add(widget.NewLabel(i18n.T("gui.label_logs")))
	// ИСПОЛЬЗУЕМ УЖЕ СОЗДАННЫЕ В initUIComponents ВИДЖЕТЫ
	logArea := newMinSizeWidget(g.logScroll, fyne.NewSize(0, 150))

	topBox.Add(logArea)
	topBox.Add(widget.NewLabel(i18n.T("gui.label_progress")))
	topBox.Add(progressBar)

	g.table = g.createResultTable()

	headerObjects := []fyne.CanvasObject{
		widget.NewLabel(i18n.T("gui.header_host")), widget.NewLabel(i18n.T("gui.header_port")), widget.NewLabel(i18n.T("gui.header_type")),
		widget.NewLabel(i18n.T("gui.header_country")), widget.NewLabel(i18n.T("gui.header_tcp")), widget.NewLabel(i18n.T("gui.header_http")),
	}
	if g.sysProxyManager.IsSupported() {
		headerObjects = append(headerObjects, widget.NewLabel(""))
	}
	tableHeader := container.NewGridWithColumns(len(headerObjects), headerObjects...)

	scalableTable := newResizableTable(
		g.table,
		tableHeader,
		g.sysProxyManager.IsSupported(),
		float32(g.cfg.MinWidth),
		float32(g.cfg.MinHeight),
	)

	paddedTable := container.NewPadded(scalableTable)

	content := container.NewBorder(
		topBox,
		buttonsContainer,
		nil,
		nil,
		paddedTable,
	)

	g.window.SetContent(content)
}

func (g *AppGUI) setUIState(running bool) {
	if running {
		g.btnCheckList.Disable()
		g.btnCheckSingle.Disable()
		g.btnCancel.Enable()
	} else {
		g.btnCheckList.Enable()
		g.btnCheckSingle.Enable()
		g.btnCancel.Disable()
		g.cancelFunc = nil
	}
}

func (g *AppGUI) runBatchCheck() {
	g.logBuffer = ""
	g.appendLog(i18n.T("gui.log_preparing") + "\n")
	_ = g.progress.Set(0)

	g.progressBar.Show()

	ctx, cancel := context.WithCancel(context.Background())
	g.cancelFunc = cancel

	fyne.DoAndWait(func() {
		g.setUIState(true)
	})
	defer fyne.DoAndWait(func() {
		g.setUIState(false)
		g.progressBar.Hide()
	})

	g.appendLog(fmt.Sprintf("%s: %s...\n", i18n.T("gui.log_fetching"), g.cfg.Source))

	fetcherInstance := fetcher.NewFetcher(g.cfg.Source)
	verifierInstance := services.NewDefaultVerifier()

	validProxies, err := services.RunPipeline(ctx, fetcherInstance, verifierInstance, g.cfg, g.geoIPResolver, services.PipelineCallbacks{
		OnFetched: func(total int) {
			g.appendLog(fmt.Sprintf("%s: %d...\n", i18n.T("gui.log_found"), total))
		},
		OnProgress: func(current, total int) {
			if ctx.Err() == nil {
				_ = g.progress.Set(float64(current) / float64(total))
			}
		},
	})
	if err != nil {
		g.appendLog(fmt.Sprintf("%s: %v", i18n.T("gui.log_fetch_error"), err))
		return
	}

	guiItems := make([]*ProxyItemWrapper, len(validProxies))
	for i, p := range validProxies {
		guiItems[i] = &ProxyItemWrapper{
			Host:    p.Host,
			Port:    p.Port,
			Type:    p.Type,
			Country: p.Country,
			TCP:     p.CheckResult.ProxyLatencyStr,
			HTTP:    p.CheckResult.ReqLatencyStr,
		}
	}

	fyne.Do(func() {
		g.proxyItems = guiItems
		if g.table != nil {
			g.table.Refresh()
		}
	})

	if ctx.Err() != nil {
		g.appendLog(i18n.T("gui.log_stopped") + "\n")
	} else {
		g.appendLog(fmt.Sprintf("%s: %d\n", i18n.T("gui.log_done"), len(validProxies)))

		if err := g.cache.Save(validProxies); err != nil {
			g.appendLog(fmt.Sprintf("%s: %v\n", i18n.T("gui.log_cache_error"), err))
		} else {
			g.appendLog(i18n.T("gui.log_cache_saved") + "\n")
		}
	}
	_ = g.progress.Set(1.0)
}

func (g *AppGUI) createResultTable() *widget.Table {
	cols := 6
	if g.sysProxyManager.IsSupported() {
		cols = 7
	}

	table := widget.NewTable(
		func() (int, int) {
			return len(g.proxyItems), cols
		},
		func() fyne.CanvasObject {
			return newTableCell()
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			if id.Row < 0 || id.Row >= len(g.proxyItems) {
				return
			}

			item := g.proxyItems[id.Row]

			tc, ok := cell.(*tableCell)
			if !ok {
				return
			}

			if g.sysProxyManager.IsSupported() && id.Col == 6 {
				h := item.Host
				pt := item.Port
				t := item.Type
				tc.updateButton(func() {
					g.applySystemProxy(h, pt, string(t))
				})
				return
			}

			var text string
			switch id.Col {
			case 0:
				text = item.Host
			case 1:
				text = item.Port
			case 2:
				text = string(item.Type)
			case 3:
				text = item.Country
			case 4:
				text = item.TCP
			case 5:
				text = item.HTTP
			}
			tc.updateText(text)
		},
	)

	return table
}

func (g *AppGUI) applySystemProxy(host, port, proxyType string) {
	err := g.sysProxyManager.SetProxy(host, port, proxyType)
	if err != nil {
		g.appendLog(fmt.Sprintf("%s %s:%s (%s): %v\n", i18n.T("gui.log_apply_error"), host, port, proxyType, err))
	} else {
		g.appendLog(fmt.Sprintf("%s: %s://%s:%s\n", i18n.T("gui.log_apply_success"), strings.ToLower(proxyType), host, port))
		g.switchProxy.SetChecked(true)
	}
}
