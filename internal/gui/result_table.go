package gui

import (
	"image/color"
	"proxy-checker/internal/common/i18n"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type (
	tableCell struct {
		widget.BaseWidget

		label     *widget.Label
		btn       *widget.Button
		bg        *canvas.Rectangle
		clipboard fyne.Clipboard
	}

	resizableTable struct {
		widget.BaseWidget
		table        *widget.Table
		header       *fyne.Container
		hasButtonCol bool
		minWidth     float32
		minHeight    float32
	}
)

func newTableCell(clipboard fyne.Clipboard) *tableCell {
	cell := &tableCell{
		label:     widget.NewLabel(""),
		btn:       widget.NewButton(i18n.T("gui.btn_apply"), nil),
		bg:        canvas.NewRectangle(color.Transparent),
		clipboard: clipboard,
	}
	cell.label.Truncation = fyne.TextTruncateClip
	cell.btn.Hide()
	cell.ExtendBaseWidget(cell)
	return cell
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

	proportions := []float32{0.20, 0.20, 0.07, 0.10, 0.13, 0.15, 0.15}

	w.table.SetColumnWidth(0, usableWidth*proportions[0])
	w.table.SetColumnWidth(1, usableWidth*proportions[1])
	w.table.SetColumnWidth(2, usableWidth*proportions[2])
	w.table.SetColumnWidth(3, usableWidth*proportions[3])
	w.table.SetColumnWidth(4, usableWidth*proportions[4])
	w.table.SetColumnWidth(5, usableWidth*proportions[5])
	w.table.SetColumnWidth(6, usableWidth*proportions[6])

	if w.hasButtonCol {
		w.table.SetColumnWidth(7, buttonWidth)
	}
}

func (c *tableCell) setHighlighted(isHighlighted bool, isDarkVariant bool) {
	if isHighlighted {
		if isDarkVariant {
			c.bg.FillColor = color.NRGBA{R: 45, G: 65, B: 95, A: 160}
		} else {
			c.bg.FillColor = color.NRGBA{R: 176, G: 224, B: 255, A: 160}
		}
		c.bg.CornerRadius = 0
	} else {
		c.bg.FillColor = color.Transparent
	}
	c.bg.Refresh()
}

// TappedSecondary handles the right-click event to show the copy context menu.
func (c *tableCell) TappedSecondary(pe *fyne.PointEvent) {
	if c.label.Text == "" || c.btn.Visible() {
		return
	}
	copyItem := fyne.NewMenuItem(i18n.T("gui.ctx_copy"), func() {
		if c.clipboard != nil {
			c.clipboard.SetContent(c.label.Text)
		}
	})

	menu := fyne.NewMenu("", copyItem)
	canvas := fyne.CurrentApp().Driver().CanvasForObject(c)
	if canvas != nil {
		widget.NewPopUpMenu(menu, canvas).ShowAtPosition(pe.AbsolutePosition)
	}
}

func (c *tableCell) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(c.bg, c.label, c.btn))
}
