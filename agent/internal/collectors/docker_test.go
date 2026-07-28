package collectors

import (
	"errors"
	"testing"
)

func TestInspectContainerInfersOwnerFromHomeMount(t *testing.T) {
	oldRunDockerInspect := runDockerInspect
	runDockerInspect = func(containerID string) ([]byte, error) {
		if containerID != "abcdef1234567890" {
			t.Fatalf("containerID = %q, want full id", containerID)
		}
		return []byte(`[
			{
				"Id": "abcdef1234567890",
				"Name": "/trainer",
				"Config": {"User": ""},
				"HostConfig": {"Binds": null},
				"Mounts": [
					{"Type": "bind", "Source": "/home/alice/projects/model", "Destination": "/workspace"}
				]
			}
		]`), nil
	}
	t.Cleanup(func() { runDockerInspect = oldRunDockerInspect })

	info := InspectContainer(&ContainerInfo{ID: "abcdef123456", FullID: "abcdef1234567890", Name: "abcdef123456"})
	if info.OwnerUser != "alice" {
		t.Fatalf("OwnerUser = %q, want alice", info.OwnerUser)
	}
	if info.Name != "trainer" {
		t.Fatalf("Name = %q, want trainer", info.Name)
	}
}

func TestInspectContainerInfersOwnerFromHostConfigBind(t *testing.T) {
	oldRunDockerInspect := runDockerInspect
	runDockerInspect = func(containerID string) ([]byte, error) {
		return []byte(`[
			{
				"Id": "abcdef1234567890",
				"Name": "/trainer",
				"Config": {"User": ""},
				"HostConfig": {"Binds": ["/home/bob/data:/data:rw"]},
				"Mounts": []
			}
		]`), nil
	}
	t.Cleanup(func() { runDockerInspect = oldRunDockerInspect })

	info := InspectContainer(&ContainerInfo{ID: "abcdef123456"})
	if info.OwnerUser != "bob" {
		t.Fatalf("OwnerUser = %q, want bob", info.OwnerUser)
	}
}

func TestInspectContainerIgnoresContainerInternalUser(t *testing.T) {
	oldRunDockerInspect := runDockerInspect
	runDockerInspect = func(containerID string) ([]byte, error) {
		return []byte(`[
			{
				"Id": "abcdef1234567890",
				"Name": "/trainer",
				"Config": {"User": "carol:users"},
				"HostConfig": {"Binds": null},
				"Mounts": []
			}
		]`), nil
	}
	t.Cleanup(func() { runDockerInspect = oldRunDockerInspect })

	info := InspectContainer(&ContainerInfo{ID: "abcdef123456"})
	if info.OwnerUser != "" {
		t.Fatalf("OwnerUser = %q, want empty owner", info.OwnerUser)
	}
}

func TestInspectContainerKeepsCgroupInfoWhenInspectFails(t *testing.T) {
	oldRunDockerInspect := runDockerInspect
	runDockerInspect = func(containerID string) ([]byte, error) {
		return nil, errors.New("docker unavailable")
	}
	t.Cleanup(func() { runDockerInspect = oldRunDockerInspect })

	info := InspectContainer(&ContainerInfo{ID: "abcdef123456", FullID: "abcdef1234567890", Name: "abcdef123456"})
	if info.ID != "abcdef123456" || info.Name != "abcdef123456" || info.OwnerUser != "" {
		t.Fatalf("InspectContainer() = %+v, want original info without owner", info)
	}
}

func TestInferUserFromPath(t *testing.T) {
	cases := map[string]string{
		"/home/alice/project":       "alice",
		"/Users/bob/workspace":      "bob",
		"/var/lib/docker/volumes/x": "",
		"/home/root/project":        "",
	}

	for path, want := range cases {
		if got := inferUserFromPath(path); got != want {
			t.Fatalf("inferUserFromPath(%q) = %q, want %q", path, got, want)
		}
	}
}
