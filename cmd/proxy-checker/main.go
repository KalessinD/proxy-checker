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

	"go.uber.org/zap"
)

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

	if err := common.InitLogger(cfg.LogPath, !isGUI); err != nil {
		fatal(err)
	}
	defer func() {
		_ = zap.S().Sync()
	}()

	setupLanguage(cfg)

	if isGUI {
		gui.Run(cfg)
		return
	}

	opts, err := cli.ParseFlags(cfg)
	if err != nil {
		fatal(err)
	}

	cli.Run(cfg, opts)
}

func setupLanguage(cfg *config.Config) {
	if cfg.Lang != "ru" && cfg.Lang != "en" {
		cfg.Lang = "en"
	}
	if err := i18n.Init(cfg.Lang); err != nil {
		fatal(fmt.Errorf("%s: %w", i18n.T("main.err_lang_init"), err))
	}
}
