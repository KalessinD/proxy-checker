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

// newTestAppGUIWithMockStorage creates an AppGUI whose cache field is replaced
// by the provided MockStorageInterface and whose system proxy manager is a
// mock returning non-supported defaults. The controller combines both mocks
// and verifies expectations automatically at test cleanup.
// ShowMainScreen must run first: it initializes the UI widgets (filter selects,
// result table) that loadCacheForSources mutates indirectly.
func newTestAppGUIWithMockStorage(
	t *testing.T,
	controller *gomock.Controller,
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

	appGUI := NewAppGUI(
		testApp,
		cfg,
		logger,
		"dev",
		services.NewDefaultPipelineRunner(),
		mockSystemProxy,
	)
	appGUI.ShowMainScreen()
	appGUI.cache = mockStorage

	return appGUI
}

func TestLoadCacheForSources_SingleSourceMapsItems(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStorage := mocks.NewMockStorageInterface(ctrl)

	cachedItems := []*services.ProxyItemFull{
		{ProxyItem: services.ProxyItem{Host: "1.1.1.1", Port: "1080", Source: common.SourceProxyMania}},
		{ProxyItem: services.ProxyItem{Host: "2.2.2.2", Port: "8080", Source: common.SourceProxyMania}},
	}

	configuration := config.DefaultConfig()
	mockStorage.EXPECT().
		Load(configuration.Sources[0], configuration.Type).
		Return(cachedItems, nil)

	appGUI := newTestAppGUIWithMockStorage(t, ctrl, mockStorage)
	appGUI.loadCacheForSources(configuration.Sources, configuration.Type)

	require.Len(t, appGUI.proxyItems, 2, "All cached items must be mapped to the table")
	assert.Equal(t, "1.1.1.1", appGUI.proxyItems[0].Host)
	assert.Equal(t, "1080", appGUI.proxyItems[0].Port)
	assert.Equal(t, "2.2.2.2", appGUI.proxyItems[1].Host)
}

func TestLoadCacheForSources_DeduplicatesAcrossSources(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStorage := mocks.NewMockStorageInterface(ctrl)

	proxyType := common.ProxyType("socks5")
	mockStorage.EXPECT().
		Load(common.SourceProxyMania, proxyType).
		Return([]*services.ProxyItemFull{
			{ProxyItem: services.ProxyItem{Host: "1.1.1.1", Port: "1080"}},
			{ProxyItem: services.ProxyItem{Host: "2.2.2.2", Port: "8080"}},
		}, nil)
	mockStorage.EXPECT().
		Load(common.SourceTheSpeedX, proxyType).
		Return([]*services.ProxyItemFull{
			{ProxyItem: services.ProxyItem{Host: "1.1.1.1", Port: "1080"}},
		}, nil)

	appGUI := newTestAppGUIWithMockStorage(t, ctrl, mockStorage)

	sources := []common.Source{common.SourceProxyMania, common.SourceTheSpeedX}
	appGUI.loadCacheForSources(sources, proxyType)

	require.Len(t, appGUI.proxyItems, 2, "Duplicated host:port pairs across sources must be removed")
	assert.Equal(t, "1.1.1.1", appGUI.proxyItems[0].Host)
	assert.Equal(t, "2.2.2.2", appGUI.proxyItems[1].Host)
}

func TestLoadCacheForSources_LoadErrorLeavesTableEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStorage := mocks.NewMockStorageInterface(ctrl)

	storageError := errors.New("cache storage failure (test)")
	mockStorage.EXPECT().
		Load(gomock.Any(), gomock.Any()).
		Return(nil, storageError)

	appGUI := newTestAppGUIWithMockStorage(t, ctrl, mockStorage)

	configuration := config.DefaultConfig()
	appGUI.loadCacheForSources(configuration.Sources, configuration.Type)

	assert.Empty(t, appGUI.proxyItems, "On Load error the table must stay empty without panic")
}
