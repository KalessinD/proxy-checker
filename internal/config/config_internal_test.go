package config

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestNormalizeGeoIPPath(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("path layout test is only meaningful on darwin and linux")
	}

	volumeRoot := string(filepath.Separator)
	homeRoot := "home"
	if runtime.GOOS == "darwin" {
		homeRoot = "Users"
	}
	homeDir := filepath.Join(volumeRoot, homeRoot, "tester")
	t.Setenv("HOME", homeDir)

	// The damaged shape: the leading slash was lost when the path was saved.
	damaged := filepath.Join(homeRoot, "tester", "Library", "proxy.mmdb")

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty path stays empty", in: "", want: ""},
		{name: "absolute path unchanged", in: filepath.Join(volumeRoot, "opt", "db.mmdb"), want: filepath.Clean(filepath.Join(volumeRoot, "opt", "db.mmdb"))},
		{name: "tilde expanded", in: "~/data/db.mmdb", want: filepath.Join(homeDir, "data", "db.mmdb")},
		{name: "lost leading slash restored under home root", in: damaged, want: volumeRoot + damaged},
		{name: "unknown relative path only cleaned", in: filepath.Join("var", "db.mmdb"), want: filepath.Join("var", "db.mmdb")},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := &Config{GeoIPDBPath: testCase.in}

			normalizeGeoIPPath(cfg)

			if cfg.GeoIPDBPath != testCase.want {
				t.Errorf("normalizeGeoIPPath() = %q, want %q", cfg.GeoIPDBPath, testCase.want)
			}
		})
	}
}
