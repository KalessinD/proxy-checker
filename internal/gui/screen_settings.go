package gui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (g *AppGUI) showSettingsScreen() {
	// 1. Тип прокси
	proxyTypes := []string{"http", "https", "socks4", "socks5", "все"}
	radioType := widget.NewRadioGroup(proxyTypes, func(s string) {
		if s == "все" {
			g.cfg.Type = "all"
		} else {
			g.cfg.Type = s
		}
	})
	currentType := g.cfg.Type
	if currentType == "all" {
		radioType.SetSelected("все")
	} else {
		radioType.SetSelected(currentType)
	}

	// 2. RTT
	rttOptions := []string{}
	for i := 50; i <= 500; i += 50 {
		rttOptions = append(rttOptions, strconv.Itoa(i))
	}
	selectRTT := widget.NewSelect(rttOptions, func(s string) {
		val, _ := strconv.Atoi(s)
		g.cfg.RTT = val
	})
	selectRTT.SetSelected(strconv.Itoa(g.cfg.RTT))

	// 3. Workers
	workerOptions := []string{"2", "4", "8", "16", "32", "64", "128", "256"}
	selectWorkers := widget.NewSelect(workerOptions, func(s string) {
		val, _ := strconv.Atoi(s)
		g.cfg.Workers = val
	})
	selectWorkers.SetSelected(strconv.Itoa(g.cfg.Workers))

	// 4. Source (ИСПРАВЛЕНО: Список с двумя значениями)
	sources := []string{"proxymania", "thespeedx"}
	selectSource := widget.NewSelect(sources, func(s string) {
		g.cfg.Source = s
	})
	// Устанавливаем текущее значение из конфига
	selectSource.SetSelected(g.cfg.Source)

	// 5. Timeout
	timeoutOptions := []string{"1s", "3s", "5s", "10s", "20s", "30s"}
	selectTimeout := widget.NewSelect(timeoutOptions, func(s string) {
		d, _ := time.ParseDuration(s)
		g.cfg.Timeout = d
	})
	currentTimeoutStr := fmt.Sprintf("%ds", int(g.cfg.Timeout.Seconds()))
	selectTimeout.SetSelected(currentTimeoutStr)

	// 6. Pages
	pageOptions := []string{"1", "2", "3", "4", "5"}
	selectPages := widget.NewSelect(pageOptions, func(s string) {
		val, _ := strconv.Atoi(s)
		g.cfg.Pages = val
	})
	selectPages.SetSelected(strconv.Itoa(g.cfg.Pages))

	// 7. Target Site
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

	// 8. Theme
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

	formItems := []*widget.FormItem{
		widget.NewFormItem("Тип прокси:", radioType),
		widget.NewFormItem("Макс. RTT (мс):", selectRTT),
		widget.NewFormItem("Потоки:", selectWorkers),
		widget.NewFormItem("Источник:", selectSource), // Вот этот элемент
		widget.NewFormItem("Таймаут:", selectTimeout),
		widget.NewFormItem("Число страниц:", selectPages),
		widget.NewFormItem("Сайт проверки:", selectTarget),
		widget.NewFormItem("", customBox),
		widget.NewFormItem("Тема интерфейса:", selectTheme),
	}

	settingsForm := widget.NewForm(formItems...)
	buttonsBox := container.NewHBox(btnBack, layout.NewSpacer(), btnSave)

	content := container.NewBorder(
		nil, buttonsBox, nil, nil,
		container.NewVBox(
			widget.NewLabel("Настройки проверки"),
			widget.NewSeparator(),
			settingsForm,
		),
	)

	g.window.SetContent(content)
}
