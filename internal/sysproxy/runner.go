package sysproxy

import (
	"bytes"
	"os/exec"
)

// CommandRunner executes external commands and returns their captured output.
// It abstracts process execution so platform proxy managers can be tested
// with a fake implementation on any OS.
type CommandRunner interface {
	Run(name string, args ...string) (stdout string, stderr string, err error)
}

// execCommandRunner runs real binaries with os/exec.
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
