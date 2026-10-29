package sysproxy

import (
	"errors"
	"proxy-checker/internal/common/i18n"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRunner records commands and returns canned responses keyed by
// the full command line ("name arg1 arg2 ...").
type fakeRunner struct {
	responses map[string]string
	failures  map[string]error
	calls     [][]string
}

func (f *fakeRunner) Run(name string, args ...string) (string, string, error) {
	call := append([]string{name}, args...)
	f.calls = append(f.calls, call)

	key := strings.Join(call, " ")
	if err, ok := f.failures[key]; ok {
		return "", "", err
	}
	return f.responses[key], "", nil
}

const (
	routeDefaultCmd      = "route -n get default"
	listHardwarePortsCmd = "networksetup -listallhardwareports"
	networkSetupBin      = "networksetup"
	serviceWiFi          = "Wi-Fi"
	socksDisabledOutput  = "Enabled: No\n"
)

func (f *fakeRunner) networkSetupCalls() [][]string {
	calls := make([][]string, 0, len(f.calls))
	for _, call := range f.calls {
		if call[0] == networkSetupBin {
			calls = append(calls, call)
		}
	}
	return calls
}

func newFakeManager(t *testing.T, responses map[string]string, failures map[string]error) (*networkSetupProxyManager, *fakeRunner) {
	runner := &fakeRunner{responses: responses, failures: failures}
	manager, ok := newNetworkSetupProxyManager(runner).(*networkSetupProxyManager)
	require.True(t, ok, "Factory must return the networkSetupProxyManager")
	return manager, runner
}

const (
	routeOutputEn0   = "   route to: default\ndestination: default\n   interface: en0\n"
	hardwarePortsOut = "Hardware Port: Wi-Fi\nDevice: en0\nEthernet Address: aa:bb:cc:dd:ee:ff\n"
)

func TestNewNetworkSetupProxyManager_IsSupported(t *testing.T) {
	_ = i18n.Init("en")
	manager, _ := newFakeManager(t, nil, nil)
	assert.Equal(t, runtime.GOOS == "darwin", manager.IsSupported())
}

func TestNetworkSetupManager_GetActiveNetworkService(t *testing.T) {
	_ = i18n.Init("en")

	t.Run("Resolves service from route and hardware ports", func(t *testing.T) {
		manager, runner := newFakeManager(t, map[string]string{
			routeDefaultCmd:      routeOutputEn0,
			listHardwarePortsCmd: hardwarePortsOut,
		}, nil)

		service, err := manager.getActiveNetworkService()
		require.NoError(t, err)
		assert.Equal(t, serviceWiFi, service)
		require.Len(t, runner.calls, 2, "Route and hardware ports queries expected")
	})

	t.Run("Route command failure", func(t *testing.T) {
		manager, _ := newFakeManager(t, nil, map[string]error{
			routeDefaultCmd: errors.New("route not found"),
		})

		_, err := manager.getActiveNetworkService()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get default route")
	})

	t.Run("Route output without interface", func(t *testing.T) {
		//nolint:gosec // false positive: test fixture output, not credentials
		manager, _ := newFakeManager(t, map[string]string{
			routeDefaultCmd: "destination: default\n",
		}, nil)

		_, err := manager.getActiveNetworkService()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse default interface")
	})

	t.Run("Hardware ports command failure", func(t *testing.T) {
		//nolint:gosec // false positive: test fixture output, not credentials
		manager, _ := newFakeManager(t, map[string]string{
			routeDefaultCmd: routeOutputEn0,
		}, map[string]error{
			listHardwarePortsCmd: errors.New("boom"),
		})

		_, err := manager.getActiveNetworkService()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list hardware ports")
	})

	t.Run("Unknown interface", func(t *testing.T) {
		//nolint:gosec // false positive: test fixture output, not credentials
		manager, _ := newFakeManager(t, map[string]string{
			routeDefaultCmd:      "   interface: en9\n",
			listHardwarePortsCmd: hardwarePortsOut,
		}, nil)

		_, err := manager.getActiveNetworkService()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "network service not found for interface: en9")
	})
}

func TestNetworkSetupManager_SetNetworkSetup_ArgumentOrder(t *testing.T) {
	_ = i18n.Init("en")
	manager, runner := newFakeManager(t, nil, nil)

	require.NoError(t, manager.setNetworkSetup(serviceWiFi, "-setwebproxy", "1.2.3.4", "8080"))

	calls := runner.networkSetupCalls()
	require.Len(t, calls, 1, "Single networksetup call expected")
	// The command flag must come first, the service must follow it.
	assert.Equal(t, []string{networkSetupBin, "-setwebproxy", serviceWiFi, "1.2.3.4", "8080"}, calls[0])
}

func TestNetworkSetupManager_SetNetworkSetup_EmptyArgs(t *testing.T) {
	_ = i18n.Init("en")
	manager, runner := newFakeManager(t, nil, nil)

	err := manager.setNetworkSetup(serviceWiFi)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "command flag is required")
	assert.Empty(t, runner.networkSetupCalls(), "No command must be executed without a flag")
}

func TestNetworkSetupManager_SetProxy_SOCKS5(t *testing.T) {
	_ = i18n.Init("en")
	manager, runner := newFakeManager(t, map[string]string{
		routeDefaultCmd:      routeOutputEn0,
		listHardwarePortsCmd: hardwarePortsOut,
	}, nil)

	require.NoError(t, manager.SetProxy("5.6.7.8", "1080", "socks5"))

	calls := runner.networkSetupCalls()
	require.Len(t, calls, 5, "1 discovery + 2 reset + 2 socks calls expected")
	assert.Equal(t, []string{networkSetupBin, "-setwebproxystate", serviceWiFi, "off"}, calls[1])
	assert.Equal(t, []string{networkSetupBin, "-setsocksfirewallproxystate", serviceWiFi, "off"}, calls[2])
	assert.Equal(t, []string{networkSetupBin, "-setsocksfirewallproxy", serviceWiFi, "5.6.7.8", "1080"}, calls[3])
	assert.Equal(t, []string{networkSetupBin, "-setsocksfirewallproxystate", serviceWiFi, "on"}, calls[4])
}

func TestNetworkSetupManager_GetMode(t *testing.T) {
	_ = i18n.Init("en")

	testCases := []struct {
		name         string
		webOutput    string
		socksOutput  string
		expectedMode string
	}{
		{
			name:         "Web proxy enabled",
			webOutput:    "Enabled: Yes\nServer: 1.1.1.1\nPort: 8080\n",
			socksOutput:  socksDisabledOutput,
			expectedMode: ProxyModeManual,
		},
		{
			name:         "SOCKS proxy enabled",
			webOutput:    socksDisabledOutput,
			socksOutput:  "Enabled: Yes\nServer: 2.2.2.2\nPort: 1080\n",
			expectedMode: ProxyModeManual,
		},
		{
			name:         "No proxy enabled",
			webOutput:    socksDisabledOutput,
			socksOutput:  socksDisabledOutput,
			expectedMode: ProxyModeNone,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			//nolint:gosec // false positive: test fixture output, not credentials
			manager, _ := newFakeManager(t, map[string]string{
				routeDefaultCmd:                             routeOutputEn0,
				listHardwarePortsCmd:                        hardwarePortsOut,
				"networksetup -getwebproxy Wi-Fi":           tt.webOutput,
				"networksetup -getsocksfirewallproxy Wi-Fi": tt.socksOutput,
			}, nil)

			mode, err := manager.GetMode()
			require.NoError(t, err)
			assert.Equal(t, tt.expectedMode, mode)
		})
	}
}

func TestNetworkSetupManager_IgnoreHosts(t *testing.T) {
	_ = i18n.Init("en")

	t.Run("Get returns trimmed domains", func(t *testing.T) {
		//nolint:gosec // false positive: test fixture output, not credentials
		manager, _ := newFakeManager(t, map[string]string{
			routeDefaultCmd:      routeOutputEn0,
			listHardwarePortsCmd: hardwarePortsOut,
			"networksetup -getproxybypassdomains Wi-Fi": " *.local\n169.254/16 \n", //nolint:gosec // false positive: test fixture output, not credentials
		}, nil)

		hosts, err := manager.GetIgnoreHosts()
		require.NoError(t, err)
		assert.Equal(t, "*.local\n169.254/16", hosts)
	})

	t.Run("Get returns empty when no bypass domains set", func(t *testing.T) {
		//nolint:gosec // false positive: test fixture output, not credentials
		manager, _ := newFakeManager(t, map[string]string{
			routeDefaultCmd:      routeOutputEn0,
			listHardwarePortsCmd: hardwarePortsOut,
			"networksetup -getproxybypassdomains Wi-Fi": "There aren't any bypass domains set\n",
		}, nil)

		hosts, err := manager.GetIgnoreHosts()
		require.NoError(t, err)
		assert.Empty(t, hosts)
	})

	t.Run("Set filters empty lines and uses Empty placeholder", func(t *testing.T) {
		manager, runner := newFakeManager(t, map[string]string{
			routeDefaultCmd:      routeOutputEn0,
			listHardwarePortsCmd: hardwarePortsOut,
		}, nil)

		require.NoError(t, manager.SetIgnoreHosts(" *.local \n\n 169.254/16\n"))

		calls := runner.networkSetupCalls()
		require.Len(t, calls, 2, "1 discovery + 1 set call expected")
		assert.Equal(t, []string{networkSetupBin, "-setproxybypassdomains", serviceWiFi, "*.local", "169.254/16"}, calls[1])
	})

	t.Run("Set with blank text uses Empty placeholder", func(t *testing.T) {
		manager, runner := newFakeManager(t, map[string]string{
			routeDefaultCmd:      routeOutputEn0,
			listHardwarePortsCmd: hardwarePortsOut,
		}, nil)

		require.NoError(t, manager.SetIgnoreHosts("\n\n"))

		calls := runner.networkSetupCalls()
		require.Len(t, calls, 2)
		assert.Equal(t, []string{networkSetupBin, "-setproxybypassdomains", serviceWiFi, "Empty"}, calls[1])
	})
}
