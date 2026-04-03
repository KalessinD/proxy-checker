//go:build linux

package sysproxy_test

import (
	"os"
	"proxy-checker/internal/sysproxy"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			// Сохраняем и восстанавливаем env
			oldDesktop := os.Getenv("XDG_CURRENT_DESKTOP")
			oldSession := os.Getenv("DESKTOP_SESSION")
			defer func() {
				t.Setenv("XDG_CURRENT_DESKTOP", oldDesktop)
				t.Setenv("DESKTOP_SESSION", oldSession)
			}()

			t.Setenv("XDG_CURRENT_DESKTOP", tt.desktopEnv)
			t.Setenv("DESKTOP_SESSION", tt.sessionEnv)
			result := sysproxy.IsGnomeDesktop()

			assert.Equal(t, tt.expectMatch, result, "Not expected result")
		})
	}
}
