package collectors

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
)

var cgroupRe = regexp.MustCompile(
	`/docker/([0-9a-f]{64})|/docker-([0-9a-f]{64})|/containers/([0-9a-f]{64})`,
)

type ContainerInfo struct {
	ID   string
	Name string
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
				return &ContainerInfo{ID: short, Name: short}
			}
		}
	}
	return nil
}
