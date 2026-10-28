package services_test

import (
	"context"
	"errors"
	"os"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/config"
	"proxy-checker/internal/fetcher"
	"proxy-checker/internal/mocks"
	"proxy-checker/internal/services"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	_ = i18n.Init("en")
	os.Exit(m.Run())
}

// basePipelineConfig builds the narrow pipeline config used in pipeline tests.
func basePipelineConfig() *services.PipelineConfig {
	return services.NewPipelineConfig(&config.Config{
		Type:       common.ProxySOCKS5,
		Sources:    []common.Source{common.SourceProxyMania},
		Timeout:    5 * time.Second,
		Workers:    2,
		DestAddr:   "google.com",
		CheckHTTP2: false,
		Lang:       "en",
	})
}

// expectedFetchSettings computes the exact settings RunPipeline must pass to fetchers.
func expectedFetchSettings(cfg *services.PipelineConfig) fetcher.Settings {
	return fetcher.Settings{
		Type:     cfg.Type,
		MaxRTT:   cfg.RTT,
		Pages:    cfg.Pages,
		Timeout:  int(cfg.Timeout),
		Resolver: nil,
		Lang:     cfg.Lang,
	}
}

// expectVerify registers a verifier expectation for a single proxy address.
func expectVerify(
	ctrl *gomock.Controller,
	proxyAddr string,
	result services.Result,
) *mocks.MockProxyVerifier {
	verifierMock := mocks.NewMockProxyVerifier(ctrl)
	verifierMock.EXPECT().
		Verify(gomock.Any(), proxyAddr, "google.com", "socks5", false).
		Return(result)
	return verifierMock
}

func TestRunPipeline_FetcherReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	cfg := basePipelineConfig()

	fetcherMock := mocks.NewMockFetcher(ctrl)
	fetcherMock.EXPECT().
		Fetch(gomock.Any(), expectedFetchSettings(cfg)).
		Return(nil, errors.New("network error"))

	validProxies, pipelineErr := services.RunPipeline(t.Context(), services.PipelineParams{
		Fetchers: []services.SourceFetcher{
			{Source: common.SourceProxyMania, Fetcher: fetcherMock},
		},
		Verifier: mocks.NewMockProxyVerifier(ctrl),
		Config:   cfg,
	}, services.PipelineCallbacks{})

	require.Error(t, pipelineErr)
	assert.Contains(t, pipelineErr.Error(), "pipeline fetch error")
	assert.Nil(t, validProxies)
}

func TestRunPipeline_NilDependencies(t *testing.T) {
	ctrl := gomock.NewController(t)
	cfg := basePipelineConfig()
	ctx := t.Context()

	_, pipelineErr := services.RunPipeline(ctx, services.PipelineParams{
		Verifier: mocks.NewMockProxyVerifier(ctrl),
		Config:   cfg,
	}, services.PipelineCallbacks{})
	require.Error(t, pipelineErr)
	assert.Contains(t, pipelineErr.Error(), "fetchers list is empty")

	_, pipelineErr = services.RunPipeline(ctx, services.PipelineParams{
		Fetchers: []services.SourceFetcher{
			{Source: common.SourceProxyMania, Fetcher: mocks.NewMockFetcher(ctrl)},
		},
		Verifier: nil,
		Config:   cfg,
	}, services.PipelineCallbacks{})
	require.Error(t, pipelineErr)
	assert.Contains(t, pipelineErr.Error(), "verifier is nil")

	_, pipelineErr = services.RunPipeline(ctx, services.PipelineParams{
		Fetchers: []services.SourceFetcher{
			{Source: common.SourceProxyMania, Fetcher: mocks.NewMockFetcher(ctrl)},
		},
		Verifier: mocks.NewMockProxyVerifier(ctrl),
		Config:   nil,
	}, services.PipelineCallbacks{})
	require.Error(t, pipelineErr)
	assert.Contains(t, pipelineErr.Error(), "config is nil")
}

func TestRunPipeline_FetcherReturnsEmptyList(t *testing.T) {
	ctrl := gomock.NewController(t)
	cfg := basePipelineConfig()

	fetcherMock := mocks.NewMockFetcher(ctrl)
	fetcherMock.EXPECT().
		Fetch(gomock.Any(), expectedFetchSettings(cfg)).
		Return([]*fetcher.ProxyItem{}, nil)

	var fetchedTotal int
	validProxies, pipelineErr := services.RunPipeline(t.Context(), services.PipelineParams{
		Fetchers: []services.SourceFetcher{
			{Source: common.SourceProxyMania, Fetcher: fetcherMock},
		},
		Verifier: mocks.NewMockProxyVerifier(ctrl),
		Config:   cfg,
	}, services.PipelineCallbacks{
		OnFetched: func(total int) { fetchedTotal = total },
	})

	require.NoError(t, pipelineErr)
	assert.Empty(t, validProxies)
	assert.Equal(t, 0, fetchedTotal)
}

func TestRunPipeline_AllProxiesInvalid(t *testing.T) {
	ctrl := gomock.NewController(t)
	cfg := basePipelineConfig()

	fetcherMock := mocks.NewMockFetcher(ctrl)
	fetcherMock.EXPECT().
		Fetch(gomock.Any(), expectedFetchSettings(cfg)).
		Return([]*fetcher.ProxyItem{
			{Host: "1.1.1.1", Port: "8080", Type: common.ProxySOCKS5},
		}, nil)

	verifierMock := expectVerify(ctrl, "1.1.1.1:8080", services.Result{Error: errors.New("timeout")})

	validProxies, pipelineErr := services.RunPipeline(t.Context(), services.PipelineParams{
		Fetchers: []services.SourceFetcher{
			{Source: common.SourceProxyMania, Fetcher: fetcherMock},
		},
		Verifier: verifierMock,
		Config:   cfg,
	}, services.PipelineCallbacks{})

	require.NoError(t, pipelineErr)
	assert.Empty(t, validProxies, "Invalid proxies must be filtered out")
}

func TestRunPipeline_SuccessfulPipeline(t *testing.T) {
	ctrl := gomock.NewController(t)
	cfg := basePipelineConfig()

	fetcherMock := mocks.NewMockFetcher(ctrl)
	fetcherMock.EXPECT().
		Fetch(gomock.Any(), expectedFetchSettings(cfg)).
		Return([]*fetcher.ProxyItem{
			{Host: "1.1.1.1", Port: "8080", Type: common.ProxySOCKS5},
			{Host: "2.2.2.2", Port: "8080", Type: common.ProxySOCKS5},
		}, nil)

	verifierMock := mocks.NewMockProxyVerifier(ctrl)
	verifierMock.EXPECT().
		Verify(gomock.Any(), "1.1.1.1:8080", "google.com", "socks5", false).
		Return(services.Result{ReqLatency: 50 * time.Millisecond, StatusCode: 200})
	verifierMock.EXPECT().
		Verify(gomock.Any(), "2.2.2.2:8080", "google.com", "socks5", false).
		Return(services.Result{ReqLatency: 10 * time.Millisecond, StatusCode: 200})

	var fetchedTotal int
	var progressCalls []int
	var progressMutex sync.Mutex

	validProxies, pipelineErr := services.RunPipeline(t.Context(), services.PipelineParams{
		Fetchers: []services.SourceFetcher{
			{Source: common.SourceProxyMania, Fetcher: fetcherMock},
		},
		Verifier: verifierMock,
		Config:   cfg,
	}, services.PipelineCallbacks{
		OnFetched: func(total int) { fetchedTotal = total },
		OnProgress: func(current, _ int) {
			progressMutex.Lock()
			defer progressMutex.Unlock()
			progressCalls = append(progressCalls, current)
		},
	})

	require.NoError(t, pipelineErr)
	require.Len(t, validProxies, 2, "Both valid proxies must be returned")

	assert.Equal(t, 2, fetchedTotal, "OnFetched callback must receive total items count")
	assert.Len(t, progressCalls, 2, "OnProgress must be called for each item")

	// CheckBatch sorts by ReqLatency, so 2.2.2.2 (10ms) must be first
	assert.Equal(t, "2.2.2.2", validProxies[0].Host)
	assert.Equal(t, "1.1.1.1", validProxies[1].Host)
}

func TestRunPipeline_MultipleFetchersAggregatesResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	cfg := basePipelineConfig()

	fetcherMockProxyMania := mocks.NewMockFetcher(ctrl)
	fetcherMockProxyMania.EXPECT().
		Fetch(gomock.Any(), expectedFetchSettings(cfg)).
		Return([]*fetcher.ProxyItem{
			{Host: "1.1.1.1", Port: "8080", Type: common.ProxySOCKS5},
		}, nil)

	fetcherMockTheSpeedX := mocks.NewMockFetcher(ctrl)
	fetcherMockTheSpeedX.EXPECT().
		Fetch(gomock.Any(), expectedFetchSettings(cfg)).
		Return([]*fetcher.ProxyItem{
			{Host: "2.2.2.2", Port: "8080", Type: common.ProxySOCKS5},
		}, nil)

	verifierMock := mocks.NewMockProxyVerifier(ctrl)
	verifierMock.EXPECT().
		Verify(gomock.Any(), "1.1.1.1:8080", "google.com", "socks5", false).
		Return(services.Result{ReqLatency: 10 * time.Millisecond, StatusCode: 200})
	verifierMock.EXPECT().
		Verify(gomock.Any(), "2.2.2.2:8080", "google.com", "socks5", false).
		Return(services.Result{ReqLatency: 20 * time.Millisecond, StatusCode: 200})

	var fetchedTotal int
	validProxies, pipelineErr := services.RunPipeline(t.Context(), services.PipelineParams{
		Fetchers: []services.SourceFetcher{
			{Source: common.SourceProxyMania, Fetcher: fetcherMockProxyMania},
			{Source: common.SourceTheSpeedX, Fetcher: fetcherMockTheSpeedX},
		},
		Verifier: verifierMock,
		Config:   cfg,
	}, services.PipelineCallbacks{
		OnFetched: func(total int) { fetchedTotal = total },
	})

	require.NoError(t, pipelineErr)
	assert.Equal(t, 2, fetchedTotal, "OnFetched must receive aggregated count from all fetchers")
	require.Len(t, validProxies, 2, "Must aggregate and validate items from multiple fetchers")
	assert.Equal(t, "1.1.1.1", validProxies[0].Host)
	assert.Equal(t, "2.2.2.2", validProxies[1].Host)
}

// runPipelineResult carries both RunPipeline outputs into the test goroutine.
type runPipelineResult struct {
	validProxies []*services.ProxyItemFull
	pipelineErr  error
}

func TestRunPipeline_StopsOnContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	cfg := basePipelineConfig()

	fetcherMock := mocks.NewMockFetcher(ctrl)
	fetcherMock.EXPECT().
		Fetch(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ fetcher.Settings) ([]*fetcher.ProxyItem, error) {
			// Block until the context is cancelled by the test.
			<-ctx.Done()
			return nil, ctx.Err()
		}).AnyTimes()

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	done := make(chan runPipelineResult, 1)
	go func() {
		validProxies, pipelineErr := services.RunPipeline(ctx, services.PipelineParams{
			Fetchers: []services.SourceFetcher{
				{Source: common.SourceProxyMania, Fetcher: fetcherMock},
			},
			Verifier: mocks.NewMockProxyVerifier(ctrl),
			Config:   cfg,
		}, services.PipelineCallbacks{})
		done <- runPipelineResult{validProxies: validProxies, pipelineErr: pipelineErr}
	}()

	select {
	case pipelineResult := <-done:
		require.Error(t, pipelineResult.pipelineErr, "Cancelled pipeline must report the error")
		assert.Contains(t, pipelineResult.pipelineErr.Error(), "pipeline fetch error")
		assert.Nil(t, pipelineResult.validProxies)
	case <-time.After(2 * time.Second):
		t.Fatal("RunPipeline did not exit on context cancellation, potential deadlock")
	}
}
