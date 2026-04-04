package config_test

import (
	"os"
	"path/filepath"
	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig_Values(t *testing.T) {
	cfg := config.DefaultConfig()

	assert.Equal(t, common.ProxySOCKS5, cfg.Type)
	assert.Equal(t, common.SourceProxyMania, cfg.Source)
	assert.Equal(t, 10*time.Second, cfg.Timeout)
	assert.Equal(t, 256, cfg.Workers)
	assert.Equal(t, "google.com", cfg.DestAddr)
	assert.Equal(t, "en", cfg.Lang)
	assert.Equal(t, 150, cfg.RTT)
	assert.Equal(t, 4, cfg.Pages)
}

func TestSaveToFile_Success(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	cfg := config.DefaultConfig()
	cfg.Lang = "ru"
	cfg.Workers = 128

	err := cfg.SaveToFile()
	require.NoError(t, err)

	configPath := filepath.Join(tempHome, ".config", "proxy-checker.conf")
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	assert.Contains(t, string(data), "lang = \"ru\"")
	assert.Contains(t, string(data), "workers = 128")
}

func TestEnsureConfigExists(t *testing.T) {
	t.Run("Creates config if missing", func(t *testing.T) {
		tempHome := t.TempDir()
		t.Setenv("HOME", tempHome)

		err := config.EnsureConfigExists()
		require.NoError(t, err)

		expectedPath := filepath.Join(tempHome, ".config", "proxy-checker.conf")
		assert.FileExists(t, expectedPath)
	})

	t.Run("Does nothing if config already exists", func(t *testing.T) {
		tempHome := t.TempDir()
		t.Setenv("HOME", tempHome)

		configDir := filepath.Join(tempHome, ".config")
		require.NoError(t, os.MkdirAll(configDir, 0o755))
		configPath := filepath.Join(configDir, "proxy-checker.conf")
		require.NoError(t, os.WriteFile(configPath, []byte("lang = 'en'"), 0o600))

		err := config.EnsureConfigExists()
		require.NoError(t, err)

		data, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "lang = 'en'", "File must not be overwritten if it already exists")
	})
}

func TestLoad_CorruptedTOML(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	configDir := filepath.Join(tempHome, ".config")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	configPath := filepath.Join(configDir, "proxy-checker.conf")

	require.NoError(t, os.WriteFile(configPath, []byte("[[[invalid toml syntax"), 0o600))

	loadedCfg, err := config.Load()
	require.Error(t, err, "Must return TOML parsing error")
	assert.Nil(t, loadedCfg)
}
