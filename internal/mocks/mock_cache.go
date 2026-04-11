package mocks

import (
	"proxy-checker/internal/common"
	"proxy-checker/internal/services"
)

// MockCacheStorage is a mock implementation of cache.StorageInterface for testing.
type MockCacheStorage struct {
	MockLoad        func(source common.Source, proxyType common.ProxyType) ([]*services.ProxyItemFull, error)
	MockSave        func(source common.Source, proxyType common.ProxyType, items []*services.ProxyItemFull, ttl int) error
	GetFilePathMock func() string
}

func (m *MockCacheStorage) Load(source common.Source, proxyType common.ProxyType) ([]*services.ProxyItemFull, error) {
	if m.MockLoad != nil {
		return m.MockLoad(source, proxyType)
	}
	return []*services.ProxyItemFull{}, nil
}

func (m *MockCacheStorage) Save(source common.Source, proxyType common.ProxyType, items []*services.ProxyItemFull, ttl int) error {
	if m.MockSave != nil {
		return m.MockSave(source, proxyType, items, ttl)
	}
	return nil
}

func (m *MockCacheStorage) GetFilePath() string {
	if m.GetFilePathMock != nil {
		return m.GetFilePathMock()
	}
	return "/tmp/mock-cache.data"
}
