package gitstatus

import (
	"os/exec"
	"strings"
)

// ExecuteFetch runs git fetch.
func ExecuteFetch(workDir string) (string, error) {
	cmd := exec.Command("git", "fetch")
	cmd.Dir = workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", &RemoteError{Output: string(output), Err: err}
	}
	return string(output), nil
}

// ExecutePull runs git pull.
func ExecutePull(workDir string) (string, error) {
	cmd := exec.Command("git", "pull")
	cmd.Dir = workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", &RemoteError{Output: string(output), Err: err}
	}
	return string(output), nil
}

// RemoteError wraps a git remote operation error with its output.
type RemoteError struct {
	Output string
	Err    error
}

func (e *RemoteError) Error() string {
	return strings.TrimSpace(e.Output)
}
