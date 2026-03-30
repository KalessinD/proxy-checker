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

	if err := common.InitLogger(cfg.LogPath); err != nil {
		log.Fatal(err)
	}
	defer zap.S().Sync()

	if cfg.Lang != "ru" && cfg.Lang != "en" {
		cfg.Lang = "ru"
	}
	if err := i18n.Init(cfg.Lang); err != nil {
		log.Fatalf("Language loading error: %v", err)
	}

	isGUI := len(os.Args) > 1 && strings.Contains(os.Args[1], "-gui")

	if isGUI {
		gui.Run(cfg)
		return
	}

	opts, err := cli.ParseFlags(cfg)
	if err != nil {
		zap.S().Fatal(err)
	}

	cli.Run(cfg, opts)
}
