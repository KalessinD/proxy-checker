//go:build linux

package sysproxy_test

import (
	"os"
	"proxy-checker/internal/sysproxy"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSystemProxyManager_ReturnsValidInterface_linux(t *testing.T) {
	manager := sysproxy.NewSystemProxyManager()
	require.NotNil(t, manager, "Factory always must return the interface")
}

func TestNewSystemProxyManager_PlatformConsistency_linux(t *testing.T) {
	manager := sysproxy.NewSystemProxyManager()
	assert.IsNotType(t, &sysproxy.NoOpProxyManager{}, manager, "Must use gnomeProxyManager on linux")
}

func TestIsGnomeDesktop(t *testing.T) {
	tests := []struct {
		name        string
		desktopEnv  string
		sessionEnv  string
		expectMatch bool
	}{
		{
			name:        "GNOME desktop",
			desktopEnv:  "GNOME",
			sessionEnv:  "",
			expectMatch: true,
		},
		{
			name:        "Ubuntu session",
			desktopEnv:  "",
			sessionEnv:  "ubuntu",
			expectMatch: true,
		},
		{
			name:        "KDE Plasma",
			desktopEnv:  "KDE",
			sessionEnv:  "",
			expectMatch: false,
		},
		{
			name:        "Cinnamon desktop",
			desktopEnv:  "X-Cinnamon",
			sessionEnv:  "",
			expectMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment variables
			oldDesktop := os.Getenv("XDG_CURRENT_DESKTOP")
			oldSession := os.Getenv("DESKTOP_SESSION")
			defer func() {
				t.Setenv("XDG_CURRENT_DESKTOP", oldDesktop)
				t.Setenv("DESKTOP_SESSION", oldSession)
			}()

			t.Setenv("XDG_CURRENT_DESKTOP", tt.desktopEnv)
			t.Setenv("DESKTOP_SESSION", tt.sessionEnv)
			result := sysproxy.IsGnomeDesktop()

			assert.Equal(t, tt.expectMatch, result, "Result does not match expectation")
		})
	}
}

func TestIsDebianBased_OsReleaseFile(t *testing.T) {
	assert.NotPanics(t, func() {
		sysproxy.IsDebianBased()
	})
}

func TestParseGVariantStringArray(t *testing.T) {
	tests := []struct {
		name         string
		rawInput     string
		expectedList []string
	}{
		{name: "Empty array", rawInput: "[]", expectedList: nil},
		{name: "Single element", rawInput: "['localhost']", expectedList: []string{"localhost"}},
		{name: "Multiple elements sorted", rawInput: "['localhost', '127.0.0.0/8', '::1']", expectedList: []string{"127.0.0.0/8", "::1", "localhost"}},
		{name: "Elements without quotes", rawInput: "[localhost, 192.168.1.1]", expectedList: []string{"192.168.1.1", "localhost"}},
		{name: "Empty strings inside", rawInput: "['', 'test']", expectedList: []string{"test"}},
		{name: "Trailing comma", rawInput: "['localhost', ]", expectedList: []string{"localhost"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sysproxy.ParseGVariantStringArray(tt.rawInput)
			assert.Equal(t, tt.expectedList, result)
		})
	}
}

func TestFormatGVariantStringArray(t *testing.T) {
	tests := []struct {
		name           string
		inputHosts     []string
		expectedString string
	}{
		{name: "Nil slice", inputHosts: nil, expectedString: "[]"},
		{name: "Empty slice", inputHosts: []string{}, expectedString: "[]"},
		{name: "Single element", inputHosts: []string{"localhost"}, expectedString: "['localhost']"},
		{name: "Sorted and quoted", inputHosts: []string{"192.168.1.1", "localhost", "::1"}, expectedString: "['192.168.1.1', '::1', 'localhost']"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sysproxy.FormatGVariantStringArray(tt.inputHosts)
			assert.Equal(t, tt.expectedString, result)
		})
	}
}

func TestGVariantStringArray_RoundTrip(t *testing.T) {
	hosts := []string{"localhost", "192.168.1.1", "::1"}

	parsed := sysproxy.ParseGVariantStringArray(sysproxy.FormatGVariantStringArray(hosts))
	assert.Equal(t, []string{"192.168.1.1", "::1", "localhost"}, parsed, "Parse(Format(hosts)) must return the same sorted hosts")
}
