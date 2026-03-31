package services

import (
	"context"
	"proxy-checker/internal/config"
	"proxy-checker/internal/services/fetcher"
)

// PipelineCallbacks позволяет клиентам (GUI, CLI) реагировать на этапы пайплайна
type PipelineCallbacks struct {
	OnFetched  func(total int)
	OnProgress func(current, total int)
}

// RunPipeline запускает полный цикл: Fetch -> Check
// Возвращает слайс валидных прокси или ошибку.
func RunPipeline(ctx context.Context, cfg *config.Config, cb PipelineCallbacks) ([]ProxyItemFull, error) {
	f := NewFetcher(cfg.Source)
	settings := fetcher.Settings{
		Type:    cfg.Type,
		MaxRTT:  cfg.RTT,
		Pages:   cfg.Pages,
		Timeout: int(cfg.Timeout),
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
	)

	return validProxies, nil
}
