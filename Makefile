SHELL := /bin/bash
PROJECT_DIR ?= $(CURDIR)
TMPDIR ?= /tmp

APP_NAME := proxy-checker
BINARY_NAME := proxy-checker
CMD_PATH := cmd/proxy-checker/main.go
BUILD_DIR := bin

INSTALL_PATH := /usr/bin/$(BINARY_NAME)
ICON_SOURCE := assets/proxy-checker.png
ICON_INSTALL_PATH := /usr/share/pixmaps/proxy-checker.png
DESKTOP_FILE := $(HOME)/Desktop/$(APP_NAME).desktop

GOLANGCI_LINT ?= golangci-lint

CHMOD := chmod
INSTALL := install
SUDO := sudo
GO ?= go
GREP := grep
RM := rm -f
RMDIR := rm -rf
MKDIR := mkdir
CD := cd
ECHO := echo -e
NOECHO := @

OS := $(shell uname -s)

# print_title = $(ECHO) "\033[1;34m$1\033[0m"
print_info = $(ECHO) "\033[1;36m$1\033[0m"
print_warn = $(ECHO) "\033[1;33m$1\033[0m"
print_error = $(ECHO) "\033[1;31m$1\033[0m"
print_success = $(ECHO) "\033[1;32m$1\033[0m"

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

.PHONY: all build install uninstall run clean help \
        install-linux install-linux-bin install-linux-desktop-shortcut \
        install-windows install-macos install-freebsd install-unknown \
        uninstall-linux uninstall-windows uninstall-macos uninstall-freebsd uninstall-unknown \
        lint lint-vet lint-golangci lint-golangci-fix

all: clean build

help: # Shows help message
	$(NOECHO) $(GREP) -E '^[a-zA-Z0-9 -]+:.*#' Makefile | \
	sort | \
	while read -r l; do \
		printf "\033[1;36m$$(echo $$l | cut -f 1 -d':')\033[00m:$$(echo $$l | cut -f 2- -d'#')\n"; \
	done

build: # Builds app binary
	$(NOECHO) $(call print_info,Building project for $(OSTYPE)...)
	$(NOECHO) $(call print_info,Downloading dependencies...)
	$(NOECHO) $(GO) mod tidy
	$(NOECHO) $(MKDIR) -p $(BUILD_DIR)
	$(NOECHO) $(GO) build -o $(BINARY_FULL) $(CMD_PATH)
	$(NOECHO) $(call print_success,Successfully built: $(BINARY_FULL))

install: install-$(OSTYPE) # Installs the app

uninstall: uninstall-$(OSTYPE) # Uninstall the app

install-linux: install-linux-bin install-linux-desktop-shortcut # The app installation on Linux
	$(NOECHO) $(call print_info,Linux installation completed successfully.)

install-linux-bin: # The app binary installation on Linux
	$(NOECHO) $(call print_info,Installing binary to $(INSTALL_PATH)...)
	$(NOECHO) if [ -f "$(INSTALL_PATH)" ]; then \
		$(call print_info,File $(INSTALL_PATH) already exists. It will be overwritten.); \
	fi
	$(NOECHO) $(SUDO) $(MKDIR) -p /usr/bin
	$(NOECHO) $(SUDO) $(INSTALL) -m 755 $(BINARY_FULL) $(INSTALL_PATH)

install-linux-desktop-shortcut: # The app desktop shortcut installation on Linux
	$(NOECHO) if [ -f "$(ICON_SOURCE)" ]; then \
		$(call print_info,Installing application icon...); \
		$(SUDO) $(INSTALL) -m 644 $(ICON_SOURCE) $(ICON_INSTALL_PATH); \
	else \
		$(call print_warn,Warning: Icon $(ICON_SOURCE) not found, skipping shortcut creation); \
		exit 0; \
	fi
	$(NOECHO) if [ -d "$(HOME)/Desktop" ]; then \
		$(call print_info,Creating desktop shortcut...); \
		$(ECHO) "[Desktop Entry]" > $(DESKTOP_FILE); \
		$(ECHO) "Version=1.0" >> $(DESKTOP_FILE); \
		$(ECHO) "Type=Application" >> $(DESKTOP_FILE); \
		$(ECHO) "Name=Proxy Checker" >> $(DESKTOP_FILE); \
		$(ECHO) "Comment=Proxy Checker Application" >> $(DESKTOP_FILE); \
		$(ECHO) "Exec=$(INSTALL_PATH) -gui" >> $(DESKTOP_FILE); \
		$(ECHO) "Icon=$(ICON_INSTALL_PATH)" >> $(DESKTOP_FILE); \
		$(ECHO) "Terminal=false" >> $(DESKTOP_FILE); \
		$(ECHO) "Categories=Network;Utility;" >> $(DESKTOP_FILE); \
		$(ECHO) "StartupWMClass=Proxy Checker" >> $(DESKTOP_FILE); \
		$(CHMOD) +x $(DESKTOP_FILE); \
		$(call print_success,Shortcut created: $(DESKTOP_FILE)); \
	else \
		$(call print_warn,Desktop folder not found, skipping shortcut creation.); \
	fi

uninstall-linux: # The app uninstallation on Linux
	$(NOECHO) $(call print_info,Uninstalling application...)
	$(NOECHO) $(SUDO) $(RM) $(INSTALL_PATH)
	$(NOECHO) $(SUDO) $(RM) $(ICON_INSTALL_PATH)
	$(NOECHO) $(RM) $(DESKTOP_FILE)
	$(NOECHO) $(call print_success,Application uninstalled.)

install-windows: # The app installation on Windows
	$(NOECHO) $(call print_error,Error: Automated installation is not supported for Windows.)
	$(NOECHO) $(call print_error,Please use the built binary directly: $(BINARY_FULL))

install-macos: # The app installation on MacOS
	$(NOECHO) $(call print_error,Error: Automated installation is not supported for macOS.)
	$(NOECHO) $(call print_error,Please use the built binary directly: $(BINARY_FULL))

install-freebsd: # The app installation on FreeBSD
	$(NOECHO) $(call print_error,Error: Automated installation is not supported for FreeBSD.)
	$(NOECHO) $(call print_error,Please use the built binary directly: $(BINARY_FULL))

install-unknown: # The app installation on other OS
	$(NOECHO) $(call print_error,Error: Cannot detect your operating system.)
	$(NOECHO) $(call print_error,Please use the built binary directly: $(BINARY_FULL))

uninstall-windows uninstall-macos uninstall-freebsd uninstall-unknown: # The app uninstallation on different OS
	$(NOECHO) $(call print_error,Error: Automated uninstallation is not supported for $(OSTYPE))

run: build # Runs the built app
	$(NOECHO) $(call print_info,Running application...)
	@$(BINARY_FULL) -gui

clean: # Removes binaries and logs
	$(NOECHO) $(call print_info,Cleaning build artifacts...)
	$(NOECHO) $(RMDIR) $(BUILD_DIR)
	$(NOECHO) $(call print_success,Clean completed.)

lint: lint-vet lint-golangci # Runs linters

lint-vet: # Runs go vet
	$(NOECHO) $(call print_info,Running go vet with structtag check)
	$(NOECHO) $(GO) vet -structtag ./...

lint-golangci: # Runs golangci-lint
	$(NOECHO) $(call print_info,Running golangci linters)
	$(NOECHO) $(GOLANGCI_LINT) run

lint-golangci-fix: # Runs golangci-lint with auto-fix
	$(NOECHO) $(call print_info,Running golangci linters in fix mode)
	$(NOECHO) $(GOLANGCI_LINT) run --fix
