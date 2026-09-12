package runner

import (
	"errors"
	"os/exec"
	"reflect"
	"testing"
)

func TestIsPodmanUnavailable(t *testing.T) {
	machineStopped := errors.New("podman ps: exit status 125: Cannot connect to Podman. Please verify your connection to the Linux system using `podman system connection list`, or try `podman machine init` and `podman machine start` to manage a new Linux VM\nError: unable to connect to Podman socket: failed to connect: dial tcp 127.0.0.1:49248: connect: connection refused")
	if !IsPodmanUnavailable(machineStopped) {
		t.Fatal("stopped machine / refused socket should be unavailable")
	}
	if IsPodmanUnavailable(errors.New("podman ps: exit status 125: can't evaluate field Names")) {
		t.Fatal("template failure is not unavailable")
	}
	if IsPodmanUnavailable(nil) {
		t.Fatal("nil")
	}
}

func TestIsBenignPodmanListError(t *testing.T) {
	stale := []byte("Error: container abc does not exist in database: no such container\n")
	if !isBenignPodmanListError(nil, stale, errors.New("exit status 125")) {
		t.Fatal("stale container race should be benign")
	}
	if !isBenignPodmanListError([]byte("no container with name or ID"), nil, errors.New("exit status 125")) {
		t.Fatal("missing name on stdout should be benign")
	}
	if isBenignPodmanListError(nil, []byte("unable to connect to Podman socket"), errors.New("exit status 125")) {
		t.Fatal("engine/connection failure must not be treated as already-down")
	}
	if isBenignPodmanListError(nil, nil, errors.New("exit status 125")) {
		t.Fatal("bare exit 125 without a missing-container message must not be benign")
	}
	if isBenignPodmanListError(nil, stale, nil) {
		t.Fatal("success is not a list error")
	}
}

func TestIsPodmanExit(t *testing.T) {
	if isPodmanExit(errors.New("exit status 125"), 125) {
		t.Fatal("plain error is not *exec.ExitError")
	}
	err := exec.Command("sh", "-c", "exit 125").Run()
	if !isPodmanExit(err, 125) {
		t.Fatalf("sh exit 125: %v", err)
	}
	if isPodmanExit(err, 1) {
		t.Fatal("wrong exit code matched")
	}
}

func TestParsePsNameLines(t *testing.T) {
	got := parsePsNameLines([]byte("a,b\n c \n"))
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	if len(parsePsNameLines(nil)) != 0 {
		t.Fatal("empty output should yield no names")
	}
}

func TestWrapPodmanPsError_includesStderr(t *testing.T) {
	err := wrapPodmanPsError(errors.New("exit status 125"), []byte("unable to connect to Podman socket\n"))
	if err == nil || err.Error() != "podman ps: exit status 125: unable to connect to Podman socket" {
		t.Fatalf("got %v", err)
	}
	bare := wrapPodmanPsError(errors.New("exit status 125"), nil)
	if bare.Error() != "podman ps: exit status 125" {
		t.Fatalf("got %v", bare)
	}
}
