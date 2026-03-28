//go:build linux
// +build linux

package gui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// isSystemProxySupported проверяет, является ли ОС Debian-совместимой
// и запущен ли GNOME-совместимый десктоп.
func isSystemProxySupported() bool {
	// 1. Проверка на Debian совместимость (Debian, Ubuntu, Mint и т.д.)
	if !isDebianBased() {
		return false
	}

	// 2. Проверка на GNOME совместимый десктоп
	if !isGnomeDesktop() {
		return false
	}

	// 3. Проверка наличия утилиты gsettings
	if _, err := exec.LookPath("gsettings"); err != nil {
		return false
	}

	return true
}

// isDebianBased проверяет наличие apt или /etc/debian_version
func isDebianBased() bool {
	// Простой способ: проверить наличие /etc/debian_version или ID_LIKE=debian в os-release
	if _, err := os.Stat("/etc/debian_version"); err == nil {
		return true
	}

	// Читаем /etc/os-release
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return false
	}

	// Ищем ID_LIKE=debian или ID=ubuntu/debian
	content := string(data)
	if strings.Contains(content, "ID_LIKE=debian") ||
		strings.Contains(content, "ID=debian") ||
		strings.Contains(content, "ID=ubuntu") {
		return true
	}

	// Проверяем наличие apt (косвенный признак)
	if _, err := exec.LookPath("apt"); err == nil {
		return true
	}

	return false
}

// isGnomeDesktop проверяет переменные окружения десктопа
func isGnomeDesktop() bool {
	// Обычно это XDG_CURRENT_DESKTOP или DESKTOP_SESSION
	desktop := os.Getenv("XDG_CURRENT_DESKTOP")
	session := os.Getenv("DESKTOP_SESSION")

	desktop = strings.ToLower(desktop)
	session = strings.ToLower(session)

	// Список совместимых сред: GNOME, Ubuntu:GNOME, Cinnamon, Unity
	if strings.Contains(desktop, "gnome") ||
		strings.Contains(desktop, "unity") ||
		strings.Contains(desktop, "cinnamon") ||
		strings.Contains(session, "gnome") ||
		strings.Contains(session, "ubuntu") ||
		strings.Contains(session, "cinnamon") {
		return true
	}

	return false
}

// setSystemProxy применяет настройки прокси через gsettings
func setSystemProxy(host, port, proxyType string) error {
	schema := "org.gnome.system.proxy"

	// Приводим тип к нижнему регистру для проверки
	pType := strings.ToLower(proxyType)

	// 1. Сначала сбрасываем настройки других типов, чтобы избежать конфликтов
	// (Опционально, но безопаснее)
	// resetProxySettings()

	// 2. Включаем ручной режим
	if err := gsettingsSet(schema, "mode", "'manual'"); err != nil {
		return err
	}

	// 3. Устанавливаем хост и порт в зависимости от типа
	switch pType {
	case "http", "https":
		// Для HTTP/HTTPS ставим в http и https схемы
		if err := setProxySubkey("http", host, port); err != nil {
			return err
		}
		if err := setProxySubkey("https", host, port); err != nil {
			return err
		}
		// Обычно SSL прокси используют ту же схему https
		// gsettings set org.gnome.system.proxy.https host ...

	case "socks4":
		if err := setProxySubkey("socks", host, port); err != nil {
			return err
		}
	case "socks5":
		// host = "socks5://" + host
		if err := setProxySubkey("socks", host, port); err != nil {
			return err
		}
	default:
		// Если тип all или неизвестен, ставим в HTTP по умолчанию
		if err := setProxySubkey("http", host, port); err != nil {
			return err
		}
	}

	return nil
}

// setProxySubkey вспомогательная функция для установки конкретного протокола
func setProxySubkey(proto, host, port string) error {
	schema := fmt.Sprintf("org.gnome.system.proxy.%s", proto)
	if err := gsettingsSet(schema, "host", fmt.Sprintf("'%s'", host)); err != nil {
		return err
	}
	if err := gsettingsSet(schema, "port", port); err != nil {
		return err
	}
	return nil
}

// gsettingsSet выполняет команду gsettings set
func gsettingsSet(schema, key, value string) error {
	cmd := exec.Command("gsettings", "set", schema, key, value)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gsettings error: %v, %s", err, stderr.String())
	}
	return nil
}
