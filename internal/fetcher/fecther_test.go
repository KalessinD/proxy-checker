package fetcher_test

import (
	"proxy-checker/internal/common"
	"proxy-checker/internal/fetcher"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFetcher_KnownSources(t *testing.T) {
	t.Run("ProxyMania source", func(t *testing.T) {
		fetcherInstance := fetcher.NewFetcher(common.SourceProxyMania)
		assert.IsType(t, &fetcher.ProxyManiaFetcher{}, fetcherInstance)
	})

	t.Run("TheSpeedX source", func(t *testing.T) {
		fetcherInstance := fetcher.NewFetcher(common.SourceTheSpeedX)
		assert.IsType(t, &fetcher.TextListFetcher{}, fetcherInstance)
	})
}

func TestNewFetcher_UnknownSource(t *testing.T) {
	unknownSource := common.Source("totally_unknown_source")
	fetcherInstance := fetcher.NewFetcher(unknownSource)

	assert.IsType(t, &fetcher.ProxyManiaFetcher{}, fetcherInstance)
}
