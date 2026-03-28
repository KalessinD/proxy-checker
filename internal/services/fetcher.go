package services

import (
	"proxy-checker/internal/services/fetcher"
)

// NewFetcher обертка для фабрики из пакета fetcher
func NewFetcher(source string) fetcher.Fetcher {
	return fetcher.NewFetcher(source)
}
