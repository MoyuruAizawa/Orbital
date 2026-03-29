package colima

import (
	"limarun/internal/util"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func Start(ctx context.Context, profile string) error {
	if _, err := util.RunStreamCommand(ctx, "colima", "start", "--profile", profile); err != nil {
		return fmt.Errorf("start colima: %w", err)
	}
	return nil
}

func IsRunning(ctx context.Context, profile string) (bool, error) {
	output, err := util.RunCommand(ctx, "colima", "status", "--profile", profile, "--json")
	if err != nil {
		trimmed := strings.TrimSpace(output)
		if strings.Contains(trimmed, "is not running") {
			return false, nil
		}
		return false, fmt.Errorf("check colima status: %w", err)
	}

	var status statusResponse
	if err := json.Unmarshal([]byte(output), &status); err != nil {
		return false, fmt.Errorf("decode colima status: %w", err)
	}

	return true, nil
}

type statusResponse struct {
	DisplayName      string `json:"display_name"`
	Driver           string `json:"driver"`
	Arch             string `json:"arch"`
	Runtime          string `json:"runtime"`
	MountType        string `json:"mount_type"`
	DockerSocket     string `json:"docker_socket"`
	ContainerdSocket string `json:"containerd_socket"`
	Kubernetes       bool   `json:"kubernetes"`
	CPU              int    `json:"cpu"`
	Memory           int64  `json:"memory"`
	Disk             int64  `json:"disk"`
}
