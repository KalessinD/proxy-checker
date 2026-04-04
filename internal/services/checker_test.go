package services_test

import (
	"context"
	"errors"
	"proxy-checker/internal/services"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockVerifier struct {
	mu           sync.Mutex
	responses    map[string]services.Result
	callCount    int32
	blockChannel chan struct{}
}

func (m *mockVerifier) Verify(ctx context.Context, proxyAddr, _, _ string, _ bool) services.Result {
	atomic.AddInt32(&m.callCount, 1)

	if m.blockChannel != nil {
		select {
		case <-ctx.Done():
			return services.Result{Error: ctx.Err()}
		case <-m.blockChannel:
			return services.Result{Error: errors.New("blocked by test")}
		}
	}

	if ctx.Err() != nil {
		return services.Result{Error: ctx.Err()}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if res, exists := m.responses[proxyAddr]; exists {
		return res
	}

	return services.Result{Error: errors.New("mock: unexpected proxy address")}
}

func TestCheckBatch_FiltersInvalidProxies(t *testing.T) {
	mock := &mockVerifier{
		responses: map[string]services.Result{
			"1.1.1.1:8080": {ProxyLatency: 10 * time.Millisecond, ReqLatency: 50 * time.Millisecond, StatusCode: 200, SupportsHTTP: true},
			"2.2.2.2:8080": {Error: errors.New("connection timed out")},
			"3.3.3.3:8080": {ProxyLatency: 20 * time.Millisecond, ReqLatency: 30 * time.Millisecond, StatusCode: 200, SupportsHTTP: true},
		},
	}

	items := []*services.ProxyItem{
		{Host: "1.1.1.1", Port: "8080", Type: "socks5"},
		{Host: "2.2.2.2", Port: "8080", Type: "socks5"},
		{Host: "3.3.3.3", Port: "8080", Type: "socks5"},
	}

	checker := services.NewProxyChecker()
	ctx := t.Context()
	validProxies := checker.CheckBatch(ctx, items, "google.com", "socks5", 5*time.Second, 2, false, nil, mock)

	require.Len(t, validProxies, 2, "Only valid proxies must remain")
	assert.Equal(t, "3.3.3.3", validProxies[0].Host, "The fastest proxy must be first")
	assert.Equal(t, "1.1.1.1", validProxies[1].Host)
}

func TestCheckBatch_StopsOnContextCancellation(t *testing.T) {
	mock := &mockVerifier{
		blockChannel: make(chan struct{}),
		responses: map[string]services.Result{
			"1.1.1.1:8080": {StatusCode: 200},
		},
	}

	items := make([]*services.ProxyItem, 10)
	for i := 0; i < 10; i++ {
		items[i] = &services.ProxyItem{Host: "1.1.1.1", Port: "8080", Type: "socks5"}
	}

	ctx, cancel := context.WithCancel(t.Context())
	checker := services.NewProxyChecker()

	done := make(chan []*services.ProxyItemFull)
	go func() {
		done <- checker.CheckBatch(ctx, items, "google.com", "socks5", 10*time.Second, 2, false, nil, mock)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		called := atomic.LoadInt32(&mock.callCount)
		assert.LessOrEqual(t, called, int32(2), "No extra workers should start on cancel")
	case <-time.After(2 * time.Second):
		t.Fatal("CheckBatch did not exit after context cancellation (potential deadlock)")
	}
}

func TestCheckBatch_EmptyList(t *testing.T) {
	mock := &mockVerifier{}
	checker := services.NewProxyChecker()
	ctx := t.Context()
	validProxies := checker.CheckBatch(ctx, []*services.ProxyItem{}, "google.com", "socks5", 5*time.Second, 2, false, nil, mock)

	assert.Empty(t, validProxies, "Empty list must return an empty slice")
}

func TestResolveSchema_DefaultBehavior(t *testing.T) {
	tests := []struct {
		name           string
		proxyMode      string
		forceHTTP2     bool
		expectedSchema string
	}{
		{name: "HTTP mode without force", proxyMode: "http", forceHTTP2: false, expectedSchema: "http://"},
		{name: "HTTPS mode without force", proxyMode: "https", forceHTTP2: false, expectedSchema: "https://"},
		{name: "SOCKS4 mode without force", proxyMode: "socks4", forceHTTP2: false, expectedSchema: "http://"},
		{name: "SOCKS5 mode without force", proxyMode: "socks5", forceHTTP2: false, expectedSchema: "https://"},
		{name: "Unknown mode fallback without force", proxyMode: "unknown", forceHTTP2: false, expectedSchema: "http://"},
		{name: "HTTP mode WITH force HTTP2", proxyMode: "http", forceHTTP2: true, expectedSchema: "https://"},
		{name: "SOCKS4 mode WITH force HTTP2", proxyMode: "socks4", forceHTTP2: true, expectedSchema: "https://"},
	}

	checker := services.NewProxyChecker()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualSchema := checker.ResolveSchema(tt.proxyMode, tt.forceHTTP2)
			assert.Equal(t, tt.expectedSchema, actualSchema)
		})
	}
}

func TestProxyChecker_CheckProxy_Timeout(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	checker := services.NewProxyChecker()
	res := checker.CheckProxy(ctx, "127.0.0.1:9999", "google.com", "http", false)

	assert.Error(t, res.Error)
}

func TestNewDefaultVerifier_Verify_DelegatesToChecker(t *testing.T) {
	verifier := services.NewDefaultVerifier()
	require.NotNil(t, verifier)

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately to fail fast without real network I/O

	res := verifier.Verify(ctx, "127.0.0.1:9999", "google.com", "http", false)
	assert.Error(t, res.Error, "Canceled context must cause an error")
}
