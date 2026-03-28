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

	// 1. Загружаем конфиг из файла
	if err := cfg.LoadFromFile(); err != nil {
		fmt.Fprintf(os.Stderr, "Предупреждение: не удалось загрузить конфиг: %v\n", err)
	}

	// 2. Парсим аргументы (включая новый -theme)
	if err := cfg.Parse(); err != nil {
		fmt.Printf("Ошибка аргументов: %v\n", err)
		os.Exit(1)
	}

	// 3. Валидация
	if err := cfg.Validate(); err != nil {
		fmt.Printf("Ошибка валидации: %v\n", err)
		os.Exit(1)
	}

	// 4. Запуск
	if cfg.GUI {
		gui.Run(cfg)
	} else {
		cli.Run(cfg)
	}
}
