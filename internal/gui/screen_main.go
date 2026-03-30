package gui

import (
	"context"
	"fmt"
	"image/color"
	"strings"

	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/services"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ================= Вспомогательные виджеты =================

// vPad добавляет фиксированный отступ сверху для точного выравнивания виджетов
type vPad struct {
	widget.BaseWidget
	child  fyne.CanvasObject
	topPad float32
}

func newVPad(child fyne.CanvasObject, topPad float32) *vPad {
	w := &vPad{child: child, topPad: topPad}
	w.ExtendBaseWidget(w)
	return w
}

func (v *vPad) CreateRenderer() fyne.WidgetRenderer {
	// Прозрачный прямоугольник, который будет выполнять роль жесткого отступа
	topSpacer := canvas.NewRectangle(color.Transparent)
	topSpacer.SetMinSize(fyne.NewSize(0, v.topPad))

	// Border layout прижмет child вниз, оставив topSpacer сверху
	return widget.NewSimpleRenderer(container.NewBorder(topSpacer, nil, nil, nil, v.child))
}

// borderedBox обертка, которая рисует рамку вокруг любого виджета
type borderedBox struct {
	widget.BaseWidget
	content fyne.CanvasObject
}

func newBorderedBox(content fyne.CanvasObject) *borderedBox {
	w := &borderedBox{content: content}
	w.ExtendBaseWidget(w)
	return w
}

func (b *borderedBox) CreateRenderer() fyne.WidgetRenderer {
	borderRect := canvas.NewRectangle(color.Transparent)
	borderRect.StrokeColor = theme.InputBorderColor()
	borderRect.StrokeWidth = 1

	box := container.NewPadded(container.NewMax(borderRect, b.content))
	return widget.NewSimpleRenderer(box)
}

// minSizeWidget обертка для задания минимального размера любому контейнеру
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

// tableCell кастомный виджет для одной ячейки таблицы.
type tableCell struct {
	widget.BaseWidget
	label *widget.Label
	btn   *widget.Button
}

func newTableCell() *tableCell {
	c := &tableCell{
		label: widget.NewLabel(""),
		btn:   widget.NewButton(i18n.T("gui.btn_apply"), nil),
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

// resizableTable — кастомный виджет-обертка для автоматического ресайза колонок
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
	c := container.NewBorder(w.header, nil, nil, nil, w.table)
	return widget.NewSimpleRenderer(c)
}

func (w *resizableTable) MinSize() fyne.Size {
	return fyne.NewSize(w.minWidth, w.minHeight)
}

// Resize вызывается Fyne каждый раз при изменении размера окна
func (w *resizableTable) Resize(size fyne.Size) {
	w.BaseWidget.Resize(size)
	w.updateColumnWidths(size.Width)
}

func (w *resizableTable) updateColumnWidths(availableWidth float32) {
	if availableWidth < w.minWidth {
		availableWidth = w.minWidth
	}

	buttonWidth := float32(0)
	if w.hasButtonCol {
		buttonWidth = 100
	}

	const rightMargin float32 = 25
	totalTableWidth := availableWidth - rightMargin
	usableWidth := totalTableWidth - buttonWidth

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

// ================= Логика главного экрана =================

func (g *AppGUI) showMainScreen() {
	rightButtons := container.NewHBox(
		g.btnCancel,
		g.btnCheckSingle,
		g.btnCheckList,
	)

	var leftSide fyne.CanvasObject
	if g.systemProxySupported {
		proxyLabel := widget.NewLabelWithStyle(i18n.T("gui.label_sys_proxy"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

		// ИЗМЕНЕНИЕ: Оборачиваем лейбл и чекбокс в vPad со сдвигом 3 пикселя вниз
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

	g.logLabel = widget.NewLabel("")
	g.logLabel.Wrapping = fyne.TextWrapWord
	g.logScroll = container.NewScroll(g.logLabel)

	logArea := newMinSizeWidget(g.logScroll, fyne.NewSize(0, 150))

	progressBar := widget.NewProgressBarWithData(g.progress)
	g.progressBar = progressBar

	topBox := container.NewVBox(
		widget.NewLabel(i18n.T("gui.label_logs")),
		logArea,
		widget.NewLabel(i18n.T("gui.label_progress")),
		progressBar,
	)

	g.table = g.createResultTable()

	headerObjects := []fyne.CanvasObject{
		widget.NewLabel(i18n.T("gui.header_host")), widget.NewLabel(i18n.T("gui.header_port")), widget.NewLabel(i18n.T("gui.header_type")),
		widget.NewLabel(i18n.T("gui.header_country")), widget.NewLabel(i18n.T("gui.header_tcp")), widget.NewLabel(i18n.T("gui.header_http")),
	}
	if g.systemProxySupported {
		headerObjects = append(headerObjects, widget.NewLabel(""))
	}
	tableHeader := container.NewGridWithColumns(len(headerObjects), headerObjects...)

	scalableTable := newResizableTable(
		g.table,
		tableHeader,
		g.systemProxySupported,
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
	g.appendLog(i18n.T("gui.log_preparing"))
	_ = g.progress.Set(0)
	_ = g.listData.Set([]interface{}{})

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

	g.appendLog(fmt.Sprintf(i18n.T("gui.log_fetching"), g.cfg.Source))

	validProxies, err := services.RunPipeline(ctx, g.cfg, services.PipelineCallbacks{
		OnFetched: func(total int) {
			g.appendLog(fmt.Sprintf(i18n.T("gui.log_found"), total))
		},
		OnProgress: func(current, total int32) {
			if ctx.Err() == nil {
				_ = g.progress.Set(float64(current) / float64(total))
			}
		},
	})

	if err != nil {
		g.appendLog(fmt.Sprintf(i18n.T("gui.log_fetch_error"), err))
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
	_ = g.listData.Set(items)

	if ctx.Err() != nil {
		g.appendLog(i18n.T("gui.log_stopped"))
	} else {
		g.appendLog(fmt.Sprintf(i18n.T("gui.log_done"), len(validProxies)))
	}
	_ = g.progress.Set(1.0)
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

			tc, _ := cell.(*tableCell)

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

	return table
}

func (g *AppGUI) applySystemProxy(host, port, proxyType string) {
	err := setSystemProxy(host, port, proxyType)
	if err != nil {
		g.appendLog(fmt.Sprintf(i18n.T("gui.log_apply_error"), host, port, proxyType, err))
	} else {
		g.appendLog(fmt.Sprintf(i18n.T("gui.log_apply_success"), strings.ToLower(proxyType), host, port))
		g.switchProxy.SetChecked(true)
	}
}
