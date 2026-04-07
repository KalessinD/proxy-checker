package gui

import (
	"context"
	"fmt"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/fetcher"
	"proxy-checker/internal/services"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (g *AppGUI) showMainScreen() {
	g.btnSettings = widget.NewButton(i18n.T("gui.btn_settings"), func() { g.ShowSettingsScreen() })
	g.btnCheckSingle = widget.NewButton(i18n.T("gui.btn_check_single"), func() { g.ShowSingleCheckScreen() })
	g.btnCheckList = widget.NewButton(i18n.T("gui.btn_check_list"), func() { go g.runBatchCheck() })
	g.btnCancel = widget.NewButton(i18n.T("gui.btn_cancel"), func() {
		if g.cancelFunc != nil {
			g.cancelFunc()
			g.appendLog(common.LogLevelInfo, i18n.T("gui.log_stopped")+"\n")
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
	logArea := newMinSizeWidget(g.logScroll, fyne.NewSize(0, 150))

	g.countryFilterSelect = widget.NewSelect([]string{i18n.T("gui.filter_all")}, func(selected string) {
		g.applyCountryFilter(selected)
	})
	g.countryFilterSelect.PlaceHolder = i18n.T("gui.header_country")

	g.updateCountryFilterOptions()
	g.applyCountryFilter(g.countryFilterSelect.Selected)

	filterContainer := container.NewHBox(
		widget.NewLabel(i18n.T("gui.header_country")+":"),
		g.countryFilterSelect,
	)

	topBox.Add(logArea)
	topBox.Add(filterContainer)
	topBox.Add(widget.NewLabel(i18n.T("gui.label_progress")))
	topBox.Add(progressBar)

	g.table = g.createResultTable()

	tableHeader, _ := g.buildSortableHeader(g.sysProxyManager.IsSupported())

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
	fyne.DoAndWait(func() {
		if g.logRichText != nil {
			g.logRichText.Segments = nil
			g.logRichText.Refresh()
		}
	})

	g.appendLog(common.LogLevelInfo, i18n.T("gui.log_preparing")+"\n")
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

	sourcesStr := strings.Join(common.SourcesToStrings(g.cfg.Sources), ", ")
	g.appendLog(common.LogLevelInfo, fmt.Sprintf("%s: %s...\n", i18n.T("gui.log_fetching"), sourcesStr))

	fetchers := make([]services.SourceFetcher, 0, len(g.cfg.Sources))
	for _, src := range g.cfg.Sources {
		fetchers = append(fetchers, services.SourceFetcher{
			Source:  src,
			Fetcher: fetcher.NewFetcher(src, g.logger),
		})
	}
	verifierInstance := services.NewDefaultVerifier()

	validProxies, err := services.RunPipeline(ctx, fetchers, verifierInstance, g.cfg, g.geoIPResolver, services.PipelineCallbacks{
		OnFetched: func(total int) {
			g.appendLog(common.LogLevelInfo, fmt.Sprintf("%s: %d...\n", i18n.T("gui.log_found"), total))
		},
		OnProgress: func(current, total int) {
			if ctx.Err() == nil {
				_ = g.progress.Set(float64(current) / float64(total))
			}
		},
	})
	if err != nil {
		g.appendLog(common.LogLevelError, fmt.Sprintf("%s: %v", i18n.T("gui.log_fetch_error"), err))
		return
	}

	guiItems := g.mapToWrapper(validProxies)

	fyne.Do(func() {
		g.proxyItems = guiItems
		g.resetAndApplyFilter()
	})

	if ctx.Err() != nil {
		g.appendLog(common.LogLevelInfo, i18n.T("gui.log_stopped")+"\n")
	} else {
		g.appendLog(common.LogLevelInfo, fmt.Sprintf("%s: %d\n", i18n.T("gui.log_done"), len(validProxies)))

		for _, src := range g.cfg.Sources {
			var srcProxies []*services.ProxyItemFull
			for _, p := range validProxies {
				if p.Source == src {
					srcProxies = append(srcProxies, p)
				}
			}

			var err error
			if len(srcProxies) > 0 {
				err = g.cache.Save(src, g.cfg.Type, srcProxies, g.cfg.CacheTTL)
			}
			if err != nil {
				g.appendLog(common.LogLevelError, fmt.Sprintf("%s: %v\n", i18n.T("gui.log_cache_error"), err))
			}
		}
		g.appendLog(common.LogLevelInfo, i18n.T("gui.log_cache_saved")+"\n")
	}
	_ = g.progress.Set(1.0)
}

func (g *AppGUI) createResultTable() *widget.Table {
	cols := 7
	if g.sysProxyManager.IsSupported() {
		cols = 8
	}

	table := widget.NewTable(
		func() (int, int) {
			return len(g.filteredProxyItems), cols
		},
		func() fyne.CanvasObject {
			return newTableCell()
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			if id.Row < 0 || id.Row >= len(g.filteredProxyItems) || id.Col >= cols {
				return
			}

			item := g.filteredProxyItems[id.Row]

			tc, ok := cell.(*tableCell)
			if !ok {
				return
			}

			if g.sysProxyManager.IsSupported() && id.Col == 7 {
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
				text = item.Source
			case 1:
				text = item.Host
			case 2:
				text = item.Port
			case 3:
				text = string(item.Type)
			case 4:
				text = item.Country
			case 5:
				text = item.TCP
			case 6:
				text = item.HTTP
			}

			tc.updateText(text)

			if id.Row == g.highlightedRow {
				tc.setHighlighted(true, g.isDarkTheme)
			} else {
				tc.setHighlighted(false, g.isDarkTheme)
			}
		},
	)

	return table
}

func (g *AppGUI) applySystemProxy(host, port, proxyType string) {
	err := g.sysProxyManager.SetProxy(host, port, proxyType)
	if err != nil {
		g.appendLog(common.LogLevelError, fmt.Sprintf("%s %s:%s (%s): %v\n", i18n.T("gui.log_apply_error"), host, port, proxyType, err))
		return
	}

	g.switchProxy.SetChecked(true)
	g.appendLog(common.LogLevelInfo, fmt.Sprintf("%s: %s://%s:%s\n", i18n.T("gui.log_apply_success"), strings.ToLower(proxyType), host, port))
	g.highlightProxyInList(host, port)
}

// sortProxyItems sorts the slice of UI wrappers based on the selected column index and direction.
func sortProxyItems(items []*ProxyItemWrapper, col int, asc bool) {
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		var less bool

		switch col {
		case 0:
			less = a.Source < b.Source
		case 1:
			less = a.Host < b.Host
		case 2:
			less = a.Port < b.Port
		case 3:
			less = string(a.Type) < string(b.Type)
		case 4:
			less = a.Country < b.Country
		case 5:
			less = a.TCP < b.TCP
		case 6:
			less = a.HTTP < b.HTTP
		default:
			less = false
		}

		if asc {
			return less
		}
		return !less
	})
}

// buildSortableHeader creates a table header with clickable columns for sorting.
// The sorting state (active column and direction) is managed locally via closures.
func (g *AppGUI) buildSortableHeader(hasButtonCol bool) (*fyne.Container, func(int, bool)) {
	baseTexts := []string{
		i18n.T("gui.header_source"),
		i18n.T("gui.header_host"),
		i18n.T("gui.header_port"),
		i18n.T("gui.header_type"),
		i18n.T("gui.header_country"),
		i18n.T("gui.header_tcp"),
		i18n.T("gui.header_http"),
	}

	buttons := make([]*widget.Button, len(baseTexts))
	stateCol := -1
	stateAsc := true

	updateLabels := func(activeCol int, asc bool) {
		for i, btn := range buttons {
			text := baseTexts[i]
			if i == activeCol {
				if asc {
					text += " ▲"
				} else {
					text += " ▼"
				}
			}
			btn.SetText(text)
		}
	}
	for idx, txt := range baseTexts {
		btn := widget.NewButton(txt, func() {
			if stateCol == idx {
				stateAsc = !stateAsc
			} else {
				stateCol = idx
				stateAsc = true
			}

			sortProxyItems(g.filteredProxyItems, idx, stateAsc)
			updateLabels(stateCol, stateAsc)

			if g.table != nil {
				g.table.Refresh()
			}
		})

		btn.Importance = widget.LowImportance
		btn.Alignment = widget.ButtonAlignLeading
		buttons[idx] = btn
	}

	headerObjects := make([]fyne.CanvasObject, len(buttons))
	for i, b := range buttons {
		headerObjects[i] = b
	}

	if hasButtonCol {
		headerObjects = append(headerObjects, widget.NewLabel(""))
	}

	return container.NewGridWithColumns(len(headerObjects), headerObjects...), updateLabels
}
