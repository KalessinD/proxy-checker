package mocks

import (
	"context"
	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"proxy-checker/internal/services"
)

// MockPipelineRunner is a mock implementation of services.PipelineRunner for testing.
type MockPipelineRunner struct {
	MockRun func(
		ctx context.Context,
		fetchers []services.SourceFetcher,
		verifier services.ProxyVerifier,
		cfg *config.Config,
		resolver common.GeoIPResolver,
		cb services.PipelineCallbacks,
	) ([]*services.ProxyItemFull, error)
}

func (m *MockPipelineRunner) Run(
	ctx context.Context,
	fetchers []services.SourceFetcher,
	verifier services.ProxyVerifier,
	cfg *config.Config,
	resolver common.GeoIPResolver,
	cb services.PipelineCallbacks,
) ([]*services.ProxyItemFull, error) {
	if m.MockRun != nil {
		return m.MockRun(ctx, fetchers, verifier, cfg, resolver, cb)
	}
	return []*services.ProxyItemFull{}, nil
}
