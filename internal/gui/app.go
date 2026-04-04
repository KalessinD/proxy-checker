package gui

import (
	"context"
	"fmt"
	"image/color"
	"proxy-checker/internal/cache"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/config"
	"proxy-checker/internal/services"
	"proxy-checker/internal/sysproxy"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	themeLight  = "light"
	themeDark   = "dark"
	themeSystem = "system"
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
		cache  cache.StorageInterface
		logger common.LoggerInterface

		progress   binding.Float
		proxyItems []*ProxyItemWrapper

		progressBar *widget.ProgressBar
		table       *widget.Table

		logRichText *widget.RichText
		logScroll   *container.Scroll
		logMutex    sync.Mutex

		sysProxyManager sysproxy.SystemProxyManager

		highlightedRow   int
		isDarkTheme      bool
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
		g.appendLog(common.LogLevelError, fmt.Sprintf("%s: %v\n", i18n.T("gui.settings.geoip_error"), err))
	}

	if customPath != "" {
		resolver, err := common.NewMaxMindDBResolverFromFile(customPath)
		if err == nil {
			g.geoIPResolver = resolver
			g.isGeoIPAvailable = true
			return
		}
		g.appendLog(common.LogLevelError, fmt.Sprintf("%s: %v\n", i18n.T("gui.settings.geoip_error"), err))
	}
}

func NewAppGUI(fyneApp fyne.App, cfg *config.Config, logger common.LoggerInterface) *AppGUI {
	gui := &AppGUI{
		app:             fyneApp,
		window:          fyneApp.NewWindow(common.AppName),
		cfg:             cfg,
		logger:          logger,
		progress:        binding.NewFloat(),
		proxyItems:      make([]*ProxyItemWrapper, 0),
		cache:           cache.NewFileStorage(logger),
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
			g.appendLog(common.LogLevelError, i18n.T("gui.sys_proxy_unsupported")+"\n")
			g.switchProxy.SetChecked(false)
			return
		}

		mode := sysproxy.ProxyModeNone
		if checked {
			mode = sysproxy.ProxyModeManual
		}

		if err := g.sysProxyManager.SetMode(mode); err != nil {
			g.appendLog(common.LogLevelError, fmt.Sprintf("%s: %v\n", i18n.T("gui.sys_proxy_error"), err))
			g.switchProxy.SetChecked(!checked)
		} else {
			g.appendLog(common.LogLevelInfo, fmt.Sprintf("%s: %s\n", i18n.T("gui.sys_proxy_mode_changed"), mode))
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
			g.appendLog(common.LogLevelInfo, i18n.T("gui.log_stopped")+"\n")
		}
	})
	g.btnCancel.Importance = widget.DangerImportance
	g.btnCancel.Disable()

	g.logRichText = widget.NewRichText()
	g.logRichText.Wrapping = fyne.TextWrapWord
	g.logScroll = container.NewScroll(g.logRichText)
}

func (g *AppGUI) loadSystemProxyState() {
	if !g.sysProxyManager.IsSupported() {
		g.switchProxy.Disable()
		return
	}

	currentMode, err := g.sysProxyManager.GetMode()
	if err != nil {
		g.appendLog(common.LogLevelError, fmt.Sprintf("%s: %v\n", i18n.T("gui.sys_proxy_status_error"), err))
	} else if currentMode == sysproxy.ProxyModeManual {
		g.switchProxy.SetChecked(true)
	}
}

func (g *AppGUI) appendLog(level common.LogLevel, text string) {
	g.logMutex.Lock()
	defer g.logMutex.Unlock()

	cleanText := strings.TrimSpace(text)

	// Determine the text style based on the log level
	segmentStyle := widget.RichTextStyleInline
	if level == common.LogLevelError {
		segmentStyle.ColorName = theme.ColorNameError
	}

	newSegment := &widget.TextSegment{
		Text:  text,
		Style: segmentStyle,
	}

	g.logRichText.Segments = append(g.logRichText.Segments, newSegment)

	if cleanText != "" {
		switch {
		case level == common.LogLevelError:
			g.logger.Error(cleanText)
		case level == common.LogLevelWarn:
			g.logger.Warn(cleanText)
		default:
			g.logger.Info(cleanText)
		}
	}

	if g.logRichText != nil {
		fyne.Do(func() {
			if g.logRichText != nil && g.logScroll != nil {
				g.logRichText.Refresh()
				g.logScroll.ScrollToBottom()
			}
		})
	}
}

func (g *AppGUI) applyTheme(themeName string) {
	switch strings.ToLower(themeName) {
	case themeLight:
		g.isDarkTheme = false
		g.app.Settings().SetTheme(&forcedVariantTheme{
			Theme:   theme.DefaultTheme(),
			variant: theme.VariantLight,
		})
	case themeDark:
		g.isDarkTheme = true
		g.app.Settings().SetTheme(&forcedVariantTheme{
			Theme:   theme.DefaultTheme(),
			variant: theme.VariantDark,
		})
	case themeSystem:
	default:
		g.isDarkTheme = false
		g.app.Settings().SetTheme(nil)
	}
}

// loadCacheForSource loads data from cache for the specified source and proxy type.
// If the cache is empty or expired, it clears the current list.
func (g *AppGUI) loadCacheForSource(source common.Source, proxyType common.ProxyType) {
	cachedItems, err := g.cache.Load(source, proxyType)
	if err != nil {
		g.appendLog(common.LogLevelError, fmt.Sprintf("%s: %v\n", i18n.T("gui.log_cache_error"), err))
		return
	}

	if len(cachedItems) == 0 {
		fyne.Do(func() {
			g.proxyItems = []*ProxyItemWrapper{}
			if g.table != nil {
				g.table.Refresh()
			}
		})
		return
	}

	items := g.mapToWrapper(cachedItems)

	fyne.Do(func() {
		g.proxyItems = items
		if g.table != nil {
			g.table.Refresh()
		}
	})

	g.appendLog(common.LogLevelInfo, fmt.Sprintf("%s: %d\n", i18n.T("gui.log_cache_loaded"), len(cachedItems)))
}

func (g *AppGUI) mapToWrapper(items []*services.ProxyItemFull) []*ProxyItemWrapper {
	wrappers := make([]*ProxyItemWrapper, len(items))
	for i, item := range items {
		wrappers[i] = &ProxyItemWrapper{
			Host: item.Host, Port: item.Port, Type: item.Type, Country: item.Country,
			TCP: item.CheckResult.ProxyLatencyStr, HTTP: item.CheckResult.ReqLatencyStr,
		}
	}
	return wrappers
}

func (g *AppGUI) Run() {
	g.showMainScreen()

	g.loadCacheForSource(g.cfg.Source, g.cfg.Type)
	g.restoreSystemProxyHighlight()

	g.window.ShowAndRun()
}

func (g *AppGUI) restoreSystemProxyHighlight() {
	if !g.sysProxyManager.IsSupported() {
		return
	}

	host, port, err := g.sysProxyManager.GetActiveProxy()
	if err != nil {
		g.appendLog(common.LogLevelWarn, fmt.Sprintf("%s: %v\n", i18n.T("gui.sys_proxy_status_error"), err))
		return
	}

	if host != "" && port != "" {
		g.highlightProxyInList(host, port)
	}
}

func (g *AppGUI) highlightProxyInList(host, port string) {
	g.highlightedRow = -1
	for i, item := range g.proxyItems {
		if item.Host == host && item.Port == port {
			g.highlightedRow = i
			break
		}
	}
	if g.table != nil {
		g.table.Refresh()
	}
}

func (g *AppGUI) getTargetURL() string {
	if g.isCustomTarget && g.customTargetURL != "" {
		return g.customTargetURL
	}
	return g.cfg.DestAddr
}

func Run(cfg *config.Config, logger common.LoggerInterface) {
	fyneApp := app.NewWithID(common.AppName)
	gui := NewAppGUI(fyneApp, cfg, logger)
	gui.Run()
}

func (g *AppGUI) buildTargetSelector() (*widget.Select, *widget.Entry, *fyne.Container) {
	targetSites := []string{
		"google.com",
		"youtube.com",
		"chatgpt.com",
		"web.telegram.org",
		i18n.T("gui.single.custom_site"),
	}

	customEntry := widget.NewEntry()
	customEntry.SetPlaceHolder(i18n.T("gui.single.custom_placeholder"))
	customEntry.OnChanged = func(s string) { g.customTargetURL = s }

	customBox := container.NewVBox(widget.NewLabel(i18n.T("gui.single.enter_addr")), customEntry)
	customBox.Hide()

	targetSelect := widget.NewSelect(targetSites, func(s string) {
		if s == i18n.T("gui.single.custom_site") {
			g.isCustomTarget = true
			customBox.Show()
		} else {
			g.isCustomTarget = false
			g.cfg.DestAddr = s
			g.customTargetURL = ""
			customBox.Hide()
		}
	})
	targetSelect.PlaceHolder = i18n.T("gui.settings.target_placeholder")

	return targetSelect, customEntry, customBox
}

// restoreTargetSelectorState applies the current AppGUI state (isCustomTarget, customTargetURL, cfg.DestAddr)
// to the target selector widgets so they reflect the actual configuration.
func (g *AppGUI) restoreTargetSelectorState(targetSelect *widget.Select, customEntry *widget.Entry, customBox *fyne.Container) {
	customEntry.SetText(g.customTargetURL)

	if g.isCustomTarget {
		targetSelect.SetSelected(i18n.T("gui.single.custom_site"))
		customBox.Show()
	} else if g.cfg.DestAddr != "" {
		targetSelect.SetSelected(g.cfg.DestAddr)
	}
}
