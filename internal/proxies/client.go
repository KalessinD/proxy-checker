package proxies

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"proxy-checker/internal/common/i18n"
	"strconv"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/proxy"
)

type contextDialerWrapper struct {
	Dialer proxy.Dialer
}

func (w *contextDialerWrapper) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	type result struct {
		conn net.Conn
		err  error
	}
	ch := make(chan result, 1)

	go func() {
		c, e := w.Dialer.Dial(network, addr)
		ch <- result{c, e}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		return res.conn, res.err
	}
}

func NewClient(proxyAddr, mode string, forceHTTP2 bool) (*http.Client, error) {
	var transport *http.Transport

	switch mode {
	case "socks4":
		transport = &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return socks4Dial(ctx, network, addr, proxyAddr)
			},
		}

	case "socks5":
		dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", i18n.T("proxy.err_socks5_init"), err)
		}

		contextDialer := &contextDialerWrapper{Dialer: dialer}
		transport = &http.Transport{
			DialContext: contextDialer.DialContext,
		}

	case "https":
		proxyURL, err := url.Parse("https://" + proxyAddr)
		if err != nil {
			return nil, err
		}
		transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}

	default: // http
		proxyURL, err := url.Parse("http://" + proxyAddr)
		if err != nil {
			return nil, err
		}
		transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	}

	// Включаем HTTP/2
	if err := http2.ConfigureTransport(transport); err != nil {
		return nil, err
	}

	// Принудительный HTTP/2 режим (без fallback)
	if forceHTTP2 {
		transport.ForceAttemptHTTP2 = true

		transport.TLSClientConfig = &tls.Config{
			NextProtos: []string{"h2"},
			MinVersion: tls.VersionTLS12,
		}

		transport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	}

	return &http.Client{
		Transport: transport,
	}, nil
}

func socks4Dial(ctx context.Context, network, addr, proxyAddr string) (net.Conn, error) {
	d := net.Dialer{}

	conn, err := d.DialContext(ctx, network, proxyAddr)
	if err != nil {
		return nil, err
	}

	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			conn.Close()
			return nil, err
		}
	}

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		conn.Close()
		return nil, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		conn.Close()
		return nil, err
	}

	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := net.LookupIP(host)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("%s %s", i18n.T("proxy.err_socks4_resolve"), host)
		}
		ip = ips[0]
	}

	ipv4 := ip.To4()
	if ipv4 == nil {
		conn.Close()
		return nil, errors.New(i18n.T("proxy.err_socks4_ipv4"))
	}

	req := make([]byte, 9)
	req[0] = 4
	req[1] = 1
	req[2] = byte(uint16(port >> 8))
	req[3] = byte(uint16(port))
	copy(req[4:8], ipv4)

	if _, err := conn.Write(req); err != nil {
		conn.Close()
		return nil, err
	}

	resp := make([]byte, 8)
	if _, err := io.ReadFull(conn, resp); err != nil {
		conn.Close()
		return nil, err
	}

	if resp[1] != 90 {
		conn.Close()
		return nil, fmt.Errorf("%s %d", i18n.T("proxy.err_socks4_code"), resp[1])
	}

	_ = conn.SetDeadline(time.Time{})
	return conn, nil
}
