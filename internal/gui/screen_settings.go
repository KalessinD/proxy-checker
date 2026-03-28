package gui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"proxy-checker/internal/common"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (g *AppGUI) showSettingsScreen() {
	proxyTypes := []string{"http", "https", "socks4", "socks5", "все"}
	radioType := widget.NewRadioGroup(proxyTypes, func(s string) {
		if s == "все" {
			g.cfg.Type = common.ProxyAll
		} else {
			g.cfg.Type = common.ProxyType(s)
		}
	})

	currentType := string(g.cfg.Type)
	if g.cfg.Type == common.ProxyAll {
		radioType.SetSelected("все")
	} else {
		radioType.SetSelected(currentType)
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
	rttLabel := widget.NewLabel("Макс. RTT (мс):")

	workerOptions := []string{"2", "8", "16", "32", "64", "128", "256"}
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
	pagesLabel := widget.NewLabel("Число страниц:")

	timeoutOptions := []string{"1s", "3s", "5s", "10s", "20s", "30s"}
	selectTimeout := widget.NewSelect(timeoutOptions, func(s string) {
		d, _ := time.ParseDuration(s)
		g.cfg.Timeout = d
	})
	currentTimeoutStr := fmt.Sprintf("%ds", int(g.cfg.Timeout.Seconds()))
	selectTimeout.SetSelected(currentTimeoutStr)

	targetSites := []string{
		"https://google.com",
		"https://youtube.com",
		"https://chatgpt.com",
		"https://web.telegram.org",
		"Иной сайт",
	}

	customEntry := widget.NewEntry()
	customEntry.SetPlaceHolder("https://example.com")
	customEntry.OnChanged = func(s string) { g.customTargetURL = s }

	customBox := container.NewVBox(widget.NewLabel("Введите адрес:"), customEntry)
	customBox.Hide()

	selectTarget := widget.NewSelect(targetSites, func(s string) {
		if s == "Иной сайт" {
			g.isCustomTarget = true
			customBox.Show()
		} else {
			g.isCustomTarget = false
			g.cfg.DestAddr = s
			g.customTargetURL = ""
			customBox.Hide()
		}
	})
	selectTarget.PlaceHolder = "(Выберите из списка)"

	if g.isCustomTarget {
		selectTarget.SetSelected("Иной сайт")
		customBox.Show()
	} else {
		if g.cfg.DestAddr != "" {
			selectTarget.SetSelected(g.cfg.DestAddr)
		}
	}

	themeLabels := []string{"системная", "светлая", "тёмная"}
	selectTheme := widget.NewSelect(themeLabels, func(s string) {
		var val string
		switch s {
		case "светлая":
			val = "light"
		case "тёмная":
			val = "dark"
		default:
			val = "system"
		}
		g.cfg.Theme = val
		g.applyTheme(val)
	})

	currentThemeLabel := "системная"
	switch strings.ToLower(g.cfg.Theme) {
	case "light":
		currentThemeLabel = "светлая"
	case "dark":
		currentThemeLabel = "тёмная"
	}
	selectTheme.SetSelected(currentThemeLabel)

	rttBox := container.NewGridWithColumns(2, rttLabel, selectRTT)
	pagesBox := container.NewGridWithColumns(2, pagesLabel, selectPages)
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

	btnSave := widget.NewButton("Сохранить", func() {
		if err := g.cfg.SaveToFile(); err != nil {
			g.logText.Set(fmt.Sprintf("Ошибка сохранения: %v\n", err))
		} else {
			g.logText.Set("Настройки сохранены в файл.\n")
		}
		g.showMainScreen()
	})

	btnBack := widget.NewButton("Назад", func() {
		g.showMainScreen()
	})

	settingsContent := container.NewVBox(
		widget.NewLabel("Настройки проверки"),
		widget.NewSeparator(),
		container.NewGridWithColumns(2, widget.NewLabel("Тип прокси:"), radioType),
		container.NewGridWithColumns(2, widget.NewLabel("Источник:"), selectSource),
		dynamicBox,
		container.NewGridWithColumns(2, widget.NewLabel("Потоки:"), selectWorkers),
		container.NewGridWithColumns(2, widget.NewLabel("Таймаут:"), selectTimeout),
		container.NewGridWithColumns(2, widget.NewLabel("Сайт проверки:"), selectTarget),
		customBox,
		widget.NewSeparator(),
		container.NewGridWithColumns(2, widget.NewLabel("Тема интерфейса:"), selectTheme),
	)

	buttonsBox := container.NewHBox(btnBack, layout.NewSpacer(), btnSave)

	content := container.NewBorder(
		nil, buttonsBox, nil, nil,
		settingsContent,
	)

	g.window.SetContent(content)
}
