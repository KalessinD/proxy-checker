package gui

import (
	"image/color"
	images "proxy-checker/assets"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const (
	iconWidth  = 320
	iconHeight = 320
)

// ShowAboutDialog creates and displays the About modal popup.
func (g *AppGUI) ShowAboutDialog() {
	versionLabel := widget.NewLabel(i18n.T("gui.about.version") + g.version)
	versionLabel.Wrapping = fyne.TextWrapWord

	copyrightLabel := widget.NewLabel(i18n.T("gui.about.copyright"))
	copyrightLabel.Wrapping = fyne.TextWrapWord

	leftContent := container.NewVBox(versionLabel, copyrightLabel)

	iconWidget := g.loadAboutIcon()
	rightContent := container.NewVBox(iconWidget)

	dialogContent := container.New(
		layout.NewGridLayoutWithColumns(2),
		container.NewPadded(leftContent),
		container.NewCenter(rightContent),
	)

	dialogTitle := i18n.T("gui.about.title")
	closeButtonText := i18n.T("gui.btn_close")

	customDialog := dialog.NewCustom(dialogTitle, closeButtonText, dialogContent, g.window)
	customDialog.SetDismissText(closeButtonText)
	customDialog.Resize(fyne.NewSize(640, 480))
	customDialog.Show()
}

// loadAboutIcon loads the application icon from the embedded binary data.
// This ensures the image is always available, regardless of the external file system state.
func (g *AppGUI) loadAboutIcon() fyne.CanvasObject {
	if len(images.Icon) == 0 {
		g.appendLog(common.LogLevelWarn, "About dialog: embedded icon data is empty")

		placeholder := canvas.NewRectangle(color.Transparent)
		placeholder.SetMinSize(fyne.NewSize(iconWidth, iconHeight))
		placeholder.CornerRadius = 0

		return placeholder
	}

	iconResource := fyne.NewStaticResource("proxy-checker.png", images.Icon)
	iconImage := canvas.NewImageFromResource(iconResource)
	iconImage.FillMode = canvas.ImageFillContain
	iconImage.SetMinSize(fyne.NewSize(iconWidth, iconHeight))

	return iconImage
}
