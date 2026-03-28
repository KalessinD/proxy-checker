package gui

import (
	"context"
	"fmt"
	"strings"

	"proxy-checker/internal/services"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

// tableCell кастомный виджет для одной ячейки таблицы.
type tableCell struct {
	widget.BaseWidget
	label *widget.Label
	btn   *widget.Button
}

func newTableCell() *tableCell {
	c := &tableCell{
		label: widget.NewLabel(""),
		btn:   widget.NewButton("Применить", nil),
	}
	c.label.Truncation = fyne.TextTruncateClip
	c.btn.Hide()
	c.ExtendBaseWidget(c)
	return c
}

func (c *tableCell) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(c.label, c.btn))
}

func (c *tableCell) updateText(text string) {
	c.label.SetText(text)
	c.label.Show()
	c.btn.Hide()
}

func (c *tableCell) updateButton(onTapped func()) {
	c.btn.OnTapped = onTapped
	c.btn.Show()
	c.label.Hide()
}

// resizableTable — это кастомный виджет-обертка, который автоматически
// пересчитывает ширину колонок при изменении размера окна.
type resizableTable struct {
	widget.BaseWidget
	table        *widget.Table
	header       *fyne.Container
	hasButtonCol bool
	minWidth     float32
	minHeight    float32
}

func newResizableTable(table *widget.Table, header *fyne.Container, hasButtonCol bool, minW, minH float32) *resizableTable {
	w := &resizableTable{
		table:        table,
		header:       header,
		hasButtonCol: hasButtonCol,
		minWidth:     minW,
		minHeight:    minH,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *resizableTable) CreateRenderer() fyne.WidgetRenderer {
	// Помещаем заголовок сверху, а таблицу на оставшееся место
	c := container.NewBorder(w.header, nil, nil, nil, w.table)
	return widget.NewSimpleRenderer(c)
}

func (w *resizableTable) MinSize() fyne.Size {
	return fyne.NewSize(w.minWidth, w.minHeight)
}

// МАГИЯ ЗДЕСЬ: Fyne вызывает этот метод каждый раз при ресайзе окна
func (w *resizableTable) Resize(size fyne.Size) {
	// Сначала даем базовому виджету обновить свой внутренний размер
	w.BaseWidget.Resize(size)

	// Затем пересчитываем ширину колонок таблицы исходя из НОВОЙ ширины
	w.updateColumnWidths(size.Width)
}

func (w *resizableTable) updateColumnWidths(availableWidth float32) {
	// Не даем таблице сжиматься меньше заданного минимума
	if availableWidth < w.minWidth {
		availableWidth = w.minWidth
	}

	buttonWidth := float32(0)
	if w.hasButtonCol {
		buttonWidth = 110 // Фиксированная ширина под кнопку
	}

	// Оставшееся пространство делим пропорционально
	usableWidth := availableWidth - buttonWidth

	// Пропорции: Host(30%), Port(8%), Type(10%), Country(15%), TCP(18.5%), HTTP(18.5%)
	proportions := []float32{0.30, 0.08, 0.10, 0.15, 0.185, 0.185}

	w.table.SetColumnWidth(0, usableWidth*proportions[0])
	w.table.SetColumnWidth(1, usableWidth*proportions[1])
	w.table.SetColumnWidth(2, usableWidth*proportions[2])
	w.table.SetColumnWidth(3, usableWidth*proportions[3])
	w.table.SetColumnWidth(4, usableWidth*proportions[4])
	w.table.SetColumnWidth(5, usableWidth*proportions[5])

	if w.hasButtonCol {
		w.table.SetColumnWidth(6, buttonWidth)
	}
}

// --- Конец вспомогательных структур ---

func (g *AppGUI) showMainScreen() {
	btnSettings := widget.NewButton("Настройки", func() {
		g.showSettingsScreen()
	})

	g.btnCheckSingle = widget.NewButton("Проверить один прокси", func() {
		g.showSingleCheckScreen()
	})

	g.btnCheckList = widget.NewButton("Проверить по источнику", func() {
		go g.runBatchCheck()
	})

	g.btnCancel = widget.NewButton("Прервать", func() {
		if g.cancelFunc != nil {
			g.cancelFunc()
			g.logText.Set("Проверка прервана пользователем.\n")
		}
	})
	g.btnCancel.Importance = widget.DangerImportance
	g.btnCancel.Hide()

	rightButtons := container.NewHBox(
		g.btnCancel,
		g.btnCheckSingle,
		g.btnCheckList,
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
	g.progressBar = progressBar
	// progressBar.Hide()

	topBox := container.NewVBox(
		widget.NewLabel("Логи:"),
		logEntry,
		widget.NewLabel("Прогресс:"),
		progressBar,
	)

	g.table = g.createResultTable()

	// Создаем заголовок (GridWithColumns сам растянется на ширину таблицы)
	headerObjects := []fyne.CanvasObject{
		widget.NewLabel("Host"), widget.NewLabel("Port"), widget.NewLabel("Type"),
		widget.NewLabel("Country"), widget.NewLabel("TCP"), widget.NewLabel("HTTP"),
	}
	if g.systemProxySupported {
		headerObjects = append(headerObjects, widget.NewLabel(""))
	}
	tableHeader := container.NewGridWithColumns(len(headerObjects), headerObjects...)

	// ИСПОЛЬЗУЕМ НАШУ НОВУЮ ОБЕРТКУ ВМЕСТО container.NewBorder И minSizeWidget
	scalableTable := newResizableTable(
		g.table,
		tableHeader,
		g.systemProxySupported,
		600,
		float32(g.cfg.MinHeight),
	)

	content := container.NewBorder(
		topBox,
		buttonsContainer,
		nil,
		nil,
		scalableTable, // Передаем обертку
	)

	g.window.SetContent(content)
}

func (g *AppGUI) setUIState(running bool) {
	if running {
		g.btnCheckList.Disable()
		g.btnCheckSingle.Disable()
		g.btnCancel.Show()
	} else {
		g.btnCheckList.Enable()
		g.btnCheckSingle.Enable()
		g.btnCancel.Hide()
		g.cancelFunc = nil
	}
}

func (g *AppGUI) runBatchCheck() {
	g.logText.Set("Подготовка...\n")
	g.progress.Set(0)
	g.listData.Set([]interface{}{})

	g.progressBar.Show()

	ctx, cancel := context.WithCancel(context.Background())
	g.cancelFunc = cancel

	fyne.DoAndWait(func() {
		g.setUIState(true)
	})
	defer fyne.DoAndWait(func() {
		g.setUIState(false)
		// g.progressBar.Hide()
	})

	currentLog, _ := g.logText.Get()
	g.logText.Set(currentLog + fmt.Sprintf("Загрузка прокси из источника: %s...\n", g.cfg.Source))

	validProxies, err := services.RunPipeline(ctx, g.cfg, services.PipelineCallbacks{
		OnFetched: func(total int) {
			currentLog, _ := g.logText.Get()
			g.logText.Set(currentLog + fmt.Sprintf("Найдено: %d. Проверка...\n", total))
		},
		OnProgress: func(current, total int32) {
			if ctx.Err() == nil {
				g.progress.Set(float64(current) / float64(total))
			}
		},
	})

	if err != nil {
		currentLog, _ := g.logText.Get()
		g.logText.Set(currentLog + fmt.Sprintf("Ошибка получения прокси: %v\n", err))
		return
	}

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
	if ctx.Err() != nil {
		g.logText.Set(currentLog + "Проверка остановлена.\n")
	} else {
		g.logText.Set(currentLog + fmt.Sprintf("Готово. Найдено рабочих: %d\n", len(validProxies)))
	}
	g.progress.Set(1.0)
}

func (g *AppGUI) createResultTable() *widget.Table {
	cols := 6
	if g.systemProxySupported {
		cols = 7
	}

	table := widget.NewTable(
		func() (int, int) {
			length := g.listData.Length()
			return length, cols
		},
		func() fyne.CanvasObject {
			return newTableCell()
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			val, err := g.listData.GetItem(id.Row)
			if err != nil {
				return
			}

			item, _ := val.(binding.Untyped).Get()
			p, ok := item.(ProxyItemWrapper)
			if !ok {
				return
			}

			tc := cell.(*tableCell)

			if g.systemProxySupported && id.Col == 6 {
				h := p.Host
				pt := p.Port
				t := p.Type
				tc.updateButton(func() {
					g.applySystemProxy(h, pt, string(t))
				})
				return
			}

			var text string
			switch id.Col {
			case 0:
				text = p.Host
			case 1:
				text = p.Port
			case 2:
				text = string(p.Type)
			case 3:
				text = p.Country
			case 4:
				text = p.TCP
			case 5:
				text = p.HTTP
			}
			tc.updateText(text)
		},
	)

	// Ширина больше не задается статически здесь!
	// Обертка resizableTable сама вызовет SetColumnWidth при первом отображении и при ресайзе.

	return table
}

func (g *AppGUI) applySystemProxy(host, port, proxyType string) {
	err := setSystemProxy(host, port, proxyType)
	if err != nil {
		g.logText.Set(fmt.Sprintf("Ошибка применения прокси %s:%s (%s): %v\n", host, port, proxyType, err))
	} else {
		g.logText.Set(fmt.Sprintf("Системный прокси изменен: %s://%s:%s\n", strings.ToLower(proxyType), host, port))
	}
}
