package main

import (
	"log"
	"os"
	"strings"

	"proxy-checker/internal/cli"
	"proxy-checker/internal/common"
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
