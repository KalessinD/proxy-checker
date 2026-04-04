package common_test

import (
	"os"
	"path/filepath"
	"proxy-checker/internal/common"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultLogPath(t *testing.T) {
	path := common.DefaultLogPath()

	if runtime.GOOS == "linux" {
		assert.Equal(t, "/tmp/"+common.AppName+".log", path)
	} else {
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)
		expectedPath := filepath.Join(homeDir, common.AppName+".log")
		assert.Equal(t, expectedPath, path)
	}
}

func TestInitLogger(t *testing.T) {
	tests := []struct {
		name           string
		logPath        string
		disableConsole bool
		expectError    bool
		errorContains  string
		setupFunc      func()
	}{
		{
			name:           "Success with console only",
			logPath:        "",
			disableConsole: false,
			expectError:    false,
		},
		{
			name:           "Success with file logging",
			logPath:        filepath.Join(t.TempDir(), "test.log"),
			disableConsole: false,
			expectError:    false,
		},
		{
			name:           "Fallback to NopCore when console disabled and no path",
			logPath:        "",
			disableConsole: true,
			expectError:    false,
		},
		{
			name:           "Error when log directory does not exist",
			logPath:        "/nonexistent/deep/dir/test.log",
			disableConsole: false,
			expectError:    true,
			errorContains:  "No access to log directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			logger, err := common.InitLogger(tt.logPath, tt.disableConsole)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, logger)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, logger)
			}

			if tt.logPath != "" && !tt.expectError {
				assert.FileExists(t, tt.logPath)
			}
		})
	}
}

func TestInitLogger_FileCreationPermissions(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "perms_test.log")

	logger, err := common.InitLogger(logFile, true)
	require.NoError(t, err)
	require.NotNil(t, logger)

	info, err := os.Stat(logFile)
	require.NoError(t, err)
	assert.False(t, info.IsDir())
}
