package fetcher

import (
	"context"
	"proxy-checker/internal/common"
)

// ProxyItem представляет одну запись прокси
type ProxyItem struct {
	Host    string
	Port    string
	Country string
	Type    common.ProxyType // ИСПОЛЬЗУЕМ ТИП
	RTT     string
	RTTms   int
}

// Settings параметры для получения прокси
type Settings struct {
	Type    common.ProxyType // ИСПОЛЬЗУЕМ ТИП
	MaxRTT  int
	Pages   int
	Timeout int
}

// Fetcher интерфейс для источника прокси
type Fetcher interface {
	Fetch(ctx context.Context, settings Settings) ([]ProxyItem, error)
}

// NewFetcher фабричный метод
func NewFetcher(source common.Source) Fetcher { // ИСПОЛЬЗУЕМ ТИП
	switch source {
	case common.SourceTheSpeedX:
		return &TheSpeedXFetcher{}
	case common.SourceProxyMania:
		return &ProxyManiaFetcher{}
	default:
		return &ProxyManiaFetcher{}
	}
}
