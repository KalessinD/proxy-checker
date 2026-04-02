package config_test

import (
	"path/filepath"
	"proxy-checker/internal/config"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig_ExpectedValues(t *testing.T) {
	cfg := config.DefaultConfig()

	assert.Equal(t, "system", cfg.Theme)
	assert.Equal(t, 600, cfg.MinHeight)
	assert.Equal(t, 800, cfg.MinWidth)
	assert.Equal(t, 10*time.Second, cfg.Timeout)
	assert.Equal(t, 256, cfg.Workers)
	assert.False(t, cfg.CheckHTTP2)
}

func TestConfig_SaveAndLoad_Lifecycle(t *testing.T) {
	// Подменяем домашнюю директорию для изоляции теста
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	originalCfg := config.DefaultConfig()
	originalCfg.Workers = 512
	originalCfg.Timeout = 30 * time.Second
	originalCfg.Lang = "ru"

	// Сохраняем
	err := originalCfg.SaveToFile()
	require.NoError(t, err)

	expectedFilePath := filepath.Join(tempHome, ".config", "proxy-checker.conf")
	assert.FileExists(t, expectedFilePath)

	// Загружаем
	loadedCfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, originalCfg.Workers, loadedCfg.Workers)
	assert.Equal(t, originalCfg.Timeout, loadedCfg.Timeout)
	assert.Equal(t, originalCfg.Lang, loadedCfg.Lang)
}
