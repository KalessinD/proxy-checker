package proxies

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/net/proxy"
)

// NewClient создает HTTP клиент (без изменений в логике, но DialContext теперь использует контекст)
func NewClient(proxyAddr, mode string) (*http.Client, error) {
    var transport *http.Transport
    switch mode {
    case "socks4":
        transport = &http.Transport{
            DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
                // Передаем контекст в диалер
                return socks4Dial(ctx, network, addr, proxyAddr)
            },
        }

    case "socks5":
        // Для SOCKS5 тоже лучше использовать контекст, но стандартный proxy.SOCKS5 
        // не поддерживает DialContext напрямую. 
        // Для полноценной поддержки контекста в SOCKS5 нужно писать свой диалер или использовать forwarding dialer.
        // Здесь оставим стандартную реализацию для простоты, так как основной фокус на SOCKS4.
        dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
        if err != nil {
            return nil, fmt.Errorf("ошибка инициализации SOCKS5: %w", err)
        }
        transport = &http.Transport{
            Dial: dialer.Dial,
        }

    case "https":
        proxyURL, _ := url.Parse("https://" + proxyAddr)
        transport = &http.Transport{
            Proxy: http.ProxyURL(proxyURL),
        }

    default: // "http"
        proxyURL, _ := url.Parse("http://" + proxyAddr)
        transport = &http.Transport{
            Proxy: http.ProxyURL(proxyURL),
        }
    }

    // Внимание: http.Client.Timeout — это общий таймаут на весь запрос.
    // Если мы хотим управлять таймаутом через контекст в сервисе, 
    // этот Timeout можно не выставлять или выставить чуть больше.
    return &http.Client{
        Transport: transport,
    }, nil
}

// socks4Dial реализует протокол SOCKS4 с поддержкой контекста
func socks4Dial(ctx context.Context, network, addr, proxyAddr string) (net.Conn, error) {
    // 1. Создаем Dialer. 
    // Timeout здесь можно не указывать, так как DialContext использует дедлайн из контекста.
    d := net.Dialer{}

    // 2. Используем метод DialContext структуры Dialer!
    conn, err := d.DialContext(ctx, network, proxyAddr)
    if err != nil {
        return nil, err
    }

    // 3. Если в контексте есть дедлайн, устанавливаем его на соединение.
    // Это гарантирует, что запись/чтение (handshake) не зависнут.
    if deadline, ok := ctx.Deadline(); ok {
        if err := conn.SetDeadline(deadline); err != nil {
            conn.Close()
            return nil, err
        }
    }

    // ... далее код парсинга адреса и протокола без изменений ...
    
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
            return nil, fmt.Errorf("SOCKS4: не удалось резолвить домен %s", host)
        }
        ip = ips[0]
    }

    ipv4 := ip.To4()
    if ipv4 == nil {
        conn.Close()
        return nil, errors.New("SOCKS4 поддерживает только IPv4")
    }

    req := make([]byte, 9)
    req[0] = 4
    req[1] = 1
    req[2] = byte(port >> 8)
    req[3] = byte(port)
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
        return nil, fmt.Errorf("SOCKS4 прокси вернул ошибку: код %d", resp[1])
    }

    // Сбрасываем дедлайн, чтобы дальнейшая передача данных не была ограничена временем handshake
    conn.SetDeadline(time.Time{})

    return conn, nil
}
