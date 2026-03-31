package main

import (
	"log"
	"os"
	"strings"

	"proxy-checker/internal/cli"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/config"
	"proxy-checker/internal/gui"

	"go.uber.org/zap"
)

func main() {
	_ = config.EnsureConfigExists()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// Определяем режим ДО инициализации логгера
	isGUI := len(os.Args) > 1 && strings.Contains(os.Args[1], "-gui")

	// Для GUI пишем в консоль и файл, для CLI — ТОЛЬКО в файл (disableConsole = !isGUI)
	if err := common.InitLogger(cfg.LogPath, !isGUI); err != nil {
		log.Fatal(err)
	}
	defer zap.S().Sync()

	setupLanguage(cfg)

	if isGUI {
		gui.Run(cfg)
		return
	}

	opts, err := cli.ParseFlags(cfg)
	if err != nil {
		// Используем log.Fatal, так как zap в CLI模式下 консоль не пишет
		log.Fatal(err)
	}

	cli.Run(cfg, opts)
}

func setupLanguage(cfg *config.Config) {
	if cfg.Lang != "ru" && cfg.Lang != "en" {
		cfg.Lang = "en"
	}
	if err := i18n.Init(cfg.Lang); err != nil {
		log.Fatalf(i18n.T("main.err_lang_init"), err)
	}
}
