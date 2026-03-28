package gui

import (
	"context"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
)

type ProxyItemWrapper struct {
	Host    string
	Port    string
	Type    common.ProxyType
	Country string
	TCP     string
	HTTP    string
}

type AppGUI struct {
	app    fyne.App
	window fyne.Window
	cfg    *config.Config

	progress binding.Float
	listData binding.UntypedList

	progressBar *widget.ProgressBar
	table       *widget.Table

	// ИСПРАВЛЕНИЕ: Заменяем logText binding.String на правильные компоненты для логов
	logLabel  *widget.Label
	logScroll *container.Scroll
	logBuffer string

	systemProxySupported bool

	customTargetURL string
	isCustomTarget  bool

	cancelFunc     context.CancelFunc
	btnCheckList   *widget.Button
	btnCheckSingle *widget.Button
	btnCancel      *widget.Button
}

func NewAppGUI(cfg *config.Config) *AppGUI {
	a := app.NewWithID("Proxy Checker")

	gui := &AppGUI{
		app:      a,
		window:   a.NewWindow("Proxy Checker"),
		cfg:      cfg,
		progress: binding.NewFloat(),
		listData: binding.NewUntypedList(),
	}

	gui.window.Resize(fyne.NewSize(800, 600))
	gui.applyTheme(cfg.Theme)
	gui.systemProxySupported = isSystemProxySupported()

	return gui
}

// appendLog добавляет текст в логи и автоматически прокручивает их вниз
func (g *AppGUI) appendLog(text string) {
	g.logBuffer += text
	g.logLabel.SetText(g.logBuffer)
	g.logScroll.ScrollToBottom()
}

func (g *AppGUI) applyTheme(themeName string) {
	switch strings.ToLower(themeName) {
	case "light":
		g.app.Settings().SetTheme(theme.LightTheme())
	case "dark":
		g.app.Settings().SetTheme(theme.DarkTheme())
	default:
		g.app.Settings().SetTheme(nil)
	}
}

func (g *AppGUI) Run() {
	g.showMainScreen()
	g.window.ShowAndRun()
}

func (g *AppGUI) getTargetURL() string {
	if g.isCustomTarget && g.customTargetURL != "" {
		return g.customTargetURL
	}
	return g.cfg.DestAddr
}

func Run(cfg *config.Config) {
	gui := NewAppGUI(cfg)
	gui.Run()
}
