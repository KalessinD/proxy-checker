APP_NAME := proxy-checker
BINARY_NAME := proxy-checker
CMD_PATH := cmd/proxy-checker/main.go
BUILD_DIR := bin

INSTALL_PATH := /usr/bin/$(BINARY_NAME)
ICON_SOURCE := assets/proxy-checker.png
ICON_INSTALL_PATH := /usr/share/pixmaps/proxy-checker.png
DESKTOP_FILE := $(HOME)/Desktop/$(APP_NAME).desktop

OS := $(shell uname -s)

ifeq ($(OS),Linux)
    OSTYPE := linux
else ifeq ($(OS),Darwin)
    OSTYPE := macos
else ifeq ($(OS),FreeBSD)
    OSTYPE := freebsd
else ifneq (,$(findstring MINGW,$(OS))$(findstring MSYS,$(OS))$(findstring CYGWIN,$(OS)))
    OSTYPE := windows
else
    OSTYPE := unknown
endif

ifeq ($(OSTYPE),windows)
    BINARY_FULL := $(BUILD_DIR)/$(BINARY_NAME).exe
else
    BINARY_FULL := $(BUILD_DIR)/$(BINARY_NAME)
endif

.PHONY: all build install uninstall run clean \
        install-linux install-linux-bin install-linux-desktop-shortcut \
        install-windows install-macos install-freebsd install-unknown \
        uninstall-linux uninstall-windows uninstall-macos uninstall-freebsd uninstall-unknown

all: build

build:
	@echo ">>> Building project for $(OSTYPE)..."
	@echo ">>> Downloading dependencies..."
	@go mod tidy
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BINARY_FULL) $(CMD_PATH)
	@echo ">>> Successfully built: $(BINARY_FULL)"

install: install-$(OSTYPE)

uninstall: uninstall-$(OSTYPE)

install-linux: install-linux-bin install-linux-desktop-shortcut
	@echo ">>> Linux installation completed successfully."

install-linux-bin:
	@echo ">>> Installing binary to $(INSTALL_PATH)..."
	@if [ -f "$(INSTALL_PATH)" ]; then \
		echo ">>> File $(INSTALL_PATH) already exists. It will be overwritten."; \
	fi
	@sudo mkdir -p /usr/bin
	@sudo install -m 755 $(BINARY_FULL) $(INSTALL_PATH)

install-linux-desktop-shortcut:
	@if [ -f "$(ICON_SOURCE)" ]; then \
		echo ">>> Installing application icon..."; \
		sudo install -m 644 $(ICON_SOURCE) $(ICON_INSTALL_PATH); \
	else \
		echo ">>> Warning: Icon $(ICON_SOURCE) not found, skipping shortcut creation."; \
		exit 0; \
	fi
	@if [ -d "$(HOME)/Desktop" ]; then \
		echo ">>> Creating desktop shortcut..."; \
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
		echo ">>> Shortcut created: $(DESKTOP_FILE)"; \
	else \
		echo ">>> Desktop folder not found, skipping shortcut creation."; \
	fi

uninstall-linux:
	@echo ">>> Uninstalling application..."
	@sudo rm -f $(INSTALL_PATH)
	@sudo rm -f $(ICON_INSTALL_PATH)
	@rm -f $(DESKTOP_FILE)
	@echo ">>> Application uninstalled."

install-windows:
	@echo ">>> Error: Automated installation is not supported for Windows."
	@echo ">>> Please use the built binary directly: $(BINARY_FULL)"

install-macos:
	@echo ">>> Error: Automated installation is not supported for macOS."
	@echo ">>> Please use the built binary directly: $(BINARY_FULL)"

install-freebsd:
	@echo ">>> Error: Automated installation is not supported for FreeBSD."
	@echo ">>> Please use the built binary directly: $(BINARY_FULL)"

install-unknown:
	@echo ">>> Error: Cannot detect your operating system."
	@echo ">>> Please use the built binary directly: $(BINARY_FULL)"

uninstall-windows uninstall-macos uninstall-freebsd uninstall-unknown:
	@echo ">>> Error: Automated uninstallation is not supported for $(OSTYPE)."

run: build
	@echo ">>> Running application..."
	@$(BINARY_FULL) -gui

clean:
	@echo ">>> Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo ">>> Clean completed."
