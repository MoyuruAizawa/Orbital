package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Docker  DockerConfig  `yaml:"docker"`
	Github  GithubConfig  `yaml:"github"`
	Runner  RunnerConfig  `yaml:"runner"`
	Mount   MountConfig   `yaml:"mount"`
	Runtime RuntimeConfig `yaml:"runtime"`
}

type DockerConfig struct {
	Context string `yaml:"context"`
	Image   string `yaml:"image"`
}

type GithubConfig struct {
	Org            string `yaml:"org"`
	AppID          int64  `yaml:"appId"`
	InstallationID int64  `yaml:"installationId"`
	PEMPath        string `yaml:"pem"`
}

func (c *GithubConfig) Url() string {
	return fmt.Sprintf("https://github.com/%s", c.Org)
}

type RunnerConfig struct {
	Group      string   `yaml:"group"`
	Labels     []string `yaml:"labels"`
	NamePrefix string   `yaml:"namePrefix"`
	Count      int      `yaml:"count"`
}

type MountConfig struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

type RuntimeConfig struct {
	PollIntervalSeconds int `yaml:"pollIntervalSeconds"`
}

func LoadConfig(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read file %q: %w", path, err)
	}

	var config Config
	if err := yaml.Unmarshal(content, &config); err != nil {
		return Config{}, fmt.Errorf("parse yaml: %w", err)
	}

	if err := validateConfig(config); err != nil {
		return Config{}, err
	}

	return config, nil
}

func validateConfig(config Config) error {
	if strings.TrimSpace(config.Docker.Context) == "" {
		return fmt.Errorf("docker.context is required")
	}
	if strings.TrimSpace(config.Docker.Image) == "" {
		return fmt.Errorf("docker.image is required")
	}
	if strings.TrimSpace(config.Github.Org) == "" {
		return fmt.Errorf("github.org is required")
	}
	if config.Github.AppID <= 0 {
		return fmt.Errorf("github.appId must be greater than 0")
	}
	if config.Github.InstallationID <= 0 {
		return fmt.Errorf("github.installationId must be greater than 0")
	}
	if strings.TrimSpace(config.Github.PEMPath) == "" {
		return fmt.Errorf("github.pem is required")
	}
	if strings.TrimSpace(config.Runner.NamePrefix) == "" {
		return fmt.Errorf("runner.namePrefix is required")
	}
	if config.Runner.Count <= 0 {
		return fmt.Errorf("runner.count must be greater than 0")
	}
	if strings.TrimSpace(config.Mount.Source) == "" {
		return fmt.Errorf("mount.source is required")
	}
	if strings.TrimSpace(config.Mount.Target) == "" {
		return fmt.Errorf("mount.target is required")
	}
	if config.Runtime.PollIntervalSeconds <= 0 {
		return fmt.Errorf("runtime.pollIntervalSeconds must be greater than 0")
	}

	return nil
}
