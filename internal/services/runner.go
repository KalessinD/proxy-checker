package services

import (
	"context"
	"errors"
	"fmt"
	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"proxy-checker/internal/fetcher"
)

type (
	PipelineCallbacks struct {
		OnFetched  func(total int)
		OnProgress func(current, total int)
	}

	// SourceFetcher binds a Fetcher implementation with its corresponding Source identifier.
	SourceFetcher struct {
		Source  common.Source
		Fetcher fetcher.Fetcher
	}
)

func RunPipeline(
	ctx context.Context,
	fetchers []SourceFetcher,
	verifierInstance ProxyVerifier,
	cfg *config.Config,
	resolver common.GeoIPResolver,
	cb PipelineCallbacks,
) ([]*ProxyItemFull, error) {
	if len(fetchers) == 0 {
		return nil, errors.New("pipeline initialization error: fetchers list is empty")
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

	var allProxies []*fetcher.ProxyItem
	for _, f := range fetchers {
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
