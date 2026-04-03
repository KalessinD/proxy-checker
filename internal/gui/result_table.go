package gui

import (
	"proxy-checker/internal/common/i18n"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type (
	tableCell struct {
		widget.BaseWidget
		label *widget.Label
		btn   *widget.Button
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
