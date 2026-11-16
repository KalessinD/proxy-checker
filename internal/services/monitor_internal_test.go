package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeStore is a minimal ActiveProxyStore fake for monitor tests.
type fakeStore struct {
	mu       sync.Mutex
	active   ProxyItem
	found    bool
	applied  []ProxyItem
	applyErr error
}

func (s *fakeStore) GetActiveProxy(_ context.Context) (ProxyItem, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active, s.found
}

func (s *fakeStore) ApplyProxy(_ context.Context, item ProxyItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.applyErr != nil {
		return s.applyErr
	}
	s.active = item
	s.found = true
	s.applied = append(s.applied, item)
	return nil
}

// fakeNotifier records monitoring outcomes.
type fakeNotifier struct {
	mu       sync.Mutex
	healthy  []ProxyItem
	down     []ProxyItem
	switched []ProxyItem
	failed   []string
}

func (n *fakeNotifier) Healthy(item ProxyItem) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.healthy = append(n.healthy, item)
}

func (n *fakeNotifier) ProxyDown(item ProxyItem) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.down = append(n.down, item)
}

func (n *fakeNotifier) SwitchedTo(item ProxyItem) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.switched = append(n.switched, item)
}

func (n *fakeNotifier) SwitchFailed(reason string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.failed = append(n.failed, reason)
}

// failingThenHealthyChecker fails the given address once, passes everything else.
type failingChecker struct {
	failAddr string
	batch    []*ProxyItemFull
}

func (c *failingChecker) CheckProxy(_ context.Context, proxyAddr string, _ *CheckParams) Result {
	result := Result{}
	if proxyAddr == c.failAddr {
		result.Error = errors.New("connection refused")
	}
	return result
}

func (c *failingChecker) CheckBatch(
	_ context.Context,
	proxiesList []*ProxyItem,
	_ *CheckParams,
	_ ProxyVerifier,
	_ func(current, total int),
) []*ProxyItemFull {
	if c.batch != nil {
		return c.batch
	}
	results := make([]*ProxyItemFull, 0, len(proxiesList))
	for _, item := range proxiesList {
		results = append(results, &ProxyItemFull{
			ProxyItem:   *item,
			CheckResult: Result{},
		})
	}
	return results
}

func newMonitorFixture(
	t *testing.T,
	checker ProxyChecker,
) (*ProxyMonitor, *fakeStore, *fakeNotifier) {
	t.Helper()
	store := &fakeStore{}
	notifier := &fakeNotifier{}
	monitor := NewProxyMonitor(checker, store, notifier, &CheckParams{
		Timeout: 5 * time.Second,
		Workers: 4,
	})
	return monitor, store, notifier
}

func monitorPool(items ...ProxyItem) []*ProxyItem {
	pool := make([]*ProxyItem, len(items))
	for i := range items {
		pool[i] = &items[i]
	}
	return pool
}

func TestMonitorTickHealthyProxyNotifiesHeartbeat(t *testing.T) {
	// failAddr differs from the active proxy, so the health check passes.
	checker := &failingChecker{failAddr: "0.0.0.0:0"}
	monitor, store, notifier := newMonitorFixture(t, checker)
	active := ProxyItem{Host: "10.0.0.99", Port: "1080", Country: "NL"}
	store.active = active
	store.found = true
	monitor.UpdatePool(monitorPool(ProxyItem{Host: "10.0.0.1", Port: "1080"}), true)

	monitor.tick(t.Context())

	require.Len(t, notifier.healthy, 1)
	assert.Equal(t, active.Host, notifier.healthy[0].Host)
	assert.Empty(t, notifier.down)
	assert.Empty(t, notifier.switched)
	assert.Empty(t, store.applied)
}

func TestMonitorTickSwitchesToFirstPassingCandidate(t *testing.T) {
	checker := &failingChecker{failAddr: "10.0.0.99:1080"}
	monitor, store, notifier := newMonitorFixture(t, checker)
	store.active = ProxyItem{Host: "10.0.0.99", Port: "1080", Country: "NL"}
	store.found = true
	monitor.UpdatePool(monitorPool(
		ProxyItem{Host: "10.0.0.1", Port: "1080", Country: "NL"},
		ProxyItem{Host: "10.0.0.2", Port: "1080", Country: "DE"},
	), true)

	monitor.tick(t.Context())

	require.Len(t, notifier.down, 1)
	require.Len(t, notifier.switched, 1)
	assert.Equal(t, "10.0.0.1", notifier.switched[0].Host)
	assert.Equal(t, "10.0.0.1", store.active.Host)
}

func TestMonitorTickExcludesFailedProxyFromCandidates(t *testing.T) {
	checker := &failingChecker{
		failAddr: "10.0.0.99:1080",
		// The batch returns the failed proxy itself as a "winner" —
		// the monitor must never have passed it to CheckBatch.
	}
	monitor, store, notifier := newMonitorFixture(t, checker)
	store.active = ProxyItem{Host: "10.0.0.99", Port: "1080"}
	store.found = true
	monitor.UpdatePool(monitorPool(
		ProxyItem{Host: "10.0.0.99", Port: "1080"},
		ProxyItem{Host: "10.0.0.1", Port: "1080"},
	), true)

	monitor.tick(t.Context())

	require.Len(t, notifier.switched, 1)
	assert.Equal(t, "10.0.0.1", store.active.Host)
}

func TestMonitorTickNoCandidatesNotifiesSwitchFailed(t *testing.T) {
	checker := &failingChecker{failAddr: "10.0.0.99:1080"}
	monitor, store, notifier := newMonitorFixture(t, checker)
	store.active = ProxyItem{Host: "10.0.0.99", Port: "1080"}
	store.found = true
	monitor.UpdatePool(nil, true) // fresh but empty pool

	monitor.tick(t.Context())

	require.Len(t, notifier.failed, 1)
	assert.Contains(t, notifier.failed[0], "candidate pool is empty")
}

func TestMonitorTickStalePoolIsNotUsed(t *testing.T) {
	checker := &failingChecker{failAddr: "10.0.0.99:1080"}
	monitor, store, notifier := newMonitorFixture(t, checker)
	store.active = ProxyItem{Host: "10.0.0.99", Port: "1080"}
	store.found = true
	monitor.UpdatePool(monitorPool(ProxyItem{Host: "10.0.0.1", Port: "1080"}), false)

	monitor.tick(t.Context())

	require.Len(t, notifier.down, 1)
	require.Len(t, notifier.failed, 1)
	assert.Contains(t, notifier.failed[0], "candidate pool is empty")
}

func TestMonitorTickAllCandidatesFailedNotifiesWithoutSwitch(t *testing.T) {
	checker := &failingChecker{failAddr: "10.0.0.99:1080", batch: []*ProxyItemFull{}}
	monitor, store, notifier := newMonitorFixture(t, checker)
	store.active = ProxyItem{Host: "10.0.0.99", Port: "1080"}
	store.found = true
	monitor.UpdatePool(monitorPool(ProxyItem{Host: "10.0.0.1", Port: "1080"}), true)

	monitor.tick(t.Context())

	require.Len(t, notifier.failed, 1)
	assert.Contains(t, notifier.failed[0], "no candidate passed")
	assert.Empty(t, store.applied)
}

func TestMonitorTickModeNoneOnlyNotifies(t *testing.T) {
	checker := &failingChecker{failAddr: "10.0.0.99:1080"}
	monitor, store, notifier := newMonitorFixture(t, checker)
	monitor.SetMode(MonitorSwitchNone)
	store.active = ProxyItem{Host: "10.0.0.99", Port: "1080"}
	store.found = true
	monitor.UpdatePool(monitorPool(ProxyItem{Host: "10.0.0.1", Port: "1080"}), true)

	monitor.tick(t.Context())

	require.Len(t, notifier.down, 1)
	assert.Empty(t, notifier.failed)
	assert.Empty(t, notifier.switched)
	assert.Empty(t, store.applied)
}

func TestMonitorTickSameCountryModeFiltersCandidates(t *testing.T) {
	checker := &failingChecker{failAddr: "10.0.0.99:1080"}
	monitor, store, notifier := newMonitorFixture(t, checker)
	monitor.SetMode(MonitorSwitchSameCountry)
	store.active = ProxyItem{Host: "10.0.0.99", Port: "1080", Country: "NL"}
	store.found = true
	monitor.UpdatePool(monitorPool(
		ProxyItem{Host: "10.0.0.1", Port: "1080", Country: "DE"},
		ProxyItem{Host: "10.0.0.2", Port: "1080", Country: "NL"},
	), true)

	monitor.tick(t.Context())

	require.Len(t, notifier.switched, 1)
	assert.Equal(t, "10.0.0.2", store.active.Host)
}

func TestMonitorTickRequireCountrySkipsCountrylessCandidates(t *testing.T) {
	checker := &failingChecker{failAddr: "10.0.0.99:1080"}
	monitor, store, notifier := newMonitorFixture(t, checker)
	monitor.SetMode(MonitorSwitchAny)
	monitor.SetRequireCountry(true)
	store.active = ProxyItem{Host: "10.0.0.99", Port: "1080"}
	store.found = true
	monitor.UpdatePool(monitorPool(
		ProxyItem{Host: "10.0.0.1", Port: "1080", Country: ""},
		ProxyItem{Host: "10.0.0.2", Port: "1080", Country: "DE"},
	), true)

	monitor.tick(t.Context())

	require.Len(t, notifier.switched, 1)
	assert.Equal(t, "10.0.0.2", store.active.Host)
}

func TestMonitorTickNoActiveProxyIsANoOp(t *testing.T) {
	checker := &failingChecker{failAddr: "10.0.0.99:1080"}
	monitor, _, notifier := newMonitorFixture(t, checker)

	monitor.tick(t.Context())

	assert.Empty(t, notifier.down)
}

func TestMonitorTickApplyErrorNotifiesSwitchFailed(t *testing.T) {
	checker := &failingChecker{failAddr: "10.0.0.99:1080"}
	monitor, store, notifier := newMonitorFixture(t, checker)
	store.applyErr = errors.New("system proxy manager unavailable")
	store.active = ProxyItem{Host: "10.0.0.99", Port: "1080"}
	store.found = true
	monitor.UpdatePool(monitorPool(ProxyItem{Host: "10.0.0.1", Port: "1080"}), true)

	monitor.tick(t.Context())

	require.Len(t, notifier.failed, 1)
	assert.Contains(t, notifier.failed[0], "apply 10.0.0.1:1080 failed")
}

func TestMonitorTickSelectsAtMostTopCandidates(t *testing.T) {
	var batchInput []*ProxyItem
	checker := &batchCapturingChecker{capture: &batchInput, failAddr: "10.0.0.99:1080"}
	monitor, store, _ := newMonitorFixture(t, checker)
	store.active = ProxyItem{Host: "10.0.0.99", Port: "1080"}
	store.found = true

	pool := make([]*ProxyItem, TopCandidates+3)
	for i := range pool {
		pool[i] = &ProxyItem{Host: fmt.Sprintf("10.0.0.%d", i+1), Port: "1080"}
	}
	monitor.UpdatePool(pool, true)

	monitor.tick(t.Context())

	require.Len(t, batchInput, TopCandidates)
}

func TestMonitorStartStopIsIdempotent(t *testing.T) {
	monitor, _, _ := newMonitorFixture(t, &failingChecker{})
	monitor.SetInterval(10 * time.Millisecond)

	ctx, cancel := context.WithCancel(t.Context())
	monitor.Start(ctx)
	monitor.Start(ctx) // second start must be a no-op
	cancel()
	monitor.Stop() // waits for the loop goroutine
	monitor.Stop() // second stop must be a no-op
}

func TestMonitorLoopPicksUpIntervalChangeLive(t *testing.T) {
	checker := &failingChecker{failAddr: "10.0.0.99:1080"}
	monitor, store, notifier := newMonitorFixture(t, checker)
	monitor.SetInterval(10 * time.Second) // effectively paused until changed
	store.active = ProxyItem{Host: "10.0.0.99", Port: "1080"}
	store.found = true
	monitor.UpdatePool(monitorPool(ProxyItem{Host: "10.0.0.1", Port: "1080"}), true)

	ctx, cancel := context.WithCancel(t.Context())
	monitor.Start(ctx)

	// No tick must fire while the interval is still 10s.
	time.Sleep(50 * time.Millisecond)
	notifier.mu.Lock()
	downCount := len(notifier.down)
	notifier.mu.Unlock()
	require.Zero(t, downCount)

	// Shrinking the interval live must start ticking without a restart.
	monitor.SetInterval(10 * time.Millisecond)
	waitUntil(t, 2*time.Second, func() bool {
		notifier.mu.Lock()
		defer notifier.mu.Unlock()
		return len(notifier.down) > 0
	})

	cancel()
	monitor.Stop()
}

func TestMonitorLoopRunsTicksAndStops(t *testing.T) {
	monitor, store, notifier := newMonitorFixture(t, &failingChecker{failAddr: "10.0.0.99:1080"})
	monitor.SetInterval(10 * time.Millisecond)
	store.active = ProxyItem{Host: "10.0.0.99", Port: "1080"}
	store.found = true
	monitor.UpdatePool(monitorPool(ProxyItem{Host: "10.0.0.1", Port: "1080"}), true)

	ctx, cancel := context.WithCancel(t.Context())
	monitor.Start(ctx)

	waitUntil(t, 2*time.Second, func() bool {
		notifier.mu.Lock()
		defer notifier.mu.Unlock()
		return len(notifier.down) > 0
	})

	cancel()
	monitor.Stop()
}

// waitUntil polls the predicate until it passes or the timeout expires.
func waitUntil(t *testing.T, timeout time.Duration, predicate func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

// batchCapturingChecker records CheckBatch input for assertions.
type batchCapturingChecker struct {
	capture  *[]*ProxyItem
	failAddr string
}

func (c *batchCapturingChecker) CheckProxy(_ context.Context, proxyAddr string, _ *CheckParams) Result {
	result := Result{}
	if proxyAddr == c.failAddr {
		result.Error = errors.New("connection refused")
	}
	return result
}

func (c *batchCapturingChecker) CheckBatch(
	_ context.Context,
	proxiesList []*ProxyItem,
	_ *CheckParams,
	_ ProxyVerifier,
	_ func(current, total int),
) []*ProxyItemFull {
	*c.capture = proxiesList
	results := make([]*ProxyItemFull, 0, len(proxiesList))
	for _, item := range proxiesList {
		results = append(results, &ProxyItemFull{ProxyItem: *item, CheckResult: Result{}})
	}
	return results
}
