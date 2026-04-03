package fetcher

import (
	"context"
	"proxy-checker/internal/common"
)

type ProxyItem struct {
	Host    string
	Port    string
	Country string
	Type    common.ProxyType
	RTT     string
	RTTms   int
}

type Settings struct {
	Type     common.ProxyType
	MaxRTT   int
	Pages    int
	Timeout  int
	Resolver common.GeoIPResolver
	Lang     string
}

type Fetcher interface {
	Fetch(ctx context.Context, settings Settings) ([]*ProxyItem, error)
}

func NewFetcher(source common.Source) Fetcher {
	switch source {
	case common.SourceTheSpeedX:
		return NewTheSpeedXFetcher()
	case common.SourceProxyMania:
		return NewProxyManiaFetcher()
	default:
		return NewProxyManiaFetcher()
	}
}
