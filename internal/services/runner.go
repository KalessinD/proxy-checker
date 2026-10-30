package services

import (
	"context"
	"errors"
	"fmt"
	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"proxy-checker/internal/fetcher"
)

//go:generate mockgen -destination=../../internal/mocks/mock_services.go -package=mocks proxy-checker/internal/services PipelineRunner,ProxyVerifier,ListCache

type (
	PipelineCallbacks struct {
		OnFetched  func(total int)
		OnProgress func(current, total int)
	}

	// PipelineConfig is the narrow projection of the application configuration
	// relevant to the pipeline. It decouples the pipeline from the full config.
	PipelineConfig struct {
		// CheckParams carries the network-validation settings (promoted fields).
		CheckParams

		RTT   int
		Pages int
		Lang  string
	}

	// PipelineParams groups all required dependencies and configuration for running the pipeline.
	PipelineParams struct {
		Fetchers []SourceFetcher
		Verifier ProxyVerifier
		Config   *PipelineConfig
		Resolver common.GeoIPResolver
		// ListCache is an optional cache of raw source lists; nil disables list caching.
		ListCache ListCache
	}

	// ListCache caches raw proxy lists between pipeline runs. Implementations log
	// their operational failures themselves; a returned error only steers the
	// fetch fallback and must never break the scan.
	ListCache interface {
		// Load returns the cached raw items; ErrListCacheMiss signals a stale or absent entry.
		Load(source common.Source, proxyType common.ProxyType) ([]*fetcher.ProxyItem, error)
		Save(source common.Source, proxyType common.ProxyType, items []*fetcher.ProxyItem) error
	}

	// defaultPipelineRunner is the concrete implementation of PipelineRunner.
	defaultPipelineRunner struct{}

	// PipelineRunner defines an interface for executing the proxy fetching and checking pipeline.
	// This allows decoupling GUI/CLI from the concrete implementation for testability.
	PipelineRunner interface {
		Run(ctx context.Context, params PipelineParams, cb PipelineCallbacks) ([]*ProxyItemFull, error)
	}

	// SourceFetcher binds a ProxySource implementation with its corresponding Source identifier.
	SourceFetcher struct {
		Source  common.Source
		Fetcher ProxySource
	}

	// ProxySource defines the port for retrieving raw proxy items from an
	// external source. It is implemented by fetcher adapters.
	ProxySource interface {
		Fetch(ctx context.Context, settings fetcher.Settings) ([]*fetcher.ProxyItem, error)
	}
)

// ErrListCacheMiss signals that a raw list is absent or expired in the cache.
// It is returned by ListCache implementations and consumed by the pipeline.
var ErrListCacheMiss = errors.New("list cache miss")

// NewPipelineConfig projects the application configuration onto the narrow pipeline DTO.
func NewPipelineConfig(cfg *config.Config) *PipelineConfig {
	return &PipelineConfig{
		CheckParams: *NewCheckParams(cfg),
		RTT:         cfg.RTT,
		Pages:       cfg.Pages,
		Lang:        cfg.Lang,
	}
}

// NewDefaultPipelineRunner creates a standard PipelineRunner.
func NewDefaultPipelineRunner() PipelineRunner {
	return &defaultPipelineRunner{}
}

// Run executes the pipeline using the package-level RunPipeline function.
func (r *defaultPipelineRunner) Run(
	ctx context.Context,
	params PipelineParams,
	cb PipelineCallbacks,
) ([]*ProxyItemFull, error) {
	return RunPipeline(ctx, params, cb)
}

// RunPipeline fetches proxies from all configured sources and validates them concurrently.
// A fetcher failure terminates the pipeline; per-proxy check failures only filter the item out.
func RunPipeline(ctx context.Context, params PipelineParams, cb PipelineCallbacks) ([]*ProxyItemFull, error) {
	if len(params.Fetchers) == 0 {
		return nil, errors.New("pipeline initialization error: fetchers list is empty")
	}
	if params.Verifier == nil {
		return nil, errors.New("pipeline initialization error: verifier is nil")
	}
	if params.Config == nil {
		return nil, errors.New("pipeline initialization error: config is nil")
	}

	settings := fetcher.Settings{
		Type:     params.Config.Type,
		MaxRTT:   params.Config.RTT,
		Pages:    params.Config.Pages,
		Timeout:  int(params.Config.Timeout),
		Resolver: params.Resolver,
		Lang:     params.Config.Lang,
	}

	var allProxies []*fetcher.ProxyItem
	for _, f := range params.Fetchers {
		items, err := loadCachedOrFetch(ctx, f, settings, params.ListCache)
		if err != nil {
			return nil, fmt.Errorf("pipeline fetch error: %w", err)
		}
		for _, item := range items {
			item.Source = f.Source
		}
		allProxies = append(allProxies, items...)
	}

	if cb.OnFetched != nil {
		cb.OnFetched(len(allProxies))
	}

	checker := NewProxyChecker()

	validProxies := checker.CheckBatch(
		ctx,
		allProxies,
		&params.Config.CheckParams,
		params.Verifier,
		cb.OnProgress,
	)

	return validProxies, nil
}

// loadCachedOrFetch returns raw proxy items from the list cache when it is fresh,
// falling back to the regular fetch and refreshing the cache afterwards.
func loadCachedOrFetch(
	ctx context.Context,
	sourceFetcher SourceFetcher,
	settings fetcher.Settings,
	listCache ListCache,
) ([]*fetcher.ProxyItem, error) {
	if listCache != nil {
		// Any cache failure (miss or read error) simply falls back to the regular fetch.
		if cached, loadErr := listCache.Load(sourceFetcher.Source, settings.Type); loadErr == nil {
			return cached, nil
		}
	}

	items, err := sourceFetcher.Fetcher.Fetch(ctx, settings)
	if err != nil {
		return nil, err
	}

	if listCache != nil {
		// Write failures are logged by the adapter; they only cost the next run a re-fetch.
		_ = listCache.Save(sourceFetcher.Source, settings.Type, items)
	}

	return items, nil
}
