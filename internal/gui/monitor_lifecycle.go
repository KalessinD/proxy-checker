package gui

import (
	"context"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/config"
	"proxy-checker/internal/services"
	"sort"
	"time"

	"fyne.io/fyne/v2"
)

// startProxyMonitor launches the background health-check worker
// (TODO item 1: "запуск и останов по кнопке"). Idempotent.
func (g *AppGUI) startProxyMonitor() {
	g.startProxyMonitorWith(services.NewProxyChecker(), newMonitorNotifierAdapter(g))
}

// startProxyMonitorWith is the injectable variant used by tests.
func (g *AppGUI) startProxyMonitorWith(checker services.ProxyChecker, notifier services.MonitorNotifier) {
	g.monitorMu.Lock()
	if g.monitorCancel != nil {
		g.monitorMu.Unlock()
		return // already running
	}
	monitor := services.NewProxyMonitor(
		checker,
		newActiveProxyStoreAdapter(g),
		notifier,
		services.NewCheckParams(g.cfg),
	)
	applyConfigToMonitor(monitor, g.cfg)
	ctx, cancel := context.WithCancel(context.Background())
	g.proxyMonitor = monitor
	g.monitorCancel = cancel
	g.monitorMu.Unlock()

	monitor.Start(ctx)
	g.appendLog(common.LogLevelInfo, i18n.T("gui.log_monitor_started")+"\n")
	g.updateMonitorPool()
}

// stopProxyMonitor cancels the monitor loop and waits for it to exit.
// Idempotent; safe to call after the window has closed: Fyne runs a pending
// fyne.DoAndWait inline once the event loop drains, so Stop cannot deadlock.
func (g *AppGUI) stopProxyMonitor() {
	g.monitorMu.Lock()
	// Invariant: monitor is non-nil whenever cancel is non-nil — both are
	// set and cleared together under this lock. Captured without holding
	// the lock during Stop(): Stop blocks until the loop goroutine exits,
	// and blocking under monitorMu would stall start/stop callers.
	cancel := g.monitorCancel
	monitor := g.proxyMonitor
	g.monitorCancel = nil
	g.proxyMonitor = nil
	g.monitorMu.Unlock()

	if cancel == nil {
		return
	}
	cancel()
	monitor.Stop() // blocks until the loop goroutine exits
	g.appendLog(common.LogLevelInfo, i18n.T("gui.log_monitor_stopped")+"\n")
}

// monitorRunning reports whether the monitor is currently active.
func (g *AppGUI) monitorRunning() bool {
	g.monitorMu.Lock()
	defer g.monitorMu.Unlock()
	return g.monitorCancel != nil
}

// applyMonitorSettings pushes the settings changed on the settings screen
// into the running monitor (hot reload: an interval change resets the
// running timer). No-op when the monitor is not running.
func (g *AppGUI) applyMonitorSettings() {
	g.monitorMu.Lock()
	monitor := g.proxyMonitor
	g.monitorMu.Unlock()
	if monitor == nil {
		return
	}
	applyConfigToMonitor(monitor, g.cfg)
}

// applyConfigToMonitor syncs the persisted monitor settings into the worker.
func applyConfigToMonitor(monitor *services.ProxyMonitor, cfg *config.Config) {
	monitor.SetMode(monitorServiceMode(cfg.MonitorMode))
	monitor.SetRequireCountry(cfg.MonitorRequireCountry)
	monitor.SetInterval(monitorInterval(cfg.MonitorIntervalMin))
}

// monitorServiceMode converts the persisted monitor mode into the worker's
// switch mode. Unknown values fall back to MonitorSwitchAny.
func monitorServiceMode(mode string) services.MonitorSwitchMode {
	switch mode {
	case monitorModeSameCountry:
		return services.MonitorSwitchSameCountry
	case monitorModeNone:
		return services.MonitorSwitchNone
	default:
		return services.MonitorSwitchAny
	}
}

// monitorInterval converts the configured minutes into the tick period,
// falling back to the worker default when unset or invalid.
func monitorInterval(minutes int) time.Duration {
	if minutes <= 0 {
		return services.DefaultMonitorInterval
	}
	return time.Duration(minutes) * time.Minute
}

// updateMonitorPool pushes a snapshot of the result table into the running
// monitor. Callable from any goroutine: the snapshot is taken inside fyne.Do
// (safe both before the event loop starts and after it drains), and
// ProxyMonitor.UpdatePool is internally synchronized.
func (g *AppGUI) updateMonitorPool() {
	fyne.Do(func() {
		items := g.monitorPoolSnapshot()
		fresh := g.monitorPoolFresh()
		g.monitorMu.Lock()
		monitor := g.proxyMonitor
		g.monitorMu.Unlock()
		if monitor != nil {
			monitor.UpdatePool(items, fresh)
		}
	})
}

// monitorPoolSnapshot converts the result-table wrappers into domain items
// for the monitor's candidate pool, sorted by the table's RTT metric (TCPms,
// TODO item 1: "критерий выбора кандидата — по времени отклика"). Country-less
// and previously failed proxies are excluded from switching. Must run on the
// UI thread.
func (g *AppGUI) monitorPoolSnapshot() []*services.ProxyItem {
	items := make([]*services.ProxyItem, 0, len(g.proxyItems))
	for _, wrapper := range g.proxyItems {
		if wrapper.IsUnavailable {
			continue
		}
		items = append(items, &services.ProxyItem{
			Host:    wrapper.Host,
			Port:    wrapper.Port,
			Country: wrapper.Country,
			Type:    wrapper.Type,
			Source:  common.Source(wrapper.Source),
			RTTms:   int(wrapper.TCPMs),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].RTTms < items[j].RTTms
	})
	return items
}

// markProxyUnavailable flags the result-table row for the failed proxy so it
// renders with the "unavailable" background and a disabled Apply button.
// Must be called on the UI thread (wrap in fyne.DoAndWait from goroutines).
func (g *AppGUI) markProxyUnavailable(host, port string) {
	for _, wrapper := range g.proxyItems {
		if wrapper.Host == host && wrapper.Port == port {
			wrapper.IsUnavailable = true
		}
	}
	g.table.Refresh()
}

// monitorPoolFresh reports whether every source represented in the table
// still has an unexpired results cache (TODO item 1: "Протухший кэш (TTL
// истёк) не используется как пул кандидатов"). Must run on the UI thread.
func (g *AppGUI) monitorPoolFresh() bool {
	if len(g.proxyItems) == 0 {
		return false
	}

	now := time.Now()
	g.cacheExpiryMutex.Lock()
	defer g.cacheExpiryMutex.Unlock()

	seen := make(map[common.Source]bool, len(g.cacheExpiry))
	for _, wrapper := range g.proxyItems {
		source := common.Source(wrapper.Source)
		if seen[source] {
			continue
		}
		seen[source] = true
		expireAt, known := g.cacheExpiry[source]
		if !known || !expireAt.After(now) {
			return false
		}
	}
	return true
}
