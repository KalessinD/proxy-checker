package services

import (
	"proxy-checker/internal/common"
	"proxy-checker/internal/services/fetcher"
)

// NewFetcher обертка для фабрики из пакета fetcher
func NewFetcher(source common.Source) fetcher.Fetcher {
	return fetcher.NewFetcher(source)
}
