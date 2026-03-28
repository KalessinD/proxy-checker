package main

import (
	"log"
	"proxy-checker/internal/cli"
	"proxy-checker/internal/config"
	"proxy-checker/internal/gui"
)

func main() {
	// Если нужно создать файл конфига при первом запуске (опционально)
	config.EnsureConfigExists()

	// Вся логика загрузки в одной строчке
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Дальше логика запуска
	if cfg.GUI {
		gui.Run(cfg)
	} else {
		cli.Run(cfg)
	}
}
