package gui

import (
	"context"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/services"
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
// for the monitor's candidate pool. Must run on the UI thread.
func (g *AppGUI) monitorPoolSnapshot() []*services.ProxyItem {
	items := make([]*services.ProxyItem, 0, len(g.proxyItems))
	for _, wrapper := range g.proxyItems {
		items = append(items, &services.ProxyItem{
			Host:    wrapper.Host,
			Port:    wrapper.Port,
			Country: wrapper.Country,
			Type:    wrapper.Type,
			Source:  common.Source(wrapper.Source),
		})
	}
	return items
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
