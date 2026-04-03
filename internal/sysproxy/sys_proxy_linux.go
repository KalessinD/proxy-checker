//go:build linux

package sysproxy

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"proxy-checker/internal/common/i18n"
	"strings"
)

type linuxProxyManager struct{}

func newLinuxProxyManager() SystemProxyManager {
	m := &linuxProxyManager{}
	return m
}

func (m *linuxProxyManager) IsSupported() bool {
	return isDebianBased() && isGnomeDesktop() && isGsettingsAvailable()
}

func (m *linuxProxyManager) GetMode() (string, error) {
	cmd := exec.Command("gsettings", "get", "org.gnome.system.proxy", "mode")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", err
	}

	mode := strings.TrimSpace(out.String())
	mode = strings.Trim(mode, "'")
	return mode, nil
}

func (m *linuxProxyManager) SetMode(mode string) error {
	return gsettingsSet("org.gnome.system.proxy", "mode", mode)
}

func (m *linuxProxyManager) SetProxy(host, port, proxyType string) error {
	pType := strings.ToLower(proxyType)

	if err := m.SetMode(ProxyModeManual); err != nil {
		return err
	}

	if err := clearAllProxies(); err != nil {
		return err
	}

	switch pType {
	case "http", "https":
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

func (m *linuxProxyManager) GetIgnoreHosts() (string, error) {
	cmd := exec.Command("gsettings", "get", "org.gnome.system.proxy", "ignore-hosts")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T("sysproxy.err_get_ignore_hosts"), err)
	}

	rawList := ParseGVariantStringArray(out.String())
	return strings.Join(rawList, "\n"), nil
}

func (m *linuxProxyManager) SetIgnoreHosts(ignoreHostsText string) error {
	lines := strings.Split(ignoreHostsText, "\n")

	var cleanedHosts []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanedHosts = append(cleanedHosts, line)
		}
	}

	gVariantArray := FormatGVariantStringArray(cleanedHosts)

	if err := gsettingsSet("org.gnome.system.proxy", "ignore-hosts", gVariantArray); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("sysproxy.err_set_ignore_hosts"), err)
	}

	return nil
}

func isGsettingsAvailable() bool {
	_, err := exec.LookPath("gsettings")
	return err == nil
}

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

func isGnomeDesktop() bool {
	desktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	session := strings.ToLower(os.Getenv("DESKTOP_SESSION"))

	return strings.Contains(desktop, "gnome") ||
		strings.Contains(desktop, "unity") ||
		strings.Contains(desktop, "cinnamon") ||
		strings.Contains(session, "gnome") ||
		strings.Contains(session, "ubuntu") ||
		strings.Contains(session, "cinnamon")
}

func clearAllProxies() error {
	for _, proto := range []string{"http", "https", "socks", "ftp"} {
		if err := clearProxySubkey(proto); err != nil {
			return fmt.Errorf("%s %s: %w", i18n.T("sysproxy.err_clear"), proto, err)
		}
	}
	return nil
}

func clearProxySubkey(proto string) error {
	schema := "org.gnome.system.proxy." + proto
	if err := gsettingsSet(schema, "host", ""); err != nil {
		return err
	}
	return gsettingsSet(schema, "port", "0")
}

func setProxySubkey(proto, host, port string) error {
	schema := "org.gnome.system.proxy." + proto
	if err := gsettingsSet(schema, "host", host); err != nil {
		return err
	}
	return gsettingsSet(schema, "port", port)
}

func gsettingsSet(schema, key, value string) error {
	cmd := exec.Command("gsettings", "set", schema, key, value)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %v, %s", i18n.T("sysproxy.err_gsettings"), err, stderr.String())
	}
	return nil
}
