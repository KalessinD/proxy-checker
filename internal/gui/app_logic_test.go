// nolint testpackage
package gui

import (
	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"proxy-checker/internal/services"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMapToWrapper_Success(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	g := NewAppGUI(testApp, cfg, logger, "dev")

	inputItems := []*services.ProxyItemFull{
		{
			ProxyItem:   services.ProxyItem{Host: "1.1.1.1", Port: "8080", Type: common.ProxySOCKS5, Country: "US", Source: common.SourceProxyMania},
			CheckResult: services.Result{ProxyLatencyStr: "10ms", ReqLatencyStr: "20ms"},
		},
		{
			ProxyItem:   services.ProxyItem{Host: "2.2.2.2", Port: "3128", Type: common.ProxyHTTP, Country: "GB"},
			CheckResult: services.Result{ProxyLatencyStr: "N/A", ReqLatencyStr: "N/A"},
		},
	}

	wrappers := g.mapToWrapper(inputItems)

	require.Len(t, wrappers, 2, "Must map exactly 2 items")

	assert.Equal(t, "proxymania", wrappers[0].Source)
	assert.Equal(t, "1.1.1.1", wrappers[0].Host)
	assert.Equal(t, "8080", wrappers[0].Port)
	assert.Equal(t, common.ProxySOCKS5, wrappers[0].Type)
	assert.Equal(t, "US", wrappers[0].Country)
	assert.Equal(t, "10ms", wrappers[0].TCP)
	assert.Equal(t, "20ms", wrappers[0].HTTP)

	assert.Equal(t, "2.2.2.2", wrappers[1].Host)
	assert.Equal(t, "N/A", wrappers[1].TCP)
}

func TestMapToWrapper_EmptySlice(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	g := NewAppGUI(testApp, cfg, logger, "dev")

	wrappers := g.mapToWrapper([]*services.ProxyItemFull{})

	require.Empty(t, wrappers, "Empty input must return empty slice")
}

func TestApplyTheme_NoPanics(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	g := NewAppGUI(testApp, cfg, logger, "dev")

	themes := []string{"light", "dark", "system", "unknown_theme_fallback"}

	for _, themeName := range themes {
		t.Run(themeName, func(t *testing.T) {
			assert.NotPanics(t, func() {
				g.applyTheme(themeName)
			}, "Applying theme must not panic on any string input")
		})
	}
}

func TestDeduplicateItems(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	g := NewAppGUI(testApp, cfg, logger, "dev")

	items := []*services.ProxyItemFull{
		{ProxyItem: services.ProxyItem{Host: "1.1.1.1", Port: "8080"}},
		{ProxyItem: services.ProxyItem{Host: "1.1.1.1", Port: "8080"}},
		{ProxyItem: services.ProxyItem{Host: "2.2.2.2", Port: "8080"}},
	}

	result := g.deduplicateItems(items)

	require.Len(t, result, 2, "Must remove exact duplicates based on Host:Port")
	assert.Equal(t, "1.1.1.1", result[0].Host)
	assert.Equal(t, "2.2.2.2", result[1].Host)
}

func TestGetTargetURL_CustomTarget(t *testing.T) {
	testApp := test.NewTempApp(t)
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	g := NewAppGUI(testApp, cfg, logger, "dev")

	// When custom target is active, it must override config
	g.isCustomTarget = true
	g.customTargetURL = "custom.com"

	actualURL := g.getTargetURL()
	assert.Equal(t, "custom.com", actualURL, "Must return custom URL when isCustomTarget is true")

	// When custom target is inactive, it must return config value
	g.isCustomTarget = false
	g.cfg.DestAddr = "default.com"

	actualURL = g.getTargetURL()
	assert.Equal(t, "default.com", actualURL, "Must return config DestAddr when isCustomTarget is false")
}

func TestHighlightProxyInList_Found(t *testing.T) {
	testApp := test.NewTempApp(t)
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	g := NewAppGUI(testApp, cfg, logger, "dev")

	g.proxyItems = []*ProxyItemWrapper{
		{Host: "1.1.1.1", Port: "8080"},
		{Host: "2.2.2.2", Port: "3128"},
		{Host: "3.3.3.3", Port: "1080"},
	}

	g.highlightProxyInList("2.2.2.2", "3128")

	assert.Equal(t, 1, g.highlightedRow, "Must update highlightedRow to the correct index")
}

func TestHighlightProxyInList_NotFound(t *testing.T) {
	testApp := test.NewTempApp(t)
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	g := NewAppGUI(testApp, cfg, logger, "dev")

	g.proxyItems = []*ProxyItemWrapper{
		{Host: "1.1.1.1", Port: "8080"},
	}

	g.highlightProxyInList("9.9.9.9", "9999")

	assert.Equal(t, -1, g.highlightedRow, "Must set highlightedRow to -1 if proxy is not found")
}
