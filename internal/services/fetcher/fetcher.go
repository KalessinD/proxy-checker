package fetcher

import (
	"context"
	"proxy-checker/internal/common"
)

type ProxyItem struct {
	Host    string           `json:"host"`
	Port    string           `json:"port"`
	Country string           `json:"country"`
	Type    common.ProxyType `json:"type"`
	RTT     string           `json:"rtt"`
	RTTms   int              `json:"rtt_ms"`
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
