package services

import (
	"context"
	"errors"
	"fmt"
	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"proxy-checker/internal/fetcher"
)

//go:generate mockgen -destination=../../internal/mocks/mock_services.go -package=mocks proxy-checker/internal/services PipelineRunner,ProxyVerifier

type (
	PipelineCallbacks struct {
		OnFetched  func(total int)
		OnProgress func(current, total int)
	}

	// PipelineParams groups all required dependencies and configuration for running the pipeline.
	PipelineParams struct {
		Fetchers []SourceFetcher
		Verifier ProxyVerifier
		Config   *config.Config
		Resolver common.GeoIPResolver
	}

	// defaultPipelineRunner is the concrete implementation of PipelineRunner.
	defaultPipelineRunner struct{}

	// PipelineRunner defines an interface for executing the proxy fetching and checking pipeline.
	// This allows decoupling GUI/CLI from the concrete implementation for testability.
	PipelineRunner interface {
		Run(ctx context.Context, params PipelineParams, cb PipelineCallbacks) ([]*ProxyItemFull, error)
	}

	// SourceFetcher binds a Fetcher implementation with its corresponding Source identifier.
	SourceFetcher struct {
		Source  common.Source
		Fetcher fetcher.Fetcher
	}
)

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
		items, err := f.Fetcher.Fetch(ctx, settings)
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
		params.Config.DestAddr,
		params.Config.Type,
		params.Config.Timeout,
		params.Config.Workers,
		params.Config.CheckHTTP2,
		cb.OnProgress,
		params.Verifier,
	)

	return validProxies, nil
}
