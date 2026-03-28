package gui

import (
	"context"
	"fmt"
	"strings"

	"proxy-checker/internal/common"
	"proxy-checker/internal/services"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (g *AppGUI) showSingleCheckScreen() {
	proxyEntry := widget.NewEntry()
	proxyEntry.SetPlaceHolder("host:port")

	proxyTypes := []string{"http", "https", "socks4", "socks5", "все"}
	radioType := widget.NewRadioGroup(proxyTypes, nil)
	currentType := string(g.cfg.Type)
	if g.cfg.Type == common.ProxyAll {
		radioType.SetSelected("все")
	} else {
		radioType.SetSelected(currentType)
	}

	targetSites := []string{
		"https://google.com",
		"https://youtube.com",
		"https://chatgpt.com",
		"https://web.telegram.org",
		"Иной сайт",
	}

	customEntry := widget.NewEntry()
	customEntry.SetPlaceHolder("https://example.com")
	customEntry.SetText(g.customTargetURL)
	customEntry.OnChanged = func(s string) { g.customTargetURL = s }

	customBox := container.NewVBox(widget.NewLabel("Введите адрес:"), customEntry)
	customBox.Hide()

	targetSelect := widget.NewSelect(targetSites, func(s string) {
		if s == "Иной сайт" {
			g.isCustomTarget = true
			customBox.Show()
		} else {
			g.isCustomTarget = false
			g.cfg.DestAddr = s
			customBox.Hide()
		}
	})
	targetSelect.PlaceHolder = "(Выберите из списка)"

	if g.isCustomTarget {
		targetSelect.SetSelected("Иной сайт")
		customBox.Show()
	} else {
		if g.cfg.DestAddr != "" {
			targetSelect.SetSelected(g.cfg.DestAddr)
		}
	}

	btnRun := widget.NewButton("Запустить проверку", func() {
		addr := proxyEntry.Text
		target := g.getTargetURL()

		selectedType := radioType.Selected
		checkType := selectedType
		if selectedType == "все" {
			checkType = "socks5"
		}

		if addr == "" {
			g.appendLog("Ошибка: введите адрес прокси\n") // ИСПРАВЛЕНО
			return
		}

		g.showMainScreen()
		g.appendLog(fmt.Sprintf("Проверка %s -> %s (Type: %s)...\n", addr, target, checkType)) // ИСПРАВЛЕНО
		g.progress.Set(0)

		go func() {
			parts := strings.Split(addr, ":")
			host := parts[0]
			port := ""
			if len(parts) > 1 {
				port = parts[1]
			}

			ctx, cancel := context.WithTimeout(context.Background(), g.cfg.Timeout)
			defer cancel()

			res := services.CheckProxy(ctx, addr, target, checkType)

			if res.Error != nil {
				g.appendLog(fmt.Sprintf("Ошибка: %v\n", res.Error)) // ИСПРАВЛЕНО
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

			g.listData.Set([]interface{}{item})
			g.appendLog(fmt.Sprintf("Проверка завершена. Статус: %d\n", res.StatusCode)) // ИСПРАВЛЕНО
			g.progress.Set(1.0)
		}()
	})

	btnBack := widget.NewButton("Назад", func() {
		g.showMainScreen()
	})

	buttonsBox := container.NewHBox(btnBack, layout.NewSpacer(), btnRun)

	inputForm := widget.NewForm(
		widget.NewFormItem("Тип прокси:", radioType),
		widget.NewFormItem("Адрес прокси", proxyEntry),
		widget.NewFormItem("Сайт для проверки", targetSelect),
		widget.NewFormItem("", customBox),
	)

	content := container.NewBorder(
		nil,
		buttonsBox,
		nil, nil,
		container.NewVBox(
			widget.NewLabel("Проверка одного прокси"),
			widget.NewSeparator(),
			inputForm,
		),
	)

	g.window.SetContent(content)
}
