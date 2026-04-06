package services_test

import (
	"context"
	"errors"
	"os"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/config"
	"proxy-checker/internal/fetcher"
	"proxy-checker/internal/services"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	_ = i18n.Init("en")
	os.Exit(m.Run())
}

type mockPipelineFetcher struct {
	mu        sync.Mutex
	responses map[string][]*fetcher.ProxyItem
	err       error
	callCount int32
	blockChan chan struct{}
}

func (m *mockPipelineFetcher) Fetch(ctx context.Context, settings fetcher.Settings) ([]*fetcher.ProxyItem, error) {
	if m.blockChan != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-m.blockChan:
			return nil, errors.New("blocked by test")
		}
	}

	atomic.AddInt32(&m.callCount, 1)

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.err != nil {
		return nil, m.err
	}

	key := string(settings.Type)
	if items, ok := m.responses[key]; ok {
		return items, nil
	}
	return []*fetcher.ProxyItem{}, nil
}

type mockPipelineVerifier struct {
	responses map[string]services.Result
}

func (m *mockPipelineVerifier) Verify(ctx context.Context, proxyAddr, _, _ string, _ bool) services.Result {
	if ctx.Err() != nil {
		return services.Result{Error: ctx.Err()}
	}
	if res, ok := m.responses[proxyAddr]; ok {
		return res
	}
	return services.Result{Error: errors.New("mock: unexpected address")}
}

func basePipelineConfig() *config.Config {
	return &config.Config{
		Type:       common.ProxySOCKS5,
		Sources:    []common.Source{common.SourceProxyMania},
		Timeout:    5 * time.Second,
		Workers:    2,
		DestAddr:   "google.com",
		CheckHTTP2: false,
		Lang:       "en",
	}
}

func TestRunPipeline_FetcherReturnsError(t *testing.T) {
	mockFetcher := &mockPipelineFetcher{
		err: errors.New("network error"),
	}
	mockVerifier := &mockPipelineVerifier{}
	fetchers := []services.SourceFetcher{
		{Source: common.SourceProxyMania, Fetcher: mockFetcher},
	}

	ctx := t.Context()
	cfg := basePipelineConfig()

	validProxies, err := services.RunPipeline(ctx, fetchers, mockVerifier, cfg, nil, services.PipelineCallbacks{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline fetch error")
	assert.Nil(t, validProxies)
	assert.Equal(t, int32(1), atomic.LoadInt32(&mockFetcher.callCount))
}

func TestRunPipeline_NilDependencies(t *testing.T) {
	cfg := basePipelineConfig()
	ctx := t.Context()

	_, err := services.RunPipeline(ctx, nil, &mockPipelineVerifier{}, cfg, nil, services.PipelineCallbacks{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetchers list is empty")

	_, err = services.RunPipeline(
		ctx,
		[]services.SourceFetcher{{Source: common.SourceProxyMania, Fetcher: &mockPipelineFetcher{}}},
		nil,
		cfg,
		nil,
		services.PipelineCallbacks{},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "verifier is nil")
}

func TestRunPipeline_FetcherReturnsEmptyList(t *testing.T) {
	mockFetcher := &mockPipelineFetcher{
		responses: map[string][]*fetcher.ProxyItem{string(common.ProxySOCKS5): {}},
	}
	mockVerifier := &mockPipelineVerifier{}
	fetchers := []services.SourceFetcher{
		{Source: common.SourceProxyMania, Fetcher: mockFetcher},
	}

	var fetchedTotal int
	cb := services.PipelineCallbacks{
		OnFetched: func(total int) { fetchedTotal = total },
	}

	ctx := t.Context()
	cfg := basePipelineConfig()

	validProxies, err := services.RunPipeline(ctx, fetchers, mockVerifier, cfg, nil, cb)

	require.NoError(t, err)
	assert.Empty(t, validProxies)
	assert.Equal(t, 0, fetchedTotal)
}

func TestRunPipeline_AllProxiesInvalid(t *testing.T) {
	items := []*fetcher.ProxyItem{
		{Host: "1.1.1.1", Port: "8080", Type: common.ProxySOCKS5},
	}

	mockFetcher := &mockPipelineFetcher{
		responses: map[string][]*fetcher.ProxyItem{string(common.ProxySOCKS5): items},
	}
	mockVerifier := &mockPipelineVerifier{
		responses: map[string]services.Result{
			"1.1.1.1:8080": {Error: errors.New("timeout")},
		},
	}

	fetchers := []services.SourceFetcher{
		{Source: common.SourceProxyMania, Fetcher: mockFetcher},
	}

	ctx := t.Context()
	cfg := basePipelineConfig()

	validProxies, err := services.RunPipeline(ctx, fetchers, mockVerifier, cfg, nil, services.PipelineCallbacks{})

	require.NoError(t, err)
	assert.Empty(t, validProxies, "Invalid proxies must be filtered out")
}

func TestRunPipeline_SuccessfulPipeline(t *testing.T) {
	items := []*fetcher.ProxyItem{
		{Host: "1.1.1.1", Port: "8080", Type: common.ProxySOCKS5},
		{Host: "2.2.2.2", Port: "8080", Type: common.ProxySOCKS5},
	}

	mockFetcher := &mockPipelineFetcher{
		responses: map[string][]*fetcher.ProxyItem{string(common.ProxySOCKS5): items},
	}
	mockVerifier := &mockPipelineVerifier{
		responses: map[string]services.Result{
			"1.1.1.1:8080": {ReqLatency: 50 * time.Millisecond, StatusCode: 200},
			"2.2.2.2:8080": {ReqLatency: 10 * time.Millisecond, StatusCode: 200},
		},
	}

	fetchers := []services.SourceFetcher{
		{Source: common.SourceProxyMania, Fetcher: mockFetcher},
	}

	var fetchedTotal int
	var progressCalls []int
	var progressMutex sync.Mutex

	cb := services.PipelineCallbacks{
		OnFetched: func(total int) { fetchedTotal = total },
		OnProgress: func(current, _ int) {
			progressMutex.Lock()
			defer progressMutex.Unlock()
			progressCalls = append(progressCalls, current)
		},
	}

	ctx := t.Context()
	cfg := basePipelineConfig()

	validProxies, err := services.RunPipeline(ctx, fetchers, mockVerifier, cfg, nil, cb)

	require.NoError(t, err)
	require.Len(t, validProxies, 2, "Both valid proxies must be returned")

	assert.Equal(t, 2, fetchedTotal, "OnFetched callback must receive total items count")
	assert.Len(t, progressCalls, 2, "OnProgress must be called for each item")

	// CheckBatch sorts by ReqLatency, so 2.2.2.2 (10ms) must be first
	assert.Equal(t, "2.2.2.2", validProxies[0].Host)
	assert.Equal(t, "1.1.1.1", validProxies[1].Host)
}

func TestRunPipeline_MultipleFetchersAggregatesResults(t *testing.T) {
	items1 := []*fetcher.ProxyItem{
		{Host: "1.1.1.1", Port: "8080", Type: common.ProxySOCKS5},
	}
	items2 := []*fetcher.ProxyItem{
		{Host: "2.2.2.2", Port: "8080", Type: common.ProxySOCKS5},
	}

	mockFetcher1 := &mockPipelineFetcher{
		responses: map[string][]*fetcher.ProxyItem{string(common.ProxySOCKS5): items1},
	}
	mockFetcher2 := &mockPipelineFetcher{
		responses: map[string][]*fetcher.ProxyItem{string(common.ProxySOCKS5): items2},
	}

	fetchers := []services.SourceFetcher{
		{Source: common.SourceProxyMania, Fetcher: mockFetcher1},
		{Source: common.SourceTheSpeedX, Fetcher: mockFetcher2},
	}

	mockVerifier := &mockPipelineVerifier{
		responses: map[string]services.Result{
			"1.1.1.1:8080": {ReqLatency: 10 * time.Millisecond, StatusCode: 200},
			"2.2.2.2:8080": {ReqLatency: 20 * time.Millisecond, StatusCode: 200},
		},
	}

	var fetchedTotal int
	cb := services.PipelineCallbacks{
		OnFetched: func(total int) { fetchedTotal = total },
	}

	ctx := t.Context()
	cfg := basePipelineConfig()

	validProxies, err := services.RunPipeline(ctx, fetchers, mockVerifier, cfg, nil, cb)

	require.NoError(t, err)
	assert.Equal(t, 2, fetchedTotal, "OnFetched must receive aggregated count from all fetchers")
	require.Len(t, validProxies, 2, "Must aggregate and validate items from multiple fetchers")
	assert.Equal(t, "1.1.1.1", validProxies[0].Host)
	assert.Equal(t, "2.2.2.2", validProxies[1].Host)
}

func TestRunPipeline_StopsOnContextCancellation(t *testing.T) {
	mockFetcher := &mockPipelineFetcher{
		blockChan: make(chan struct{}),
	}
	mockVerifier := &mockPipelineVerifier{}
	fetchers := []services.SourceFetcher{
		{Source: common.SourceProxyMania, Fetcher: mockFetcher},
	}

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	done := make(chan []*services.ProxyItemFull)
	go func() {
		res, _ := services.RunPipeline(ctx, fetchers, mockVerifier, basePipelineConfig(), nil, services.PipelineCallbacks{})
		done <- res
	}()

	select {
	case <-done:
		// Success, pipeline exited on context cancel
	case <-time.After(2 * time.Second):
		t.Fatal("RunPipeline did not exit on context cancellation, potential deadlock")
	}
}
