package services

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultMonitorInterval is the tick period used when no explicit interval
// is set (TODO item 1: "проверка раз в минуту").
const DefaultMonitorInterval = time.Minute

// TopCandidates caps how many best-latency candidates are batch-checked
// before switching (TODO item 1: "берём N лучших из пула, ориентир 5").
const TopCandidates = 5

// MonitorSwitchMode defines the behavior when the active proxy is down
// (TODO item 1: "поведение при недоступности текущего прокси").
type MonitorSwitchMode int

const (
	// MonitorSwitchAny switches to any available proxy from the pool.
	MonitorSwitchAny MonitorSwitchMode = iota
	// MonitorSwitchSameCountry switches to a proxy from the same country
	// as the failed one.
	MonitorSwitchSameCountry
	// MonitorSwitchNone never switches; only notifies about unavailability.
	MonitorSwitchNone
)

// ActiveProxyStore is the port to the currently applied system proxy.
// The adapter (GUI) is responsible for filling the full ProxyItem,
// including Country, from the result table.
type ActiveProxyStore interface {
	GetActiveProxy(ctx context.Context) (ProxyItem, bool)
	ApplyProxy(ctx context.Context, item ProxyItem) error
}

// MonitorNotifier reports monitoring outcomes to the user
// (log lines, GUI banners, system notifications — adapter's choice).
type MonitorNotifier interface {
	// Healthy reports a successful periodic health-check: the monitor's
	// heartbeat proving the worker is alive and watching the proxy.
	Healthy(item ProxyItem)
	ProxyDown(item ProxyItem)
	SwitchedTo(item ProxyItem)
	SwitchFailed(reason string)
}

// ProxyMonitor periodically health-checks the active proxy and, on failure,
// switches to the best available candidate from the fresh result pool.
type ProxyMonitor struct {
	checker  ProxyChecker
	store    ActiveProxyStore
	notifier MonitorNotifier
	params   *CheckParams

	// settingsMu guards the user-configurable settings and the pool;
	// it is never held during network calls.
	settingsMu     sync.Mutex
	interval       time.Duration
	mode           MonitorSwitchMode
	requireCountry bool
	pool           []*ProxyItem

	// intervalChanged coalesces SetInterval signals for the running loop
	// (buffered size 1; a non-blocking send drops the signal if one is
	// already pending — the loop always re-reads the current interval).
	intervalChanged chan struct{}

	// lifecycleMu guards start/stop; wg waits for the loop goroutine.
	lifecycleMu sync.Mutex
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewProxyMonitor builds a monitor. Defaults: interval 1m, mode Any,
// requireCountry false, empty pool (no candidates until UpdatePool is called).
func NewProxyMonitor(
	checker ProxyChecker,
	store ActiveProxyStore,
	notifier MonitorNotifier,
	params *CheckParams,
) *ProxyMonitor {
	return &ProxyMonitor{
		checker:         checker,
		store:           store,
		notifier:        notifier,
		params:          params,
		interval:        DefaultMonitorInterval,
		mode:            MonitorSwitchAny,
		intervalChanged: make(chan struct{}, 1),
	}
}

// SetInterval sets the tick period; a running loop resets its timer and
// picks up the new value without a restart.
func (m *ProxyMonitor) SetInterval(interval time.Duration) {
	m.settingsMu.Lock()
	m.interval = interval
	m.settingsMu.Unlock()
	select {
	case m.intervalChanged <- struct{}{}:
	default:
	}
}

// SetMode sets the switch mode; read at the start of every tick.
func (m *ProxyMonitor) SetMode(mode MonitorSwitchMode) {
	m.settingsMu.Lock()
	defer m.settingsMu.Unlock()
	m.mode = mode
}

// SetRequireCountry toggles skipping country-less candidates.
// Only meaningful in MonitorSwitchAny mode (TODO item 1).
func (m *ProxyMonitor) SetRequireCountry(require bool) {
	m.settingsMu.Lock()
	defer m.settingsMu.Unlock()
	m.requireCountry = require
}

// UpdatePool replaces the candidate pool. A stale snapshot (fresh=false)
// must not be used for switching (TODO item 1), so the pool is cleared.
// The slice is copied to detach from the caller's backing array.
func (m *ProxyMonitor) UpdatePool(pool []*ProxyItem, fresh bool) {
	m.settingsMu.Lock()
	defer m.settingsMu.Unlock()
	if !fresh {
		m.pool = nil
		return
	}
	snapshot := make([]*ProxyItem, len(pool))
	copy(snapshot, pool)
	m.pool = snapshot
}

// Start launches the monitor loop in a background goroutine. Idempotent:
// starting an already-running monitor does nothing.
func (m *ProxyMonitor) Start(parent context.Context) {
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()
	if m.cancel != nil {
		return
	}
	runCtx, cancel := context.WithCancel(parent)
	m.cancel = cancel
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.loop(runCtx)
	}()
}

// Stop cancels the loop and blocks until the current tick finishes.
// Safe to call multiple times and from any goroutine.
func (m *ProxyMonitor) Stop() {
	m.lifecycleMu.Lock()
	cancel := m.cancel
	m.cancel = nil
	m.lifecycleMu.Unlock()
	if cancel != nil {
		cancel()
	}
	m.wg.Wait()
}

// loop runs the periodic health-check until ctx is canceled. The interval
// is re-read on every timer reset, so SetInterval applies without a restart.
func (m *ProxyMonitor) loop(ctx context.Context) {
	timer := time.NewTimer(m.currentInterval())
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			m.tick(ctx)
			timer.Reset(m.currentInterval())
		case <-m.intervalChanged:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(m.currentInterval())
		}
	}
}

// currentInterval returns the configured tick period under the settings lock.
func (m *ProxyMonitor) currentInterval() time.Duration {
	m.settingsMu.Lock()
	defer m.settingsMu.Unlock()
	return m.interval
}

// tick performs one health-check cycle: probe the active proxy and,
// on failure, notify and try to switch according to the current mode.
func (m *ProxyMonitor) tick(ctx context.Context) {
	if ctx.Err() != nil {
		return // shutdown in progress — nothing to monitor
	}

	m.settingsMu.Lock()
	mode := m.mode
	pool := m.pool
	m.settingsMu.Unlock()

	active, found := m.store.GetActiveProxy(ctx)
	if !found {
		return // nothing applied — nothing to monitor
	}

	failedAddr := fmt.Sprintf("%s:%s", active.Host, active.Port)
	checkCtx, cancel := context.WithTimeout(ctx, m.params.Timeout)
	result := m.checker.CheckProxy(checkCtx, failedAddr, m.params)
	cancel()
	if result.Error == nil {
		m.notifier.Healthy(active) // heartbeat: one record per interval
		return
	}
	if ctx.Err() != nil {
		return // shutdown in progress — do not switch mid-teardown
	}

	m.notifier.ProxyDown(active)

	if mode == MonitorSwitchNone {
		return // notify only (TODO item 1, mode 3)
	}
	m.switchFromPool(ctx, active, pool, mode)
}

// switchFromPool batch-checks up to TopCandidates candidates and applies
// the first passing one (the pool arrives sorted by the latency metric).
func (m *ProxyMonitor) switchFromPool(
	ctx context.Context,
	failed ProxyItem,
	pool []*ProxyItem,
	mode MonitorSwitchMode,
) {
	candidates := m.selectCandidates(failed, pool, mode)
	if len(candidates) == 0 {
		m.notifier.SwitchFailed("candidate pool is empty")
		return
	}

	results := m.checker.CheckBatch(ctx, candidates, m.params, nil, nil)
	if len(results) == 0 {
		m.notifier.SwitchFailed("no candidate passed the batch check")
		return
	}

	winner := results[0].ProxyItem
	if err := m.store.ApplyProxy(ctx, winner); err != nil {
		m.notifier.SwitchFailed(fmt.Sprintf("apply %s:%s failed: %v", winner.Host, winner.Port, err))
		return
	}
	m.notifier.SwitchedTo(winner)
}

// selectCandidates picks up to TopCandidates proxies from the pool:
// the failed proxy is excluded; SameCountry keeps only matching country;
// Any may skip country-less proxies when requireCountry is set.
func (m *ProxyMonitor) selectCandidates(
	failed ProxyItem,
	pool []*ProxyItem,
	mode MonitorSwitchMode,
) []*ProxyItem {
	m.settingsMu.Lock()
	requireCountry := m.requireCountry
	m.settingsMu.Unlock()

	candidates := make([]*ProxyItem, 0, TopCandidates)
	for _, item := range pool {
		if item == nil || (item.Host == failed.Host && item.Port == failed.Port) {
			continue
		}
		switch {
		case mode == MonitorSwitchSameCountry && item.Country != failed.Country:
			continue
		case mode == MonitorSwitchAny && requireCountry && item.Country == "":
			continue
		}
		candidates = append(candidates, item)
		if len(candidates) == TopCandidates {
			break
		}
	}
	return candidates
}
