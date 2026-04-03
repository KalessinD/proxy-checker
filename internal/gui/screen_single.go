package gui

import (
	"context"
	"fmt"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/services"
	"strings"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (g *AppGUI) showSingleCheckScreen() {
	proxyEntry := widget.NewEntry()
	proxyEntry.SetPlaceHolder(i18n.T("gui.single.placeholder"))

	proxyTypes := []string{"http", "https", "socks4", "socks5", i18n.T("gui.single.type_all")}
	radioType := widget.NewRadioGroup(proxyTypes, nil)
	currentType := string(g.cfg.Type)
	if g.cfg.Type == common.ProxyAll {
		radioType.SetSelected(i18n.T("gui.single.type_all"))
	} else {
		radioType.SetSelected(currentType)
	}

	targetSites := []string{
		"google.com",
		"youtube.com",
		"chatgpt.com",
		"web.telegram.org",
		i18n.T("gui.single.custom_site"),
	}

	customEntry := widget.NewEntry()
	customEntry.SetPlaceHolder(i18n.T("gui.single.custom_placeholder"))
	customEntry.SetText(g.customTargetURL)
	customEntry.OnChanged = func(s string) { g.customTargetURL = s }

	customBox := container.NewVBox(widget.NewLabel(i18n.T("gui.single.enter_addr")), customEntry)
	customBox.Hide()

	targetSelect := widget.NewSelect(targetSites, func(s string) {
		if s == i18n.T("gui.single.custom_site") {
			g.isCustomTarget = true
			customBox.Show()
		} else {
			g.isCustomTarget = false
			g.cfg.DestAddr = s
			customBox.Hide()
		}
	})
	targetSelect.PlaceHolder = i18n.T("gui.settings.target_placeholder")

	if g.isCustomTarget {
		targetSelect.SetSelected(i18n.T("gui.single.custom_site"))
		customBox.Show()
	} else if g.cfg.DestAddr != "" {
		targetSelect.SetSelected(g.cfg.DestAddr)
	}

	btnRun := widget.NewButton(i18n.T("gui.btn_run"), func() {
		addr := proxyEntry.Text
		target := g.getTargetURL()

		selectedType := radioType.Selected
		checkType := selectedType
		if selectedType == i18n.T("gui.single.type_all") {
			checkType = "socks5"
		}

		if addr == "" {
			g.appendLog(i18n.T("gui.single.err_empty_addr") + "\n")
			return
		}

		g.showMainScreen()
		g.appendLog(fmt.Sprintf("%s: %s -> %s (%s)...\n", i18n.T("gui.single.log_checking"), addr, target, checkType))
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

			res := services.CheckProxy(ctx, addr, target, checkType, g.cfg.CheckHTTP2)
			if res.Error != nil {
				g.appendLog(fmt.Sprintf("%s %v\n", i18n.T("cli.fail"), res.Error))
				return
			}

			item := ProxyItemWrapper{
				Host:    host,
				Port:    port,
				Type:    common.ProxyType(checkType),
				Country: "N/A",
				TCP:     res.ProxyLatencyStr,
				HTTP:    res.ReqLatencyStr,
			}

			_ = g.listData.Set([]interface{}{item})
			g.appendLog(fmt.Sprintf("%s: %d\n", i18n.T("gui.single.log_done"), res.StatusCode))
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
