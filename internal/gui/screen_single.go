package gui

import (
	"context"
	"fmt"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/services"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (g *AppGUI) ShowSingleCheckScreen() {
	proxyEntry := widget.NewEntry()
	proxyEntry.SetPlaceHolder(i18n.T("gui.single.placeholder"))

	allTypeTranslation := i18n.T("gui.single.type_all")
	proxyTypes := append(common.AllowedProxyTypesStrings(), allTypeTranslation)
	radioType := widget.NewRadioGroup(proxyTypes, nil)
	currentType := string(g.cfg.Type)
	if g.cfg.Type == common.ProxyAll {
		radioType.SetSelected(allTypeTranslation)
	} else {
		radioType.SetSelected(currentType)
	}

	targetSelect, customEntry, customBox := g.buildTargetSelector()
	g.restoreTargetSelectorState(targetSelect, customEntry, customBox)

	btnRun := widget.NewButton(i18n.T("gui.btn_run"), func() {
		addr := proxyEntry.Text
		target := g.getTargetURL()

		selectedType := radioType.Selected
		checkType := selectedType
		if selectedType == allTypeTranslation {
			checkType = "socks5"
		}

		if addr == "" {
			g.appendLog(common.LogLevelInfo, i18n.T("gui.single.err_empty_addr")+"\n")
			return
		}

		g.showMainScreen()
		g.appendLog(common.LogLevelInfo, fmt.Sprintf("%s: %s -> %s (%s)...\n", i18n.T("gui.single.log_checking"), addr, target, checkType))
		_ = g.progress.Set(0)

		go func() {
			parts := strings.Split(addr, ":")
			host := parts[0]
			port := ""
			if len(parts) > 1 {
				port = parts[1]
			}

			ctx, cancel := context.WithTimeout(context.Background(), g.cfg.Timeout)
			defer cancel()

			checker := services.NewProxyChecker()
			res := checker.CheckProxy(ctx, addr, target, checkType, g.cfg.CheckHTTP2)
			if res.Error != nil {
				g.appendLog(common.LogLevelError, fmt.Sprintf("%s %v\n", i18n.T("cli.fail"), res.Error))
				return
			}

			item := &ProxyItemWrapper{
				Host:    host,
				Port:    port,
				Type:    common.ProxyType(checkType),
				Country: "N/A",
				TCP:     res.ProxyLatencyStr,
				HTTP:    res.ReqLatencyStr,
			}

			fyne.Do(func() {
				g.proxyItems = []*ProxyItemWrapper{item}
				if g.table != nil {
					g.table.Refresh()
				}
			})
			g.appendLog(common.LogLevelInfo, fmt.Sprintf("%s: %d\n", i18n.T("gui.single.log_done"), res.StatusCode))
			_ = g.progress.Set(1.0)
		}()
	})

	btnBack := widget.NewButton(i18n.T("gui.btn_back"), func() {
		g.showMainScreen()
	})

	buttonsBox := container.NewHBox(btnBack, layout.NewSpacer(), btnRun)

	inputForm := widget.NewForm(
		widget.NewFormItem(i18n.T("gui.settings.type"), radioType),
		widget.NewFormItem(i18n.T("gui.single.title"), proxyEntry),
		widget.NewFormItem(i18n.T("gui.settings.target"), targetSelect),
		widget.NewFormItem("", customBox),
	)

	content := container.NewBorder(
		nil,
		buttonsBox,
		nil, nil,
		container.NewVBox(
			widget.NewLabel(i18n.T("gui.single.title")),
			widget.NewSeparator(),
			inputForm,
		),
	)

	g.window.SetContent(content)
}
