# Название приложения и бинаря
APP_NAME := proxy-checker
BINARY_NAME := proxy-checker
CMD_PATH := cmd/proxy-checker/main.go
BUILD_DIR := bin
INSTALL_PATH := /usr/bin/$(BINARY_NAME)
ICON_SOURCE = assets/proxy-checker.png
ICON_INSTALL_PATH = /usr/share/pixmaps/proxy-checker.png

# Путь к рабочему столу (обычно $HOME/Desktop)
DESKTOP_FILE := $(HOME)/Desktop/$(APP_NAME).desktop

.PHONY: all build install

# Цель all: сначала сборка, потом установка
all: build install

# Цель build: сборка проекта
build:
	@echo ">>> Сборка проекта..."
	# 1. Инициализация модуля (только если go.mod отсутствует)
	@if [ ! -f go.mod ]; then \
		echo "Инициализация go модуля..."; \
		go mod init $(APP_NAME); \
	fi
	# 2. Загрузка зависимостей
	@echo "Загрузка зависимостей..."
	go mod tidy
	# 3. Создание папки и сборка
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo ">>> Сборка завершена: $(BUILD_DIR)/$(BINARY_NAME)"

# Цель install: установка в систему
install:
	@echo ">>> Установка..."
	# 1. Проверяем наличие прав root для записи в /usr/bin
	@if [ ! -f $(INSTALL_PATH) ]; then \
		echo "Установка бинаря в $(INSTALL_PATH)..."; \
		sudo mkdir -p /usr/bin; \sudo install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_PATH); \
	else \
		echo "Файл $(INSTALL_PATH) уже существует. Пропуск копирования."; \
	fi
	# 2. Установка иконки в системную директорию /usr/share/pixmaps
	@if [ -f "$(ICON_SOURCE)" ]; then \
		echo "Установка иконки..."; \
		sudo install -m 644 $(ICON_SOURCE) $(ICON_INSTALL_PATH); \
	else \
		echo "Ошибка: Файл иконки $(ICON_SOURCE) не найден!"; \
	fi
	# 3. Создаем ярлык на рабочем столе
	#  $ xprop WM_CLASS # then click on window
	# WM_CLASS(STRING) = "Proxy Checker", "Proxy Checker"
	@if [ -d "$(HOME)/Desktop" ]; then \
		echo "Создание ярлыка на рабочем столе..."; \
		echo "[Desktop Entry]" > $(DESKTOP_FILE); \
		echo "Version=1.0" >> $(DESKTOP_FILE); \
		echo "Type=Application" >> $(DESKTOP_FILE); \
		echo "Name=Proxy Checker" >> $(DESKTOP_FILE); \
		echo "Comment=Proxy Checker Application" >> $(DESKTOP_FILE); \
		echo "Exec=$(INSTALL_PATH) -gui" >> $(DESKTOP_FILE); \
		echo "Icon=$(ICON_INSTALL_PATH)" >> $(DESKTOP_FILE); \
		echo "Terminal=false" >> $(DESKTOP_FILE); \
		echo "Categories=Network;Utility;" >> $(DESKTOP_FILE); \
		echo "StartupWMClass=Proxy Checker" >> $(DESKTOP_FILE); \
		chmod +x $(DESKTOP_FILE); \
		echo "Ярлык создан: $(DESKTOP_FILE)"; \
	else \
		echo "Папка Desktop не найдена, ярлык не создан."; \
	fi
	@echo ">>> Установка завершена."

run:
	@./$(BUILD_DIR)/$(BINARY_NAME) -gui