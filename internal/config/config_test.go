package config_test

import (
	"os"
	"path/filepath"
	"proxy-checker/internal/config"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		assert.Contains(t, string(data), "lang = 'en'", "Файл не должен перезаписываться, если уже существует")
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
	require.Error(t, err, "Должна возвращаться ошибка парсинга TOML")
	assert.Nil(t, loadedCfg)
}

func TestSaveToFile_PermissionDenied(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	configDir := filepath.Join(tempHome, ".config")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	require.NoError(t, os.Chmod(configDir, 0o555))
	defer func() { _ = os.Chmod(configDir, 0o755) }()

	cfg := config.DefaultConfig()
	err := cfg.SaveToFile()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}
