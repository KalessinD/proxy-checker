package common

import (
	"fmt"
	"os"
	"path/filepath"
	"proxy-checker/internal/common/i18n"
	"runtime"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LoggerInterface interface {
	Info(fields ...any)
	Error(fields ...any)
	Warn(fields ...any)
	Debug(fields ...any)

	Infof(msg string, fields ...any)
	Errorf(msg string, fields ...any)
	Warnf(msg string, fields ...any)
	Debugf(msg string, fields ...any)

	Sync() error
}

type ZapLogger struct {
	*zap.SugaredLogger
}

func NewZapLogger(logger *zap.SugaredLogger) LoggerInterface {
	return &ZapLogger{SugaredLogger: logger}
}

func InitLogger(logPath string, disableConsole bool) (LoggerInterface, error) {
	pe := zap.NewProductionEncoderConfig()
	pe.TimeKey = "time"
	pe.EncodeTime = zapcore.ISO8601TimeEncoder

	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(pe),
		zapcore.AddSync(os.Stdout),
		zap.InfoLevel,
	)

	var cores []zapcore.Core

	if !disableConsole {
		cores = append(cores, consoleCore)
	}

	if logPath != "" {
		dir := filepath.Dir(logPath)

		_, err := os.Stat(dir)
		if err != nil {
			return nil, fmt.Errorf("%s %s", i18n.T("log.warn_dir_access"), dir)
		}

		file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("%s (%s): %w", i18n.T("log.warn_open_file"), logPath, err)
		}

		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(pe),
			zapcore.AddSync(file),
			zap.InfoLevel,
		)
		cores = append(cores, fileCore)
	}

	if len(cores) == 0 {
		cores = append(cores, zapcore.NewNopCore())
	}

	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return NewZapLogger(logger.Sugar()), nil
}

func DefaultLogPath() string {
	if runtime.GOOS == "linux" {
		return "/tmp/" + AppName + ".log"
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return AppName + ".log"
	}
	return filepath.Join(home, AppName+".log")
}
