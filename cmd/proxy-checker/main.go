package main

import (
	"fmt"
	"os"

	"proxy-checker/internal/cli"
	"proxy-checker/internal/config"
	"proxy-checker/internal/gui"
)

func main() {
	cfg := &config.Config{}

	if err := config.EnsureConfigExists(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка создания конфига: %v\n", err)
	}

	if err := cfg.LoadFromFile(); err != nil {
		fmt.Fprintf(os.Stderr, "Предупреждение: не удалось загрузить конфиг: %v\n", err)
	}

	if err := cfg.Parse(); err != nil {
		fmt.Printf("Ошибка аргументов: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		fmt.Printf("Ошибка валидации: %v\n", err)
		os.Exit(1)
	}

	if cfg.GUI {
		gui.Run(cfg)
	} else {
		cli.Run(cfg)
	}
}
