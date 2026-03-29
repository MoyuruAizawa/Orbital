package main

import (
	"context"
	"flag"
	"fmt"
	"limarun/internal/colima"
	"limarun/internal/config"
	"limarun/internal/docker"
	"limarun/internal/github"
	"os"
	"os/signal"
	"strconv"
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

	fmt.Printf("checking Colima[%s] status...\n", cfg.Colima.Profile)
	colimaRunning, err := colima.IsRunning(ctx, cfg.Colima.Profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to check Colima[%s] status: %v\n", cfg.Colima.Profile, err)
		os.Exit(1)
	}
	if colimaRunning {
		fmt.Printf("Colima[%s] is running\n", cfg.Colima.Profile)
	} else {
		fmt.Printf(ansiYellow+"WARN: Colima[%s] is not running, starting it automatically..."+ansiReset+"\n", cfg.Colima.Profile)
		if err := colima.Start(ctx, cfg.Colima.Profile); err != nil {
			fmt.Fprintf(os.Stderr, "failed to start Colima[%s]: %v\n", cfg.Colima.Profile, err)
			os.Exit(1)
		}
		fmt.Printf("Colima[%s] started\n", cfg.Colima.Profile)
	}

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

		running, err := docker.IsContainerRunning(ctx, containerName)
		if err != nil {
			return fmt.Errorf("inspect container %q: %w", containerName, err)
		}
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

		if err := docker.RemoveContainer(ctx, containerName); err != nil {
			return fmt.Errorf("remove stale container %q: %w", containerName, err)
		}

		if err := docker.RunContainer(
			ctx,
			cfg.Runner.Image,
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
		if err := docker.StopAndRemoveContainer(ctx, name); err != nil {
			return err
		}
	}

	return nil
}

func runnerContainerName(cfg config.Config, index int) string {
	return cfg.Runner.Image + "-" + strconv.Itoa(index+1)
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
