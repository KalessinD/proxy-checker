package gui

import (
	"context"
	"fmt"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/config"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"go.uber.org/zap"
)

const (
	themeLight = "light"
	themeDark  = "dark"
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

	logLabel  *widget.Label
	logScroll *container.Scroll
	logBuffer string

	systemProxySupported bool

	customTargetURL string
	isCustomTarget  bool

	cancelFunc context.CancelFunc

	btnCancel      *widget.Button
	btnCheckSingle *widget.Button
	btnCheckList   *widget.Button
	btnSettings    *widget.Button
	switchProxy    *widget.Check
}

func NewAppGUI(cfg *config.Config) *AppGUI {
	a := app.NewWithID(common.AppName)

	gui := &AppGUI{
		app:      a,
		window:   a.NewWindow(common.AppName),
		cfg:      cfg,
		progress: binding.NewFloat(),
		listData: binding.NewUntypedList(),
	}

	gui.window.Resize(fyne.NewSize(800, 600))
	gui.applyTheme(cfg.Theme)
	gui.systemProxySupported = isSystemProxySupported()

	gui.btnSettings = widget.NewButton(i18n.T("gui.btn_settings"), func() {
		gui.showSettingsScreen()
	})

	gui.switchProxy = widget.NewCheck("", func(checked bool) {
		if !gui.systemProxySupported {
			gui.appendLog(i18n.T("gui.sys_proxy_unsupported"))
			gui.switchProxy.SetChecked(false)
			return
		}

		var mode string
		if checked {
			mode = "manual"
		} else {
			mode = "none"
		}

		if err := setSystemProxyMode(mode); err != nil {
			gui.appendLog(fmt.Sprintf(i18n.T("gui.sys_proxy_error"), err))
			gui.switchProxy.SetChecked(!checked)
		} else {
			gui.appendLog(fmt.Sprintf(i18n.T("gui.sys_proxy_mode_changed"), mode))
		}
	})

	if !gui.systemProxySupported {
		gui.switchProxy.Disable()
	}

	gui.btnCheckSingle = widget.NewButton(i18n.T("gui.btn_check_single"), func() {
		gui.showSingleCheckScreen()
	})

	gui.btnCheckList = widget.NewButton(i18n.T("gui.btn_check_list"), func() {
		go gui.runBatchCheck()
	})

	gui.btnCancel = widget.NewButton(i18n.T("gui.btn_cancel"), func() {
		if gui.cancelFunc != nil {
			gui.cancelFunc()
			gui.appendLog(i18n.T("gui.log_stopped"))
		}
	})
	gui.btnCancel.Importance = widget.DangerImportance
	gui.btnCancel.Disable()

	if gui.systemProxySupported {
		currentMode, err := getSystemProxyMode()
		if err != nil {
			gui.appendLog(fmt.Sprintf(i18n.T("gui.sys_proxy_status_error"), err))
		} else if currentMode == "manual" {
			gui.switchProxy.SetChecked(true)
		}
	}

	return gui
}

// appendLog безопасно добавляет текст в логи из любого потока
func (g *AppGUI) appendLog(text string) {
	g.logBuffer += text

	cleanText := strings.TrimSpace(text)
	if cleanText != "" {
		if strings.Contains(strings.ToLower(cleanText), "ошибка") || strings.Contains(strings.ToLower(cleanText), "error") {
			zap.S().Error(cleanText)
		} else {
			zap.S().Info(cleanText)
		}
	}

	if g.logLabel != nil {
		fyne.Do(func() {
			if g.logLabel != nil && g.logScroll != nil {
				g.logLabel.SetText(g.logBuffer)
				g.logScroll.ScrollToBottom()
			}
		})
	}
}

func (g *AppGUI) applyTheme(themeName string) {
	switch strings.ToLower(themeName) {
	case themeLight:
		// nolint staticcheck
		g.app.Settings().SetTheme(theme.LightTheme())
	case themeDark:
		// nolint staticcheck
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
