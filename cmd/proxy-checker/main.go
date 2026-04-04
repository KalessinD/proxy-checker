package main

import (
	"fmt"
	"os"
	"proxy-checker/internal/cli"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/config"
	"proxy-checker/internal/gui"
	"strings"
)

// Version is injected at build time using -ldflags="-X main.Version=$(APP_VERSION)"
var Version = "dev"

func fatal(err error) {
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
	}
	os.Exit(1)
}

func main() {
	_ = config.EnsureConfigExists()

	cfg, err := config.Load()
	if err != nil {
		fatal(err)
	}

	isGUI := len(os.Args) > 1 && strings.Contains(os.Args[1], "-gui")

	logger, err := common.InitLogger(cfg.LogPath, !isGUI)
	if err != nil {
		fatal(err)
	}

	defer func() { _ = logger.Sync() }()

	setupLanguage(cfg)

	if isGUI {
		gui.Run(cfg, logger, Version)
		return
	}

	opts, err := cli.ParseFlags(cfg, os.Args[1:])
	if err != nil {
		fatal(err)
	}

	cli.Run(cfg, opts, logger)
}

func setupLanguage(cfg *config.Config) {
	if cfg.Lang != "ru" && cfg.Lang != "en" {
		cfg.Lang = "en"
	}
	if err := i18n.Init(cfg.Lang); err != nil {
		fatal(fmt.Errorf("%s: %w", i18n.T("main.err_lang_init"), err))
	}
}
