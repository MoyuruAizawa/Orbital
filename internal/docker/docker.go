package docker

import (
	"context"
	"fmt"
	"limarun/internal/util"
	"strings"
)

func CheckConnection(ctx context.Context, dockerContext string) error {
	_, err := util.RunCommand(ctx,
		"docker",
		"--context", dockerContext,
		"info",
	)
	if err != nil {
		return err
	}

	return nil
}

func RunContainer(
	ctx context.Context,
	dockerContext string,
	image string,
	containerName string,
	githubUrl string,
	runnerToken string,
	runnerName string,
	runnerGroup string,
	runnerLabels []string,
	mntSrcPath string,
	mntDestPath string,
) error {
	_, err := util.RunStreamCommand(ctx,
		"docker",
		"--context", dockerContext,
		"run",
		"-d",
		"--name", containerName,
		"-e", "GITHUB_URL="+githubUrl,
		"-e", "RUNNER_NAME="+runnerName,
		"-e", "RUNNER_GROUP="+runnerGroup,
		"-e", "RUNNER_LABELS="+strings.Join(runnerLabels, ","),
		"-e", "RUNNER_TOKEN="+runnerToken,
		"-v", mntSrcPath+":"+mntDestPath,
		image,
	)
	return err
}

func IsContainerRunning(ctx context.Context, dockerContext string, name string) (bool, error) {
	output, err := util.RunCommand(ctx,
		"docker",
		"--context", dockerContext,
		"inspect",
		"-f", "{{.State.Running}}",
		name,
	)
	if err != nil {
		if strings.Contains(output, "No such object") {
			return false, nil
		}
		return false, err
	}

	return strings.TrimSpace(output) == "true", nil
}

func StopContainer(ctx context.Context, dockerContext string, name string) error {
	output, err := util.RunCommand(ctx,
		"docker",
		"--context", dockerContext,
		"stop",
		name,
	)
	if err != nil {
		if strings.Contains(output, "No such container") {
			return nil
		}
		return err
	}

	return nil
}

func RemoveContainer(ctx context.Context, dockerContext string, name string) error {
	output, err := util.RunCommand(ctx,
		"docker",
		"--context", dockerContext,
		"rm",
		"-f",
		name,
	)
	if err != nil {
		if strings.Contains(output, "No such container") {
			return nil
		}
		return err
	}

	return nil
}

func StopAndRemoveContainer(ctx context.Context, dockerContext string, name string) error {
	if err := StopContainer(ctx, dockerContext, name); err != nil {
		return fmt.Errorf("stop container %q: %w", name, err)
	}

	if err := RemoveContainer(ctx, dockerContext, name); err != nil {
		return fmt.Errorf("remove container %q: %w", name, err)
	}

	return nil
}
