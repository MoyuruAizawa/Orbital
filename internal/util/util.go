package util

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type CommandError struct {
	Command string
	Output  string
	Err     error
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("command %q failed: %v: %s", e.Command, e.Err, strings.TrimSpace(e.Output))
}

func RunStreamCommand(ctx context.Context, name string, args ...string) (string, error) {
	return command(ctx, true, name, args...)
}

func RunCommand(ctx context.Context, name string, args ...string) (string, error) {
	return command(ctx, false, name, args...)
}

func command(ctx context.Context, streamOutput bool, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	if streamOutput {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return "", &CommandError{Command: strings.Join(append([]string{name}, args...), " "), Err: err}
		}
		return "", nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), &CommandError{Command: strings.Join(append([]string{name}, args...), " "), Output: string(output), Err: err}
	}
	return string(output), nil
}
