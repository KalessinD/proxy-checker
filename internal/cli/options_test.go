package cli_test

import (
	"os"
	"proxy-checker/internal/cli"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/config"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	_ = i18n.Init("en")
	os.Exit(m.Run())
}

func TestParseFlags(t *testing.T) {
	baseConfig := func() *config.Config {
		return &config.Config{
			Type:       common.ProxySOCKS5,
			Source:     common.SourceProxyMania,
			Timeout:    10 * time.Second,
			Workers:    512,
			RTT:        150,
			DestAddr:   "google.com",
			CheckHTTP2: false,
		}
	}

	tests := []struct {
		name            string
		args            []string
		wantErr         bool
		expectedErrText string
		assertOptions   func(t *testing.T, opts *cli.Options, cfg *config.Config)
	}{
		{
			name:    "Valid single proxy check",
			args:    []string{"-proxy", "192.168.1.1:8080"},
			wantErr: false,
			assertOptions: func(t *testing.T, opts *cli.Options, _ *config.Config) {
				assert.Equal(t, "192.168.1.1:8080", opts.ProxyAddr)
				assert.False(t, opts.ProxiesStat)
			},
		},
		{
			name:    "Valid batch proxy stat",
			args:    []string{"-proxies-stat", "-check"},
			wantErr: false,
			assertOptions: func(t *testing.T, opts *cli.Options, _ *config.Config) {
				assert.True(t, opts.ProxiesStat)
				assert.True(t, opts.Check)
			},
		},
		{
			name:            "Error on mutually exclusive flags",
			args:            []string{"-proxy", "1.1.1.1:80", "-proxies-stat"},
			wantErr:         true,
			expectedErrText: "simultaneously",
		},
		{
			name:            "Error on invalid proxy address format",
			args:            []string{"-proxy", "invalid-host-no-port"},
			wantErr:         true,
			expectedErrText: "Invalid proxy address format",
		},
		{
			name:            "Error on zero RTT in stat mode",
			args:            []string{"-proxies-stat", "-rtt", "0"},
			wantErr:         true,
			expectedErrText: "RTT must be greater than 0",
		},
		{
			name:            "Error on negative RTT in stat mode",
			args:            []string{"-proxies-stat", "-rtt", "-50"},
			wantErr:         true,
			expectedErrText: "RTT must be greater than 0",
		},
		{
			name:            "Error on workers less than 1",
			args:            []string{"-proxies-stat", "-workers", "0"},
			wantErr:         true,
			expectedErrText: "Workers must be at least 1",
		},
		{
			name:            "Error on workers greater than 512",
			args:            []string{"-proxies-stat", "-workers", "520"},
			wantErr:         true,
			expectedErrText: "Workers must be no more than 512",
		},
		{
			name:            "Error on invalid proxy type",
			args:            []string{"-proxy", "1.1.1.1:80", "-type", "ftp"},
			wantErr:         true,
			expectedErrText: "Invalid proxy type: ftp",
		},
		{
			name:            "Error on invalid source",
			args:            []string{"-proxies-stat", "-source", "yandex"},
			wantErr:         true,
			expectedErrText: "Invalid source: yandex",
		},
		{
			name:    "Config values overridden by flags",
			args:    []string{"-proxy", "1.1.1.1:80", "-timeout", "5s", "-workers", "10", "-type", "http"},
			wantErr: false,
			assertOptions: func(t *testing.T, _ *cli.Options, cfg *config.Config) {
				assert.Equal(t, 5*time.Second, cfg.Timeout, "Timeout should be overridden")
				assert.Equal(t, 10, cfg.Workers, "Workers should be overridden")
				assert.Equal(t, common.ProxyHTTP, cfg.Type, "Type should be overridden")
			},
		},
		{
			name:    "Config values remain default if flags not provided",
			args:    []string{"-proxy", "1.1.1.1:80"},
			wantErr: false,
			assertOptions: func(t *testing.T, _ *cli.Options, cfg *config.Config) {
				assert.Equal(t, 10*time.Second, cfg.Timeout)
				assert.Equal(t, 512, cfg.Workers)
				assert.Equal(t, common.ProxySOCKS5, cfg.Type)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseConfig()

			opts, err := cli.ParseFlags(cfg, tt.args)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrText)
			} else {
				require.NoError(t, err)
				require.NotNil(t, opts)

				if tt.assertOptions != nil {
					tt.assertOptions(t, opts, cfg)
				}
			}
		})
	}
}
