package i18n_test

import (
	"proxy-checker/internal/common/i18n"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit_Success(t *testing.T) {
	err := i18n.Init("en")
	require.NoError(t, err)

	err = i18n.Init("ru")
	require.NoError(t, err)
}

func TestInit_FallbackToEnglish(t *testing.T) {
	err := i18n.Init("nonexistent_lang")
	require.NoError(t, err)

	translatedText := i18n.T("gui.btn_settings")
	assert.Equal(t, "Settings", translatedText)
}

func TestT_ExistingKey(t *testing.T) {
	err := i18n.Init("ru")
	require.NoError(t, err)

	translatedText := i18n.T("gui.btn_settings")
	assert.Equal(t, "Настройки", translatedText)
}

func TestT_MissingKey(t *testing.T) {
	err := i18n.Init("ru")
	require.NoError(t, err)

	missingKey := "gui.completely.missing.key"
	assert.Equal(t, missingKey, i18n.T(missingKey))
}

func TestAvailableLanguages(t *testing.T) {
	langs := i18n.AvailableLanguages()

	assert.NotEmpty(t, langs, "Language list must not be empty")
	assert.Contains(t, langs, "en", "Must contain English language")
	assert.Contains(t, langs, "ru", "Must contain Russian language")
}
