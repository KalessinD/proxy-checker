package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type (
	vPad struct {
		widget.BaseWidget
		child  fyne.CanvasObject
		topPad float32
	}

	borderedBox struct {
		widget.BaseWidget
		content fyne.CanvasObject
	}

	minSizeWidget struct {
		widget.BaseWidget
		content fyne.CanvasObject
		minSize fyne.Size
	}
)

// uiVerticalPad is the vertical padding applied to toolbar widgets so they
// align with the surrounding labels.
const uiVerticalPad float32 = 3

func newVPad(child fyne.CanvasObject) *vPad {
	w := &vPad{child: child, topPad: uiVerticalPad}
	w.ExtendBaseWidget(w)
	return w
}

func (v *vPad) CreateRenderer() fyne.WidgetRenderer {
	topSpacer := canvas.NewRectangle(color.Transparent)
	topSpacer.SetMinSize(fyne.NewSize(0, v.topPad))
	return widget.NewSimpleRenderer(container.NewBorder(topSpacer, nil, nil, nil, v.child))
}

func newBorderedBox(content fyne.CanvasObject) *borderedBox {
	w := &borderedBox{content: content}
	w.ExtendBaseWidget(w)
	return w
}

func (b *borderedBox) CreateRenderer() fyne.WidgetRenderer {
	borderRect := canvas.NewRectangle(color.Transparent)
	borderRect.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	borderRect.StrokeWidth = 1
	box := container.NewPadded(container.NewStack(borderRect, b.content))
	return widget.NewSimpleRenderer(box)
}

func newMinSizeWidget(content fyne.CanvasObject, minSize fyne.Size) *minSizeWidget {
	w := &minSizeWidget{content: content, minSize: minSize}
	w.ExtendBaseWidget(w)
	return w
}

func (w *minSizeWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(w.content)
}

func (w *minSizeWidget) MinSize() fyne.Size {
	return w.minSize
}
