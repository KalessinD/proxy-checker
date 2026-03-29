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
    if !isDebianBased() {
        return false
    }
    if !isGnomeDesktop() {
        return false
    }
    if _, err := exec.LookPath("gsettings"); err != nil {
        return false
    }
    return true
}

// isDebianBased проверяет наличие apt или /etc/debian_version
func isDebianBased() bool {
    if _, err := os.Stat("/etc/debian_version"); err == nil {
        return true
    }
    data, err := os.ReadFile("/etc/os-release")
    if err != nil {
        return false
    }
    content := string(data)
    if strings.Contains(content, "ID_LIKE=debian") ||
        strings.Contains(content, "ID=debian") ||
        strings.Contains(content, "ID=ubuntu") {
        return true
    }
    if _, err := exec.LookPath("apt"); err == nil {
        return true
    }
    return false
}

// isGnomeDesktop проверяет переменные окружения десктопа
func isGnomeDesktop() bool {
    desktop := os.Getenv("XDG_CURRENT_DESKTOP")
    session := os.Getenv("DESKTOP_SESSION")
    desktop = strings.ToLower(desktop)
    session = strings.ToLower(session)

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

// setSystemProxyMode переключает глобальный режим системного прокси (none, manual, auto)
func setSystemProxyMode(mode string) error {
    return gsettingsSet("org.gnome.system.proxy", "mode", mode)
}

// clearProxySubkey сбрасывает настройки прокси для конкретного протокола (host и port)
func clearProxySubkey(proto string) error {
    schema := fmt.Sprintf("org.gnome.system.proxy.%s", proto)
    if err := gsettingsSet(schema, "host", ""); err != nil {
        return err
    }
    // Порт обязательно сбрасываем в 0, так как gsettings ожидает число
    if err := gsettingsSet(schema, "port", "0"); err != nil {
        return err
    }
    return nil
}

// clearAllProxies очищает все настройки прокси в системе, чтобы избежать конфликтов
func clearAllProxies() error {
    // Сбрасываем все возможные типы прокси, которые использует GNOME
    for _, proto := range []string{"http", "https", "socks", "ftp"} {
        if err := clearProxySubkey(proto); err != nil {
            return fmt.Errorf("ошибка очистки прокси %s: %w", proto, err)
        }
    }
    return nil
}

// setSystemProxy применяет настройки прокси через gsettings
func setSystemProxy(host, port, proxyType string) error {
    pType := strings.ToLower(proxyType)

    // Включаем ручной режим
    if err := setSystemProxyMode("manual"); err != nil {
        return err
    }

    // СНАЧАЛА ОЧИЩАЕМ ВСЕ ТИПЫ ПРОКСИ
    // Это решает проблему, когда после SOCKS5 оставались включенными старые HTTP настройки
    if err := clearAllProxies(); err != nil {
        return err
    }

    // ПОТОМ УСТАНАВЛИВАЕМ НУЖНЫЙ
    switch pType {
    case "http", "https":
        // Обычно для HTTP прокси его применяют и для HTTP, и для HTTPS трафика
        if err := setProxySubkey("http", host, port); err != nil {
            return err
        }
        if err := setProxySubkey("https", host, port); err != nil {
            return err
        }
    case "socks4", "socks5":
        if err := setProxySubkey("socks", host, port); err != nil {
            return err
        }
    default:
        if err := setProxySubkey("http", host, port); err != nil {
            return err
        }
    }
    return nil
}

// setProxySubkey вспомогательная функция для установки конкретного протокола
func setProxySubkey(proto, host, port string) error {
    schema := fmt.Sprintf("org.gnome.system.proxy.%s", proto)
    if err := gsettingsSet(schema, "host", host); err != nil {
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

func getSystemProxyMode() (string, error) {
    cmd := exec.Command("gsettings", "get", "org.gnome.system.proxy", "mode")
    var out bytes.Buffer
    cmd.Stdout = &out

    if err := cmd.Run(); err != nil {
        return "", err
    }

    // gsettings возвращает значение в одинарных кавычках, например 'manual' или 'none'
    mode := strings.TrimSpace(out.String())
    mode = strings.Trim(mode, "'")
    return mode, nil
}
