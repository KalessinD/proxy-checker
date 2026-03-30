package common

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func InitLogger(logPath string) error {
	pe := zap.NewProductionEncoderConfig()
	pe.TimeKey = "time"
	pe.EncodeTime = zapcore.ISO8601TimeEncoder

	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(pe),
		zapcore.AddSync(os.Stdout),
		zap.InfoLevel,
	)

	cores := []zapcore.Core{consoleCore}

	if logPath != "" {
		dir := filepath.Dir(logPath)
		if _, err := os.Stat(dir); err == nil {
			file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				fileCore := zapcore.NewCore(
					zapcore.NewJSONEncoder(pe),
					zapcore.AddSync(file),
					zap.InfoLevel,
				)
				cores = append(cores, fileCore)
			} else {
				fmt.Printf("[WARN] Не удалось открыть файл логов %s: %v\n", logPath, err)
			}
		} else {
			fmt.Printf("[WARN] Нет доступа к директории логов %s. Логирование только в консоль.\n", dir)
		}
	}

	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	zap.ReplaceGlobals(logger)

	return nil
}

func DefaultLogPath() string {
	if runtime.GOOS == "linux" {
		return "/var/log/" + AppName + ".log"
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return AppName + ".log"
	}
	return filepath.Join(home, AppName+".log")
}
