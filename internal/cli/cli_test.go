package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidatesAreThePlacesThePanelLooksToo(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	got := Candidates()
	want := []string{
		"/usr/local/bin/oshelly",
		"/home/tester/.local/bin/oshelly",
		"/opt/homebrew/bin/oshelly",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d candidates, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("candidate %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// fake writes an executable that stands in for oshelly, so the client can be
// tested without installing anything.
func fake(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "oshelly")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestListParsesTheJSONThePanelReads(t *testing.T) {
	path := fake(t, `cat <<'EOF'
{
  "version": 1,
  "path": "/home/tester/.config/oshelly/oshelly.conf",
  "exists": true,
  "platform": "darwin",
  "terminal": "the default .command handler",
  "commands": [
    {"name": "Restart audio", "command": "systemctl --user restart pipewire"},
    {"name": "Update", "command": "omarchy update"}
  ]
}
EOF
`)
	client := &Client{Path: path}
	got, err := client.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != "/home/tester/.config/oshelly/oshelly.conf" {
		t.Errorf("path = %q", got.Path)
	}
	if len(got.Commands) != 2 {
		t.Fatalf("got %d commands, want 2", len(got.Commands))
	}
	if got.Commands[0].Name != "Restart audio" {
		t.Errorf("first command = %#v", got.Commands[0])
	}
	if got.Commands[1].Name != "Update" {
		t.Errorf("second command = %#v", got.Commands[1])
	}
}

// A missing commands key must not leave a nil slice, because a caller that
// ranges over it is fine but a caller that asks for its length in a menu is
// looking at a different shape than an empty config.
func TestListTurnsAMissingListIntoAnEmptyOne(t *testing.T) {
	path := fake(t, `printf '{"version":1,"path":"/tmp/x","commands":null}\n'`)
	client := &Client{Path: path}
	got, err := client.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Commands == nil {
		t.Fatal("commands is nil")
	}
	if !got.Empty() {
		t.Errorf("got %d commands, want none", len(got.Commands))
	}
}

// A config error is an error, not an empty list: the difference is the line
// number, which is the only useful thing to show.
func TestListReportsTheCliReason(t *testing.T) {
	path := fake(t, `printf 'line 4 has no colon\n' >&2; exit 1`)
	client := &Client{Path: path}
	_, err := client.List(context.Background())
	if err == nil {
		t.Fatal("a failing list was accepted")
	}
	if !strings.Contains(err.Error(), "line 4 has no colon") {
		t.Errorf("error %q does not contain the CLI's reason", err)
	}
}

func TestRunKeepsTheExitCodeAndBothStreams(t *testing.T) {
	path := fake(t, `printf 'Launched "Update".\n'; exit 0`)
	client := &Client{Path: path}
	res := client.Run(context.Background(), "Update")
	if !res.OK() {
		t.Fatalf("exit %d, detail %q", res.Code, res.Detail)
	}
	if !strings.Contains(res.Message, "Launched") {
		t.Errorf("message = %q", res.Message)
	}
}

func TestAFailedRunCarriesTheReason(t *testing.T) {
	path := fake(t, `printf 'no terminal found.\n' >&2; exit 2`)
	client := &Client{Path: path}
	res := client.Run(context.Background(), "Update")
	if res.Code != 2 {
		t.Errorf("exit %d, want 2", res.Code)
	}
	if !strings.Contains(res.Detail, "no terminal found") {
		t.Errorf("detail = %q", res.Detail)
	}
}

func TestEditCallsTheCLI(t *testing.T) {
	path := fake(t, `printf 'Editing /home/tester/.config/oshelly/oshelly.conf.\n'; exit 0`)
	client := &Client{Path: path}
	res := client.Edit(context.Background())
	if !res.OK() {
		t.Fatalf("exit %d, detail %q", res.Code, res.Detail)
	}
	if !strings.Contains(res.Message, "Editing") {
		t.Errorf("message = %q", res.Message)
	}
}

func TestFlattenMakesOneLine(t *testing.T) {
	got := flatten("first line\r\nsecond   line\n")
	if got != "first line second line" {
		t.Errorf("flatten = %q", got)
	}
}

func TestEnvironmentKeepsTheSessionAndGuaranteesHome(t *testing.T) {
	t.Setenv("HOME", "")
	env := environment()
	found := false
	for _, v := range env {
		if strings.HasPrefix(v, "HOME=") && v != "HOME=" {
			found = true
		}
	}
	if !found {
		t.Error("no HOME was added when the session had none")
	}
}
