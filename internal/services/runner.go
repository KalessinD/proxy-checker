package services

import (
	"context"
	"errors"
	"fmt"
	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"proxy-checker/internal/fetcher"
)

type PipelineCallbacks struct {
	OnFetched  func(total int)
	OnProgress func(current, total int)
}

func RunPipeline(
	ctx context.Context,
	fetcherInstance fetcher.Fetcher,
	verifierInstance ProxyVerifier,
	cfg *config.Config,
	resolver common.GeoIPResolver,
	cb PipelineCallbacks,
) ([]*ProxyItemFull, error) {
	if fetcherInstance == nil {
		return nil, errors.New("pipeline initialization error: fetcher is nil")
	}
	if verifierInstance == nil {
		return nil, errors.New("pipeline initialization error: verifier is nil")
	}

	settings := fetcher.Settings{
		Type:     cfg.Type,
		MaxRTT:   cfg.RTT,
		Pages:    cfg.Pages,
		Timeout:  int(cfg.Timeout),
		Resolver: resolver,
		Lang:     cfg.Lang,
	}

	allProxies, err := fetcherInstance.Fetch(ctx, settings)
	if err != nil {
		return nil, fmt.Errorf("pipeline fetch error: %w", err)
	}

	if cb.OnFetched != nil {
		cb.OnFetched(len(allProxies))
	}

	checker := NewProxyChecker()

	validProxies := checker.CheckBatch(
		ctx,
		allProxies,
		cfg.DestAddr,
		cfg.Type,
		cfg.Timeout,
		cfg.Workers,
		cfg.CheckHTTP2,
		cb.OnProgress,
		verifierInstance,
	)

	return validProxies, nil
}
