package fetcher

import (
	"context"
)

// ProxyItem представляет одну запись прокси
type ProxyItem struct {
	Host    string
	Port    string
	Country string
	Type    string
	RTT     string
	RTTms   int
}

// Settings параметры для получения прокси
type Settings struct {
	Type    string
	MaxRTT  int
	Pages   int
	Timeout int
}

// Fetcher интерфейс для источника прокси
type Fetcher interface {
	Fetch(ctx context.Context, settings Settings) ([]ProxyItem, error)
}

// NewFetcher фабричный метод (внутри пакета)
func NewFetcher(source string) Fetcher {
	switch source {
	case "thespeedx":
		return &TheSpeedXFetcher{}
	case "proxymania":
		return &ProxyManiaFetcher{}
	default:
		return &ProxyManiaFetcher{}
	}
}
