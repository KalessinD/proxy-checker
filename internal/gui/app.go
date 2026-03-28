package gui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"

	"proxy-checker/internal/config"
)

// ProxyItemWrapper обертка для отображения данных в таблице
type ProxyItemWrapper struct {
	Host    string
	Port    string
	Type    string
	Country string
	TCP     string
	HTTP    string
}

// AppGUI основная структура графического интерфейса
type AppGUI struct {
	app    fyne.App
	window fyne.Window
	cfg    *config.Config

	logText  binding.String
	progress binding.Float
	listData binding.UntypedList

	// Флаг поддержки системного прокси (Linux/GNOME)
	systemProxySupported bool

	customTargetURL string
	isCustomTarget  bool
}

// NewAppGUI создает новый экземпляр GUI
func NewAppGUI(cfg *config.Config) *AppGUI {
	// os.Setenv("FYNE_APP_ID", "Proxy Checker")
	a := app.NewWithID("Proxy Checker")

	gui := &AppGUI{
		app:      a,
		window:   a.NewWindow("Proxy Checker"),
		cfg:      cfg,
		logText:  binding.NewString(),
		progress: binding.NewFloat(),
		listData: binding.NewUntypedList(),
	}

	gui.window.Resize(fyne.NewSize(800, 600))
	gui.applyTheme(cfg.Theme)

	// Проверяем поддержку системы при старте
	gui.systemProxySupported = isSystemProxySupported()

	return gui
}

// applyTheme применяет тему по названию
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

// Run запускает главный цикл приложения
func (g *AppGUI) Run() {
	g.showMainScreen()
	g.window.ShowAndRun()
}

// getTargetURL возвращает целевой URL для проверки прокси.
func (g *AppGUI) getTargetURL() string {
	if g.isCustomTarget && g.customTargetURL != "" {
		return g.customTargetURL
	}
	return g.cfg.DestAddr
}

// Run обертка для запуска из main
func Run(cfg *config.Config) {
	gui := NewAppGUI(cfg)
	gui.Run()
}
