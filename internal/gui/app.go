package gui

import (
	"context"
	"fmt"
	"image/color"
	"proxy-checker/internal/cache"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/config"
	"proxy-checker/internal/sysproxy"
	"strings"
	"sync"

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

type (
	ProxyItemWrapper struct {
		Host    string           `json:"host"`
		Port    string           `json:"port"`
		Type    common.ProxyType `json:"type"`
		Country string           `json:"country"`
		TCP     string           `json:"tcp"`
		HTTP    string           `json:"http"`
	}

	AppGUI struct {
		app    fyne.App
		window fyne.Window
		cfg    *config.Config
		cache  cache.Storage

		progress   binding.Float
		proxyItems []*ProxyItemWrapper

		progressBar *widget.ProgressBar
		table       *widget.Table

		logLabel  *widget.Label
		logScroll *container.Scroll
		logBuffer string
		logMutex  sync.Mutex

		sysProxyManager sysproxy.SystemProxyManager

		isGeoIPAvailable bool
		geoIPResolver    common.GeoIPResolver

		customTargetURL string
		isCustomTarget  bool

		cancelFunc context.CancelFunc

		btnCancel      *widget.Button
		btnCheckSingle *widget.Button
		btnCheckList   *widget.Button
		btnSettings    *widget.Button
		switchProxy    *widget.Check
	}

	forcedVariantTheme struct {
		fyne.Theme
		variant fyne.ThemeVariant
	}
)

func (t *forcedVariantTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return t.Theme.Color(name, t.variant)
}

func (g *AppGUI) initGeoIP(customPath string) {
	if g.geoIPResolver != nil {
		_ = g.geoIPResolver.Close()
		g.geoIPResolver = nil
		g.isGeoIPAvailable = false
	}

	if len(common.GeoIPData) > 0 {
		resolver, err := common.NewMaxMindDBResolverFromBytes(common.GeoIPData)
		if err == nil {
			g.geoIPResolver = resolver
			g.isGeoIPAvailable = true
			return
		}
		g.appendLog(fmt.Sprintf("%s: %v\n", i18n.T("gui.settings.geoip_error"), err))
	}

	if customPath != "" {
		resolver, err := common.NewMaxMindDBResolverFromFile(customPath)
		if err == nil {
			g.geoIPResolver = resolver
			g.isGeoIPAvailable = true
			return
		}
		g.appendLog(fmt.Sprintf("%s: %v\n", i18n.T("gui.settings.geoip_error"), err))
	}
}

func NewAppGUI(cfg *config.Config) *AppGUI {
	a := app.NewWithID(common.AppName)

	gui := &AppGUI{
		app:             a,
		window:          a.NewWindow(common.AppName),
		cfg:             cfg,
		progress:        binding.NewFloat(),
		proxyItems:      make([]*ProxyItemWrapper, 0),
		cache:           cache.NewFileCache(),
		sysProxyManager: sysproxy.NewSystemProxyManager(),
	}

	gui.window.Resize(fyne.NewSize(800, 600))
	gui.applyTheme(cfg.Theme)

	gui.initGeoIP(cfg.GeoIPDBPath)
	gui.initUIComponents()
	gui.loadSystemProxyState()

	return gui
}

func (g *AppGUI) initUIComponents() {
	g.btnSettings = widget.NewButton(i18n.T("gui.btn_settings"), func() {
		g.showSettingsScreen()
	})

	g.switchProxy = widget.NewCheck("", func(checked bool) {
		if !g.sysProxyManager.IsSupported() {
			g.appendLog(i18n.T("gui.sys_proxy_unsupported") + "\n")
			g.switchProxy.SetChecked(false)
			return
		}

		mode := sysproxy.ProxyModeNone
		if checked {
			mode = sysproxy.ProxyModeManual
		}

		if err := g.sysProxyManager.SetMode(mode); err != nil {
			g.appendLog(fmt.Sprintf("%s: %v\n", i18n.T("gui.sys_proxy_error"), err))
			g.switchProxy.SetChecked(!checked)
		} else {
			g.appendLog(fmt.Sprintf("%s: %s\n", i18n.T("gui.sys_proxy_mode_changed"), mode))
		}
	})

	g.btnCheckSingle = widget.NewButton(i18n.T("gui.btn_check_single"), func() {
		g.showSingleCheckScreen()
	})

	g.btnCheckList = widget.NewButton(i18n.T("gui.btn_check_list"), func() {
		go g.runBatchCheck()
	})

	g.btnCancel = widget.NewButton(i18n.T("gui.btn_cancel"), func() {
		if g.cancelFunc != nil {
			g.cancelFunc()
			g.appendLog(i18n.T("gui.log_stopped") + "\n")
		}
	})
	g.btnCancel.Importance = widget.DangerImportance
	g.btnCancel.Disable()

	g.logLabel = widget.NewLabel("")
	g.logLabel.Wrapping = fyne.TextWrapWord
	g.logScroll = container.NewScroll(g.logLabel)
}

func (g *AppGUI) loadSystemProxyState() {
	if !g.sysProxyManager.IsSupported() {
		g.switchProxy.Disable()
		return
	}

	currentMode, err := g.sysProxyManager.GetMode()
	if err != nil {
		g.appendLog(fmt.Sprintf("%s: %v\n", i18n.T("gui.sys_proxy_status_error"), err))
	} else if currentMode == sysproxy.ProxyModeManual {
		g.switchProxy.SetChecked(true)
	}
}

func (g *AppGUI) appendLog(text string) {
	g.logMutex.Lock()
	g.logBuffer += text
	cleanText := strings.TrimSpace(text)
	g.logMutex.Unlock()

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
		g.app.Settings().SetTheme(&forcedVariantTheme{
			Theme:   theme.DefaultTheme(),
			variant: theme.VariantLight,
		})
	case themeDark:
		g.app.Settings().SetTheme(&forcedVariantTheme{
			Theme:   theme.DefaultTheme(),
			variant: theme.VariantDark,
		})
	default:
		g.app.Settings().SetTheme(nil)
	}
}

func (g *AppGUI) Run() {
	g.showMainScreen()

	cachedItems, err := g.cache.Load(g.cfg)
	if err != nil {
		g.appendLog(fmt.Sprintf("%s: %v\n", i18n.T("gui.log_cache_error"), err))
	} else if len(cachedItems) > 0 {
		items := make([]*ProxyItemWrapper, len(cachedItems))
		for i, item := range cachedItems {
			items[i] = &ProxyItemWrapper{
				Host:    item.Host,
				Port:    item.Port,
				Type:    item.Type,
				Country: item.Country,
				TCP:     item.CheckResult.ProxyLatencyStr,
				HTTP:    item.CheckResult.ReqLatencyStr,
			}
		}
		fyne.Do(func() {
			g.proxyItems = items
			if g.table != nil {
				g.table.Refresh()
			}
		})
		g.appendLog(fmt.Sprintf("%s: %d\n", i18n.T("gui.log_cache_loaded"), len(cachedItems)))
	}

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
