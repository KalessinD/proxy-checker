package cli_test

import (
	"io"
	"os"
	"proxy-checker/internal/cli"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/fetcher"
	"proxy-checker/internal/services"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = i18n.Init("en")
}

func TestPrintTable_FormatsOutput(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	items := []*fetcher.ProxyItem{
		{Host: "1.1.1.1", Port: "8080", Type: "socks5", Country: "US", RTT: "50 ms"},
	}

	cli.PrintTable(items)

	w.Close()
	os.Stdout = old

	data, err := io.ReadAll(r)
	require.NoError(t, err)
	output := string(data)

	assert.Contains(t, output, "1.1.1.1")
	assert.Contains(t, output, "8080")
	assert.Contains(t, output, "socks5")
	assert.Contains(t, output, "US")
	assert.Contains(t, output, "50 ms")
}

func TestPrintFullTable_FormatsOutput(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	items := []*services.ProxyItemFull{
		{
			ProxyItem:   services.ProxyItem{Host: "2.2.2.2", Port: "3128", Type: "http", Country: "GB"},
			CheckResult: services.Result{ProxyLatencyStr: "10ms", ReqLatencyStr: "20ms"},
		},
	}

	cli.PrintFullTable(items)

	w.Close()
	os.Stdout = old

	data, _ := io.ReadAll(r)
	output := string(data)

	assert.Contains(t, output, "2.2.2.2")
	assert.Contains(t, output, "10ms")
	assert.Contains(t, output, "20ms")
	assert.Contains(t, output, "GB")
}

func TestPrintTable_EmptyList(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cli.PrintTable([]*fetcher.ProxyItem{})

	w.Close()
	os.Stdout = old

	data, _ := io.ReadAll(r)
	output := string(data)

	assert.Contains(t, output, "Host")
	assert.NotContains(t, output, "1.1.1.1")
}
