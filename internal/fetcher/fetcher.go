package fetcher

import (
	"context"
	"proxy-checker/internal/common"
	"time"
)

type (
	ProxyItem struct {
		Host    string           `json:"host"`
		Port    string           `json:"port"`
		Country string           `json:"country"`
		Type    common.ProxyType `json:"type"`
		RTT     string           `json:"rtt"`
		RTTms   int              `json:"rtt_ms"`
		Source  common.Source    `json:"source"`
	}

	Settings struct {
		Type     common.ProxyType
		MaxRTT   int
		Pages    int
		Timeout  int
		Resolver common.GeoIPResolver
		Lang     string
	}

	Fetcher interface {
		Fetch(ctx context.Context, settings Settings) ([]*ProxyItem, error)
	}
)

const fetcherClientTimeout = 20 * time.Second

func NewFetcher(source common.Source, logger common.LoggerInterface) Fetcher {
	switch source {
	case common.SourceProxifly:
		return NewTextListFetcher(ProxiFlyURL, NewProxiflyProvider(), logger)
	case common.SourceTheSpeedX:
		return NewTextListFetcher(TheSpeedXBaseURL, NewTheSpeedXProvider(), logger)
	case common.SourceProxyMania:
		return NewProxyManiaFetcher(logger)
	default:
		return NewProxyManiaFetcher(logger)
	}
}
