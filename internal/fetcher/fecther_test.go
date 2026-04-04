package fetcher_test

import (
	"proxy-checker/internal/common"
	"proxy-checker/internal/fetcher"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewFetcher_KnownSources(t *testing.T) {
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	t.Run("ProxyMania source", func(t *testing.T) {
		fetcherInstance := fetcher.NewFetcher(common.SourceProxyMania, logger)
		assert.IsType(t, &fetcher.ProxyManiaFetcher{}, fetcherInstance)
	})

	t.Run("TheSpeedX source", func(t *testing.T) {
		fetcherInstance := fetcher.NewFetcher(common.SourceTheSpeedX, logger)
		assert.IsType(t, &fetcher.TextListFetcher{}, fetcherInstance)
	})

	t.Run("Proxifly source", func(t *testing.T) {
		fetcherInstance := fetcher.NewFetcher(common.SourceProxifly, logger)
		assert.IsType(t, &fetcher.TextListFetcher{}, fetcherInstance)
	})
}

func TestNewFetcher_UnknownSource(t *testing.T) {
	unknownSource := common.Source("totally_unknown_source")
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	fetcherInstance := fetcher.NewFetcher(unknownSource, logger)

	assert.IsType(t, &fetcher.ProxyManiaFetcher{}, fetcherInstance)
}
