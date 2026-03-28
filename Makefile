APP_NAME := proxy-checker
BINARY_NAME := proxy-checker
CMD_PATH := cmd/proxy-checker/main.go
BUILD_DIR := bin
INSTALL_PATH := /usr/bin/$(BINARY_NAME)
ICON_SOURCE = assets/proxy-checker.png
ICON_INSTALL_PATH = /usr/share/pixmaps/proxy-checker.png

DESKTOP_FILE := $(HOME)/Desktop/$(APP_NAME).desktop

.PHONY: all build install

all: build install

build:
	@echo ">>> Сборка проекта..."
	@if [ ! -f go.mod ]; then \
		echo "Initializing Go module..."; \
		go mod init $(APP_NAME); \
	fi
	@echo "Downloading the dependenices..."
	go mod tidy
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo ">>> Built successfully!": $(BUILD_DIR)/$(BINARY_NAME)"

install:
	@echo ">>> Installing..."
	@if [ -f $(INSTALL_PATH) ]; then \
		echo "The file $(INSTALL_PATH) is already exists. Will be overwritten"; \
	fi
	echo "Installing binary into $(INSTALL_PATH)...";
	sudo mkdir -p /usr/bin;
	sudo install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_PATH);
	@if [ -f "$(ICON_SOURCE)" ]; then \
		echo "Installing the app icon..."; \
		sudo install -m 644 $(ICON_SOURCE) $(ICON_INSTALL_PATH); \
	else \
		echo "Error: the icon file $(ICON_SOURCE) wasn't found!"; \
	fi
#  	$ xprop WM_CLASS # then click on window
# 	WM_CLASS(STRING) = "Proxy Checker", "Proxy Checker"
	@if [ -d "$(HOME)/Desktop" ]; then \
		echo "Creating app link at the desktop..."; \
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
		echo "The link has been created: $(DESKTOP_FILE)"; \
	else \
		echo "The Desktop folder wasn't found."; \
	fi
	@echo ">>> Installed successfully."

run:
	@./$(BUILD_DIR)/$(BINARY_NAME) -gui