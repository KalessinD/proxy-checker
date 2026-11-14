//go:build darwin

package sysproxy

import (
	"bytes"
	"os/exec"
)

// NewSystemProxyManager returns the macOS implementation on darwin.
func NewSystemProxyManager() SystemProxyManager {
	return newNetworkSetupProxyManager(execCommandRunner{})
}

// execCommandRunner runs real binaries with os/exec. It lives in the darwin
// factory because darwin is the only platform wiring the real runner.
type execCommandRunner struct{}

func (execCommandRunner) Run(name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
