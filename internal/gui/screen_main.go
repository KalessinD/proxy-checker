package gui

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"proxy-checker/internal/services"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

// minSizeWidget - вспомогательная структура для задания минимального размера
type minSizeWidget struct {
	widget.BaseWidget
	content fyne.CanvasObject
	minSize fyne.Size
}

func newMinSizeWidget(content fyne.CanvasObject, min fyne.Size) *minSizeWidget {
	w := &minSizeWidget{content: content, minSize: min}
	w.ExtendBaseWidget(w)
	return w
}

func (w *minSizeWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(w.content)
}

func (w *minSizeWidget) MinSize() fyne.Size {
	return w.minSize
}

// showMainScreen отрисовывает главный экран
func (g *AppGUI) showMainScreen() {
	btnSettings := widget.NewButton("Настройки", func() {
		g.showSettingsScreen()
	})

	btnCheckList := widget.NewButton("Проверить по источнику", func() {
		go g.runBatchCheck()
	})

	btnCheckSingle := widget.NewButton("Проверить один прокси", func() {
		g.showSingleCheckScreen()
	})

	rightButtons := container.NewHBox(
		btnCheckSingle,
		btnCheckList,
	)

	buttonsBar := container.NewBorder(nil, nil, btnSettings, rightButtons)
	buttonsContainer := container.NewVBox(
		widget.NewLabel(""),
		container.NewPadded(buttonsBar),
	)

	logEntry := widget.NewEntryWithData(g.logText)
	logEntry.MultiLine = true
	logEntry.Wrapping = fyne.TextWrapWord

	progressBar := widget.NewProgressBarWithData(g.progress)

	topBox := container.NewVBox(
		widget.NewLabel("Логи:"),
		logEntry,
		widget.NewLabel("Прогресс:"),
		progressBar,
	)

	table := g.createResultTable()

	// Формируем заголовок
	headerObjects := []fyne.CanvasObject{
		widget.NewLabel("Host"), widget.NewLabel("Port"), widget.NewLabel("Type"),
		widget.NewLabel("Country"), widget.NewLabel("TCP"), widget.NewLabel("HTTP"),
	}
	if g.systemProxySupported {
		headerObjects = append(headerObjects, widget.NewLabel("")) // Пустой заголовок для кнопки
	}
	tableHeader := container.NewGridWithColumns(len(headerObjects), headerObjects...)

	tableWithHeader := container.NewBorder(tableHeader, nil, nil, nil, table)
	minTableContainer := newMinSizeWidget(tableWithHeader, fyne.NewSize(0, float32(g.cfg.MinHeight)))

	content := container.NewBorder(
		topBox,
		buttonsContainer,
		nil,
		nil,
		minTableContainer,
	)

	g.window.SetContent(content)
}

// runBatchCheck без изменений
func (g *AppGUI) runBatchCheck() {
	g.logText.Set("Подготовка...\n")
	g.progress.Set(0)
	g.listData.Set([]interface{}{})

	u, _ := url.Parse(g.cfg.Source)
	q := u.Query()

	if g.cfg.Type == "all" {
		q.Del("type")
	} else {
		q.Set("type", strings.ToUpper(g.cfg.Type))
	}
	q.Set("speed", strconv.Itoa(g.cfg.RTT))
	u.RawQuery = q.Encode()
	targetURL := u.String()

	go func() {
		currentLog, _ := g.logText.Get()
		g.logText.Set(currentLog + "Парсинг страниц...\n")

		allProxies, err := services.FetchAllPages(context.Background(), targetURL, g.cfg.Pages)
		if err != nil {
			currentLog, _ := g.logText.Get()
			g.logText.Set(currentLog + fmt.Sprintf("Ошибка парсинга: %v\n", err))
			return
		}

		currentLog, _ = g.logText.Get()
		g.logText.Set(currentLog + fmt.Sprintf("Найдено: %d. Проверка...\n", len(allProxies)))

		validProxies := services.CheckBatch(
			context.Background(),
			allProxies,
			g.getTargetURL(),
			g.cfg.Type,
			g.cfg.Timeout,
			g.cfg.Workers,
			func(current, total int32) {
				g.progress.Set(float64(current) / float64(total))
			},
		)

		items := make([]interface{}, len(validProxies))
		for i, p := range validProxies {
			items[i] = ProxyItemWrapper{
				Host:    p.Host,
				Port:    p.Port,
				Type:    p.Type,
				Country: p.Country,
				TCP:     p.CheckResult.ProxyLatencyStr,
				HTTP:    p.CheckResult.ReqLatencyStr,
			}
		}
		g.listData.Set(items)

		currentLog, _ = g.logText.Get()
		g.logText.Set(currentLog + fmt.Sprintf("Готово. Найдено рабочих: %d\n", len(validProxies)))
		g.progress.Set(1.0)
	}()
}

// createResultTable создает виджет таблицы
func (g *AppGUI) createResultTable() *widget.List {
	// Определяем количество колонок
	cols := 6
	if g.systemProxySupported {
		cols = 7
	}

	return widget.NewListWithData(
		g.listData,
		func() fyne.CanvasObject {
			// Создаем ячейки строки
			hostEntry := widget.NewEntry()
			portEntry := widget.NewEntry()
			typeEntry := widget.NewEntry()
			countryEntry := widget.NewEntry()
			tcpEntry := widget.NewEntry()
			httpEntry := widget.NewEntry()

			rowObjects := []fyne.CanvasObject{
				hostEntry, portEntry, typeEntry, countryEntry, tcpEntry, httpEntry,
			}

			if g.systemProxySupported {
				// Создаем кнопку явно
				applyBtn := widget.NewButton("Применить", nil)
				rowObjects = append(rowObjects, applyBtn)
			}

			return container.NewGridWithColumns(cols, rowObjects...)
		},
		func(id binding.DataItem, obj fyne.CanvasObject) {
			val := id.(binding.Untyped)
			item, _ := val.Get()
			p := item.(ProxyItemWrapper)

			row := obj.(*fyne.Container)

			// Заполняем текстовые поля
			row.Objects[0].(*widget.Entry).SetText(p.Host)
			row.Objects[1].(*widget.Entry).SetText(p.Port)
			row.Objects[2].(*widget.Entry).SetText(p.Type)
			row.Objects[3].(*widget.Entry).SetText(p.Country)
			row.Objects[4].(*widget.Entry).SetText(p.TCP)
			row.Objects[5].(*widget.Entry).SetText(p.HTTP)

			// Если есть кнопка, вешаем логику
			if g.systemProxySupported && len(row.Objects) > 6 {
				btn := row.Objects[6].(*widget.Button)

				// Копируем данные для замыкания, так как переменная p меняется в цикле
				h := p.Host
				pt := p.Port
				t := p.Type

				// Присваиваем функцию полю OnClicked
				btn.OnTapped = func() {
					g.applySystemProxy(h, pt, t)
				}
			}
		},
	)
}

// applySystemProxy обертка для вызова функции применения с логированием
func (g *AppGUI) applySystemProxy(host, port, proxyType string) {
	err := setSystemProxy(host, port, proxyType)
	if err != nil {
		g.logText.Set(fmt.Sprintf("Ошибка применения прокси %s:%s (%s): %v\n", host, port, proxyType, err))
	} else {
		g.logText.Set(fmt.Sprintf("Системный прокси изменен: %s://%s:%s\n", strings.ToLower(proxyType), host, port))
	}
}
