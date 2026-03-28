package main

import (
	"log"
	"os"
	"strings"

	"proxy-checker/internal/cli"
	"proxy-checker/internal/config"
	"proxy-checker/internal/gui"
)

func main() {
	config.EnsureConfigExists()

	// 1. Определяем, нужен ли GUI (быстрая проверка без полного парсинга)
	isGUI := len(os.Args) > 1 && strings.Contains(os.Args[1], "-gui")

	if isGUI {
		// GUI не нуждается в парсинге флагов, ему нужен только файл конфигурации
		cfg, err := config.Load()
		if err != nil {
			log.Fatal(err)
		}
		gui.Run(cfg)
		return
	}

	// 2. Если это CLI - загружаем конфиг и парсим флаги
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	opts, err := cli.ParseFlags(cfg) // Передаем cfg, чтобы флаги могли его переопределить
	if err != nil {
		log.Fatal(err)
	}

	cli.Run(cfg, opts)
}
