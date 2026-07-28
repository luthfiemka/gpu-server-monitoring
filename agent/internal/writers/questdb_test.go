package writers

import (
	"os"
	"path/filepath"
	"testing"
)

func withProcFixture(t *testing.T, files map[string]string) {
	t.Helper()

	root := t.TempDir()
	oldProcRoot := procRoot
	oldPasswdPath := passwdPath
	procRoot = filepath.Join(root, "proc")
	passwdPath = filepath.Join(root, "passwd")
	t.Cleanup(func() {
		procRoot = oldProcRoot
		passwdPath = oldPasswdPath
	})

	if err := os.WriteFile(passwdPath, []byte("root:x:0:0:root:/root:/bin/sh\nalice:x:1001:1001:Alice:/home/alice:/bin/sh\nbob:x:1002:1002:Bob:/home/bob:/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write passwd: %v", err)
	}

	for name, content := range files {
		path := filepath.Join(procRoot, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir fixture: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture %s: %v", name, err)
		}
	}
}

func TestGetUsernameUsesStatusUIDForNormalProcess(t *testing.T) {
	withProcFixture(t, map[string]string{
		"123/status": "Name:\tpython\nUid:\t1001\t1001\t1001\t1001\n",
	})

	if got := getUsername(123, false); got != "alice" {
		t.Fatalf("getUsername() = %q, want alice", got)
	}
}

func TestGetUsernameUsesLoginUIDForRootProcessStartedByUser(t *testing.T) {
	withProcFixture(t, map[string]string{
		"123/status":   "Name:\tpython\nUid:\t0\t0\t0\t0\n",
		"123/loginuid": "1002\n",
	})

	if got := getUsername(123, false); got != "bob" {
		t.Fatalf("getUsername() = %q, want bob", got)
	}
}

func TestGetUsernameUsesLoginUIDForContainerRootProcess(t *testing.T) {
	withProcFixture(t, map[string]string{
		"123/status":   "Name:\tpython\nUid:\t0\t0\t0\t0\n",
		"123/loginuid": "1001\n",
	})

	if got := getUsername(123, true); got != "alice" {
		t.Fatalf("getUsername() = %q, want alice", got)
	}
}

func TestGetUsernameUsesUserNamespaceRootMappingForContainer(t *testing.T) {
	withProcFixture(t, map[string]string{
		"123/status":   "Name:\tpython\nUid:\t0\t0\t0\t0\n",
		"123/loginuid": "4294967295\n",
		"123/uid_map":  "0 1002 1\n1 100000 65535\n",
	})

	if got := getUsername(123, true); got != "bob" {
		t.Fatalf("getUsername() = %q, want bob", got)
	}
}

func TestGetUsernameFallsBackToRootWhenContainerOwnerIsUnavailable(t *testing.T) {
	withProcFixture(t, map[string]string{
		"123/status":   "Name:\tpython\nUid:\t0\t0\t0\t0\n",
		"123/loginuid": "4294967295\n",
		"123/uid_map":  "0 0 4294967295\n",
	})

	if got := getUsername(123, true); got != "root" {
		t.Fatalf("getUsername() = %q, want root", got)
	}
}
