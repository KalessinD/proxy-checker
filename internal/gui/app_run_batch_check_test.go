// nolint testpackage
package gui

import (
	"errors"
	"testing"

	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"proxy-checker/internal/mocks"
	"proxy-checker/internal/services"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

// newTestAppGUIForRunCheck creates an AppGUI with screen widgets initialized,
// a controlled system-proxy mock and a file cache replaced by the provided
// storage mock. The pipeline runner must be injected separately because its
// expectations differ per scenario.
func newTestAppGUIForRunCheck(
	t *testing.T,
	controller *gomock.Controller,
	mockRunner *mocks.MockPipelineRunner,
	mockStorage *mocks.MockStorageInterface,
) *AppGUI {
	t.Helper()

	testApp := test.NewApp()
	t.Cleanup(testApp.Quit)

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())

	mockSystemProxy := mocks.NewMockSystemProxyManager(controller)
	mockSystemProxy.EXPECT().IsSupported().Return(false).AnyTimes()
	mockSystemProxy.EXPECT().GetMode().Return("", nil).AnyTimes()

	deps := AppDependencies{
		FyneApp:         testApp,
		Cfg:             cfg,
		Logger:          logger,
		Version:         "dev",
		PipelineRunner:  mockRunner,
		SysProxyManager: mockSystemProxy,
		Cache:           mockStorage,
	}
	appGUI := NewAppGUI(deps)
	appGUI.ShowMainScreen()

	return appGUI
}

func TestRunBatchCheck_PersistsValidProxiesPerSource(t *testing.T) {
	ctrl := gomock.NewController(t)

	configuration := config.DefaultConfig()
	verifiedItems := []*services.ProxyItemFull{
		{ProxyItem: services.ProxyItem{Host: "1.1.1.1", Port: "1080", Source: common.SourceProxyMania}},
		{ProxyItem: services.ProxyItem{Host: "2.2.2.2", Port: "8080", Source: common.SourceTheSpeedX}},
	}

	var progressCalls []int
	mockRunner := mocks.NewMockPipelineRunner(ctrl)
	mockRunner.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, params services.PipelineParams, callbacks services.PipelineCallbacks) ([]*services.ProxyItemFull, error) {
			require.NotNil(t, params.Verifier, "Pipeline params must carry the verifier")
			require.NotNil(t, params.Config, "Pipeline params must carry the config")
			require.Len(t, params.Fetchers, len(configuration.Sources), "One fetcher per configured source must be prepared")
			if callbacks.OnFetched != nil {
				callbacks.OnFetched(len(verifiedItems))
			}
			if callbacks.OnProgress != nil {
				callbacks.OnProgress(len(verifiedItems), len(verifiedItems))
				progressCalls = append(progressCalls, len(verifiedItems))
			}
			return verifiedItems, nil
		}).Times(1)

	savedItems := make(map[common.Source][]*services.ProxyItemFull)
	mockStorage := mocks.NewMockStorageInterface(ctrl)
	// runBatchCheck only saves results: cache loading happens on the startup path.
	for _, src := range configuration.Sources {
		mockStorage.EXPECT().
			Save(src, configuration.Type, gomock.Any(), configuration.CacheTTL).
			DoAndReturn(func(source common.Source, _ common.ProxyType, items []*services.ProxyItemFull, _ int) error {
				savedItems[source] = items
				return nil
			}).Times(1)
	}

	appGUI := newTestAppGUIForRunCheck(t, ctrl, mockRunner, mockStorage)
	appGUI.runBatchCheck()

	require.Len(t, progressCalls, 1, "OnProgress must be provided to the pipeline")
	require.Len(t, appGUI.proxyItems, 2, "Table must contain all verified proxies after the run")
	assert.Equal(t, "1.1.1.1", appGUI.proxyItems[0].Host)
	assert.Equal(t, "2.2.2.2", appGUI.proxyItems[1].Host)

	require.Len(t, savedItems, len(configuration.Sources), "Results must be persisted per configured source")
	for source, items := range savedItems {
		require.NotEmpty(t, items, "Each persisted bucket must hold its own proxies")
		for _, item := range items {
			assert.Equal(t, source, item.Source, "Cached item must belong to its source bucket")
		}
	}
}

func TestRunBatchCheck_PipelineErrorLeavesTableEmptyAndSkipsCacheSave(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockRunner := mocks.NewMockPipelineRunner(ctrl)
	pipelineError := errors.New("pipeline fetch error: source unavailable")
	mockRunner.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, pipelineError).
		Times(1)

	mockStorage := mocks.NewMockStorageInterface(ctrl)
	// No Save expectations: any Save call would fail the gomock controller,
	// asserting that errors must not touch the cache.

	appGUI := newTestAppGUIForRunCheck(t, ctrl, mockRunner, mockStorage)
	appGUI.runBatchCheck()

	assert.Empty(t, appGUI.proxyItems, "On pipeline error the table must stay empty")
}

func TestRunBatchCheck_CacheSaveErrorMustNotDropTableResults(t *testing.T) {
	ctrl := gomock.NewController(t)

	configuration := config.DefaultConfig()
	verifiedItems := []*services.ProxyItemFull{
		{ProxyItem: services.ProxyItem{Host: "3.3.3.3", Port: "3128", Source: common.SourceProxyMania}},
	}
	mockRunner := mocks.NewMockPipelineRunner(ctrl)
	mockRunner.EXPECT().
		Run(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, _ services.PipelineParams, callbacks services.PipelineCallbacks) ([]*services.ProxyItemFull, error) {
			if callbacks.OnFetched != nil {
				callbacks.OnFetched(len(verifiedItems))
			}
			return verifiedItems, nil
		}).Times(1)

	storageError := errors.New("cache storage failure (test)")
	mockStorage := mocks.NewMockStorageInterface(ctrl)
	// runBatchCheck only saves results: cache loading happens on the startup path.
	mockStorage.EXPECT().
		Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(storageError).
		Times(len(configuration.Sources))

	appGUI := newTestAppGUIForRunCheck(t, ctrl, mockRunner, mockStorage)
	appGUI.runBatchCheck()

	require.Len(t, appGUI.proxyItems, 1, "Cache failure must not discard verified results from the table")
	assert.Equal(t, "3.3.3.3", appGUI.proxyItems[0].Host)
}
