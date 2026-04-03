package services

import (
	"context"
	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"proxy-checker/internal/services/fetcher"
)

// PipelineCallbacks позволяет клиентам (GUI, CLI) реагировать на этапы пайплайна
type PipelineCallbacks struct {
	OnFetched  func(total int)
	OnProgress func(current, total int)
}

// RunPipeline запускает полный цикл: Fetch -> Check
func RunPipeline(ctx context.Context, cfg *config.Config, resolver common.GeoIPResolver, cb PipelineCallbacks) ([]*ProxyItemFull, error) {
	f := NewFetcher(cfg.Source)
	settings := fetcher.Settings{
		Type:     cfg.Type,
		MaxRTT:   cfg.RTT,
		Pages:    cfg.Pages,
		Timeout:  int(cfg.Timeout),
		Resolver: resolver,
		Lang:     cfg.Lang,
	}

	allProxies, err := f.Fetch(ctx, settings)
	if err != nil {
		return nil, err
	}

	if cb.OnFetched != nil {
		cb.OnFetched(len(allProxies))
	}

	validProxies := CheckBatch(
		ctx,
		allProxies,
		cfg.DestAddr,
		cfg.Type,
		cfg.Timeout,
		cfg.Workers,
		cfg.CheckHTTP2,
		cb.OnProgress,
		&defaultVerifier{},
	)

	return validProxies, nil
}
