package main

import (
	"context"
	"flag"
	"fmt"
	"orbital/internal/config"
	"orbital/internal/docker"
	"orbital/internal/github"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	ansiYellow = "\033[33m"
	ansiReset  = "\033[0m"
)

func main() {
	configPath, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		printUsage()
		os.Exit(1)
	}

	fmt.Println("loading Configuration...")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("configuration loaded")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Printf("checking Docker context[%s] connectivity...\n", cfg.Docker.Context)
	if err := docker.CheckConnection(ctx, cfg.Docker.Context); err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to Docker context[%s]: %v\n", cfg.Docker.Context, err)
		fmt.Fprintf(os.Stderr, ansiYellow+"hint: ensure the target Docker Engine is already available for the configured context"+ansiReset+"\n")
		os.Exit(1)
	}
	fmt.Printf("Docker context[%s] is reachable\n", cfg.Docker.Context)

	fmt.Printf("building runner image[%s] from source image[%s]...\n", cfg.Docker.RunnerImageName, cfg.Docker.SourceImage)
	if err := docker.BuildRunnerImage(ctx, cfg.Docker.Context, cfg.Docker.SourceImage, cfg.Docker.RunnerImageName); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build runner image[%s]: %v\n", cfg.Docker.RunnerImageName, err)
		os.Exit(1)
	}
	fmt.Printf("runner image[%s] built\n", cfg.Docker.RunnerImageName)

	if err := ensureRunners(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to ensure runners: %v\n", err)
	}

	ticker := time.NewTicker(time.Duration(cfg.Runtime.PollIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("shutdown signal received")

			if err := cleanupContainers(context.Background(), cfg); err != nil {
				fmt.Fprintf(os.Stderr, "failed to cleanup containers: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("containers stopped and removed")
			return
		case <-ticker.C:
			if err := ensureRunners(ctx, cfg); err != nil {
				fmt.Fprintf(os.Stderr, "failed to ensure runners: %v\n", err)
			}
		}
	}
}

func ensureRunners(ctx context.Context, cfg config.Config) error {
	for i := 0; i < cfg.Runner.Count; i++ {
		containerName := runnerContainerName(cfg, i)

		running := docker.IsContainerRunning(ctx, cfg.Docker.Context, containerName)
		if running {
			continue
		}

		fmt.Printf("starting container[%s]...\n", containerName)
		token, err := github.GenerateRunnerRegistrationToken(
			ctx,
			cfg.Github.Org,
			cfg.Github.AppID,
			cfg.Github.InstallationID,
			cfg.Github.PEMPath,
		)
		if err != nil {
			return fmt.Errorf("generate runner registration token for %q: %w", containerName, err)
		}

		if err := docker.RemoveContainer(ctx, cfg.Docker.Context, containerName); err != nil {
			return fmt.Errorf("remove stale container %q: %w", containerName, err)
		}

		if err := docker.RunContainer(
			ctx,
			cfg.Docker.Context,
			cfg.Docker.RunnerImageName,
			containerName,
			cfg.Github.Url(),
			token,
			runnerName(cfg, i),
			cfg.Runner.Group,
			cfg.Runner.Labels,
			cfg.Mount.Source,
			cfg.Mount.Target); err != nil {
			return fmt.Errorf("run container %q: %w", containerName, err)
		}

		fmt.Printf("container[%s] started\n", containerName)
	}

	return nil
}

func cleanupContainers(ctx context.Context, cfg config.Config) error {
	for i := 0; i < cfg.Runner.Count; i++ {
		name := runnerContainerName(cfg, i)
		fmt.Printf("cleaning up container[%s]...\n", name)
		if err := docker.StopAndRemoveContainer(ctx, cfg.Docker.Context, name); err != nil {
			return err
		}
	}

	return nil
}

func runnerContainerName(cfg config.Config, index int) string {
	return sanitizeContainerName(cfg.Docker.RunnerImageName) + "-" + strconv.Itoa(index+1)
}

func sanitizeContainerName(name string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		":", "-",
		"@", "-",
	)
	sanitized := replacer.Replace(strings.ToLower(name))
	sanitized = strings.Trim(sanitized, "-._")
	if sanitized == "" {
		return "runner"
	}
	return sanitized
}

func runnerName(cfg config.Config, index int) string {
	return cfg.Runner.NamePrefix + "-" + strconv.Itoa(index+1)
}

func parseArgs(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("missing subcommand")
	}

	switch args[0] {
	case "start":
		startFlags := flag.NewFlagSet("start", flag.ContinueOnError)
		startFlags.SetOutput(os.Stderr)

		configPath := startFlags.String("c", "", "path to config file")
		if err := startFlags.Parse(args[1:]); err != nil {
			return "", err
		}
		if *configPath == "" {
			return "", fmt.Errorf("missing required flag: --c")
		}
		if startFlags.NArg() > 0 {
			return "", fmt.Errorf("unexpected arguments: %v", startFlags.Args())
		}

		return *configPath, nil
	default:
		return "", fmt.Errorf("unknown subcommand: %s", args[0])
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "usage: %s start --c <config.yml>\n", os.Args[0])
}
