package sysproxy

// CommandRunner executes external commands and returns their captured output.
// It abstracts process execution so platform proxy managers can be tested
// with a fake implementation on any OS.
type CommandRunner interface {
	Run(name string, args ...string) (stdout string, stderr string, err error)
}
