package gui

import (
	"context"
	"fmt"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/services"
	"strings"

	"fyne.io/fyne/v2"
)

// guiActiveProxyStore adapts AppGUI to the services.ActiveProxyStore port.
// It runs on the monitor goroutine, so every AppGUI state read or mutation
// is synchronized with the UI thread via fyne.DoAndWait (AGENTS.md rule).
type guiActiveProxyStore struct {
	gui *AppGUI
}

// newActiveProxyStoreAdapter wraps the GUI as a services.ActiveProxyStore.
func newActiveProxyStoreAdapter(gui *AppGUI) services.ActiveProxyStore {
	return &guiActiveProxyStore{gui: gui}
}

// GetActiveProxy returns the proxy currently applied at the OS level.
// Country and type are looked up in the result table (the table is the only
// place the full ProxyItem, including Country, is known to the GUI).
func (s *guiActiveProxyStore) GetActiveProxy(_ context.Context) (services.ProxyItem, bool) {
	var (
		item  services.ProxyItem
		found bool
	)
	fyne.DoAndWait(func() {
		gui := s.gui
		if gui.activeProxyHost == "" || gui.activeProxyPort == "" {
			return
		}
		found = true
		item = services.ProxyItem{Host: gui.activeProxyHost, Port: gui.activeProxyPort}
		for _, wrapper := range gui.proxyItems {
			if wrapper.Host == item.Host && wrapper.Port == item.Port {
				item.Country = wrapper.Country
				item.Type = wrapper.Type
				break
			}
		}
	})
	return item, found
}

// ApplyProxy applies the given proxy through the same code path as the
// table Apply button (system settings write + UI state update + highlight).
func (s *guiActiveProxyStore) ApplyProxy(_ context.Context, item services.ProxyItem) error {
	var applyErr error
	fyne.DoAndWait(func() {
		applyErr = s.gui.applySystemProxy(item.Host, item.Port, string(item.Type))
	})
	return applyErr
}

// guiMonitorNotifier adapts AppGUI logging and desktop notifications to the
// services.MonitorNotifier port. Safe to call from any goroutine:
// appendLog self-synchronizes (logMutex + fyne.Do), and
// fyne.CurrentApp().SendNotification is documented as goroutine-safe.
type guiMonitorNotifier struct {
	gui *AppGUI
}

// newMonitorNotifierAdapter wraps the GUI as a services.MonitorNotifier.
func newMonitorNotifierAdapter(gui *AppGUI) services.MonitorNotifier {
	return &guiMonitorNotifier{gui: gui}
}

// ProxyDown reports the unavailability of the monitored proxy.
func (n *guiMonitorNotifier) ProxyDown(item services.ProxyItem) {
	n.gui.appendLog(common.LogLevelError, fmt.Sprintf(
		"%s %s:%s", i18n.T("gui.log_monitor_down"), item.Host, item.Port))
	n.gui.sendSystemNotification(
		i18n.T("gui.notify_monitor_down_title"),
		fmt.Sprintf("%s:%s", item.Host, item.Port),
	)
	fyne.DoAndWait(func() {
		n.gui.markProxyUnavailable(item.Host, item.Port)
	})
}

// SwitchedTo reports a successful auto-switch to a new proxy.
func (n *guiMonitorNotifier) SwitchedTo(item services.ProxyItem) {
	n.gui.appendLog(common.LogLevelInfo, fmt.Sprintf(
		"%s %s://%s:%s",
		i18n.T("gui.log_monitor_switched"),
		strings.ToLower(string(item.Type)),
		item.Host,
		item.Port,
	))
	n.gui.sendSystemNotification(
		i18n.T("gui.notify_monitor_switched_title"),
		fmt.Sprintf("%s:%s", item.Host, item.Port),
	)
}

// SwitchFailed reports that no proxy could be applied.
func (n *guiMonitorNotifier) SwitchFailed(reason string) {
	n.gui.appendLog(common.LogLevelError, fmt.Sprintf(
		"%s: %s", i18n.T("gui.log_monitor_switch_failed"), reason))
}
