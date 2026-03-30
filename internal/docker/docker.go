package docker

import (
	"context"
	"fmt"
	"orbital/internal/util"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type runnerBuildTemplateData struct {
	SourceImage      string
	RunnerTargetOS   string
	RunnerTargetArch string
}

type sourceImagePlatform struct {
	OS   string
	Arch string
}

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

func BuildRunnerImage(
	ctx context.Context,
	dockerContext string,
	sourceImage string,
	runnerImageName string,
) error {
	platform, err := InspectImagePlatform(ctx, dockerContext, sourceImage)
	if err != nil {
		return fmt.Errorf("inspect source image platform %q: %w", sourceImage, err)
	}

	buildDir, err := os.MkdirTemp("", "orbital-runner-build-*")
	if err != nil {
		return fmt.Errorf("create temp build dir: %w", err)
	}
	defer os.RemoveAll(buildDir)

	tmpl, err := template.New("Dockerfile").Parse(RunnerDockerfileTemplate)
	if err != nil {
		return fmt.Errorf("parse Dockerfile template: %w", err)
	}

	dockerfilePath := filepath.Join(buildDir, "Dockerfile")
	dockerfile, err := os.Create(dockerfilePath)
	if err != nil {
		return fmt.Errorf("create Dockerfile: %w", err)
	}

	if err := tmpl.Execute(dockerfile, runnerBuildTemplateData{
		SourceImage:      sourceImage,
		RunnerTargetOS:   platform.OS,
		RunnerTargetArch: platform.Arch,
	}); err != nil {
		dockerfile.Close()
		return fmt.Errorf("render Dockerfile: %w", err)
	}

	if err := dockerfile.Close(); err != nil {
		return fmt.Errorf("close Dockerfile: %w", err)
	}

	entrypointPath := filepath.Join(buildDir, "entrypoint.sh")
	if err := os.WriteFile(entrypointPath, []byte(RunnerEntrypoint), 0o755); err != nil {
		return fmt.Errorf("write entrypoint: %w", err)
	}

	_, err = util.RunStreamCommand(ctx,
		"docker",
		"--context", dockerContext,
		"build",
		"--build-arg", "RUNNER_TARGET_OS="+platform.OS,
		"--build-arg", "RUNNER_TARGET_ARCH="+platform.Arch,
		"-t", runnerImageName,
		buildDir,
	)
	if err != nil {
		return err
	}

	return nil
}

func InspectImagePlatform(ctx context.Context, dockerContext string, image string) (sourceImagePlatform, error) {
	output, err := util.RunCommand(ctx,
		"docker",
		"--context", dockerContext,
		"image",
		"inspect",
		image,
		"--format",
		"{{.Os}}/{{.Architecture}}",
	)
	if err != nil {
		return sourceImagePlatform{}, err
	}

	platform := strings.TrimSpace(output)
	parts := strings.Split(platform, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return sourceImagePlatform{}, fmt.Errorf("unexpected image platform format: %q", platform)
	}

	return sourceImagePlatform{OS: parts[0], Arch: parts[1]}, nil
}

func RunContainer(
	ctx context.Context,
	dockerContext string,
	runnerImageName string,
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
		runnerImageName,
	)
	return err
}

func IsContainerRunning(ctx context.Context, dockerContext string, name string) bool {
	output, err := util.RunCommand(ctx,
		"docker",
		"--context", dockerContext,
		"inspect",
		"-f", "{{.State.Running}}",
		name,
	)
	if err != nil {
		return false
	}

	return strings.TrimSpace(output) == "true"
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
