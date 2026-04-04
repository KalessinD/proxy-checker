package gui

import (
	"image/color"
	"os"
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
	appIconPath = "assets/images/proxy-checker.png"
	iconWidth   = 320
	iconHeight  = 320
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

// loadAboutIcon attempts to load the application icon from the assets directory.
// If the file is missing, it returns a transparent placeholder to keep the layout intact.
func (g *AppGUI) loadAboutIcon() fyne.CanvasObject {
	if _, err := os.Stat(appIconPath); err != nil {
		g.appendLog(common.LogLevelWarn, "About dialog: icon not found at "+appIconPath)

		placeholder := canvas.NewRectangle(color.Transparent)
		placeholder.SetMinSize(fyne.NewSize(iconWidth, iconHeight))
		placeholder.CornerRadius = 0

		return placeholder
	}

	iconImage := canvas.NewImageFromFile(appIconPath)
	iconImage.FillMode = canvas.ImageFillContain
	iconImage.SetMinSize(fyne.NewSize(iconWidth, iconHeight))

	return iconImage
}
