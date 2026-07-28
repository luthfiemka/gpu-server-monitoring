package collectors

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var cgroupRe = regexp.MustCompile(
	`/docker/([0-9a-f]{64})|/docker-([0-9a-f]{64})|/containers/([0-9a-f]{64})`,
)

type ContainerInfo struct {
	ID        string
	FullID    string
	Name      string
	OwnerUser string
}

func DetectContainer(pid int) *ContainerInfo {
	path := fmt.Sprintf("/proc/%d/cgroup", pid)
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		match := cgroupRe.FindStringSubmatch(scanner.Text())
		if match != nil {
			cid := match[1]
			if cid == "" {
				cid = match[2]
			}
			if cid == "" {
				cid = match[3]
			}
			if cid != "" {
				short := cid
				if len(short) > 12 {
					short = short[:12]
				}
				return &ContainerInfo{ID: short, FullID: cid, Name: short}
			}
		}
	}
	return nil
}

type dockerInspectInfo struct {
	ID     string `json:"Id"`
	Name   string `json:"Name"`
	Config struct {
		User string `json:"User"`
	} `json:"Config"`
	HostConfig struct {
		Binds []string `json:"Binds"`
	} `json:"HostConfig"`
	Mounts []struct {
		Name        string `json:"Name"`
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
	} `json:"Mounts"`
}

var runDockerInspect = func(containerID string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "inspect", containerID)
	return cmd.Output()
}

func InspectContainer(info *ContainerInfo) *ContainerInfo {
	if info == nil {
		return nil
	}

	enriched := *info
	inspectID := info.FullID
	if inspectID == "" {
		inspectID = info.ID
	}

	raw, err := runDockerInspect(inspectID)
	if err != nil {
		return &enriched
	}

	var inspected []dockerInspectInfo
	if err := json.Unmarshal(raw, &inspected); err != nil || len(inspected) == 0 {
		return &enriched
	}

	container := inspected[0]
	if container.ID != "" {
		enriched.FullID = container.ID
		enriched.ID = container.ID
		if len(enriched.ID) > 12 {
			enriched.ID = enriched.ID[:12]
		}
	}
	if container.Name != "" {
		enriched.Name = strings.TrimPrefix(container.Name, "/")
	}
	enriched.OwnerUser = inferContainerOwner(container)
	return &enriched
}

func inferContainerOwner(container dockerInspectInfo) string {
	for _, mount := range container.Mounts {
		if user := inferUserFromPath(mount.Source); user != "" {
			return user
		}
	}

	for _, bind := range container.HostConfig.Binds {
		hostPath := bind
		if idx := strings.IndexByte(bind, ':'); idx >= 0 {
			hostPath = bind[:idx]
		}
		if user := inferUserFromPath(hostPath); user != "" {
			return user
		}
	}

	return ""
}

func inferUserFromPath(path string) string {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "." || path == string(filepath.Separator) {
		return ""
	}

	parts := strings.Split(path, string(filepath.Separator))
	for i, part := range parts {
		lower := strings.ToLower(part)
		if (lower == "home" || lower == "users") && i+1 < len(parts) {
			user := strings.TrimSpace(parts[i+1])
			if user != "" && user != "root" && !strings.HasPrefix(user, ".") {
				return user
			}
		}
	}
	return ""
}
