package services

import (
	"context"
)

type ProxyVerifier interface {
	Verify(ctx context.Context, proxyAddr, destAddr, mode string, checkHTTP2 bool) Result
}
