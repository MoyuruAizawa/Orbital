package docker

import (
	"context"
	"fmt"
	"limarun/internal/util"
	"strings"
)

func RunContainer(
	ctx context.Context,
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

func IsContainerRunning(ctx context.Context, name string) (bool, error) {
	output, err := util.RunCommand(ctx,
		"docker",
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

func StopContainer(ctx context.Context, name string) error {
	output, err := util.RunCommand(ctx,
		"docker",
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

func RemoveContainer(ctx context.Context, name string) error {
	output, err := util.RunCommand(ctx,
		"docker",
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

func StopAndRemoveContainer(ctx context.Context, name string) error {
	if err := StopContainer(ctx, name); err != nil {
		return fmt.Errorf("stop container %q: %w", name, err)
	}

	if err := RemoveContainer(ctx, name); err != nil {
		return fmt.Errorf("remove container %q: %w", name, err)
	}

	return nil
}
