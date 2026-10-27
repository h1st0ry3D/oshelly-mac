package panel

import (
	"strings"
	"testing"

	"github.com/h1st0ry3D/oshelly-mac/internal/cli"
)

func settings(names ...string) cli.Settings {
	s := cli.Settings{Path: "/home/tester/.config/oshelly/oshelly.conf", Exists: true}
	for _, n := range names {
		s.Commands = append(s.Commands, cli.Command{Name: n, Command: "true"})
	}
	return s
}

// find returns the first item with the given title.
func find(m *Model, title string) *Item {
	for _, it := range m.Items {
		if it.Title == title {
			return it
		}
	}
	return nil
}

func TestEveryCommandGetsOneRowThatRunsIt(t *testing.T) {
	m := Build(settings("Restart audio", "Update"), Idle, false)
	for _, want := range []string{"Restart audio", "Update"} {
		it := find(m, want)
		if it == nil {
			t.Fatalf("no item for command %q", want)
		}
		if it.Action != ActionRun {
			t.Errorf("%q has action %d, want ActionRun", want, it.Action)
		}
		if it.Name != want {
			t.Errorf("%q will run %q", want, it.Name)
		}
	}

	if it := find(m, "Edit config"); it == nil || it.Action != ActionEdit {
		t.Error("there is no Edit config item")
	}
	if it := find(m, "Terminal: System default"); it == nil {
		t.Error("there is no Terminal submenu")
	}
}

// The tick has to be on the saved choice, not on the first entry or on none.
func TestTheTerminalChoiceIsTicked(t *testing.T) {
	s := settings("Restart audio")
	s.TerminalChoice = "Ghostty"
	m := Build(s, Idle, false)

	terminal := find(m, "Terminal: Ghostty")
	if terminal == nil {
		t.Fatalf("no Terminal submenu for Ghostty:\n%s", m.Describe())
	}
	var ghostty, system *Item
	for _, child := range terminal.Children {
		switch child.Title {
		case "Ghostty":
			ghostty = child
		case "System default":
			system = child
		}
	}
	if ghostty == nil || !ghostty.Checked {
		t.Error("Ghostty is not ticked")
	}
	if system == nil || system.Checked {
		t.Error("System default is ticked while Ghostty is the choice")
	}
}

// A separator before the first command is what tells it apart from the status
// line. Without it the first command reads as part of the result message.
func TestTheFirstCommandIsSeparatedFromTheStatus(t *testing.T) {
	m := Build(settings("Restart audio", "Update"), Idle, false)
	if it := find(m, "Restart audio"); it == nil || !it.SeparatorBefore {
		t.Error("the first command has no separator above it")
	}
	if it := find(m, "Update"); it == nil || it.SeparatorBefore {
		t.Error("the second command has a separator above it, which would be a wall")
	}
}

func TestBusyRefusesEveryCommandAndReload(t *testing.T) {
	m := Build(settings("Restart audio", "Update"), Idle, true)
	for _, want := range []string{"Restart audio", "Update", "Terminal: System default", "Edit config", "Reload"} {
		it := find(m, want)
		if it == nil {
			t.Fatalf("no item %q", want)
		}
		if !it.Disabled {
			t.Errorf("%q is enabled while a launch is in flight", want)
		}
	}
	if it := find(m, "Quit"); it == nil || it.Disabled {
		t.Error("Quit is disabled, which would trap the app open")
	}
}

func TestAnEmptyConfigSaysWhereToWrite(t *testing.T) {
	path := "/home/tester/.config/oshelly/oshelly.conf"
	m := Build(cli.Settings{Path: path, Exists: true}, Idle, false)
	if find(m, "No commands configured.") == nil {
		t.Error("no empty-state item")
	}
	if find(m, "Edit "+path) == nil {
		t.Errorf("the empty state does not name the file:\n%s", m.Describe())
	}
}

// A parse error and an empty config are different situations. Tying them to
// one model would hide the line number that is the whole fix.
func TestAnUnreadableConfigIsNotDrawnAsAnEmptyOne(t *testing.T) {
	m := Unreadable(Status{
		Text:   "Could not read the oshelly config.",
		Error:  true,
		Detail: "line 4 has no colon",
	})
	if find(m, "No commands configured.") != nil {
		t.Error("a parse error was drawn as an empty config")
	}
	if find(m, "line 4 has no colon") == nil {
		t.Errorf("the reason is missing:\n%s", m.Describe())
	}
	if it := find(m, "Edit config"); it == nil || it.Action != ActionEdit {
		t.Error("a broken config has no way to open the file that is broken")
	}
	if it := find(m, "Reload"); it == nil || it.Action != ActionReload {
		t.Error("there is no way to re-read after fixing the file")
	}
}

func TestNotInstalledNamesEveryPlaceItLooked(t *testing.T) {
	where := []string{"/usr/local/bin/oshelly", "/home/tester/.local/bin/oshelly"}
	m := NotInstalled(where)
	for _, p := range where {
		if find(m, p) == nil {
			t.Errorf("the menu does not name %q:\n%s", p, m.Describe())
		}
	}
	if it := find(m, "Reload settings"); it == nil || it.Action != ActionReload {
		t.Error("there is no way to look again after installing it")
	}
}

func TestForRunReportsBothOutcomes(t *testing.T) {
	ok := ForRun(cli.Result{Code: 0, Message: `Launched "Update".`}, "Update")
	if ok.Error {
		t.Error("a successful launch was marked as an error")
	}
	if ok.Text != `Launched "Update".` {
		t.Errorf("text = %q", ok.Text)
	}

	bad := ForRun(cli.Result{Code: 2, Detail: "no terminal found."}, "Update")
	if !bad.Error {
		t.Error("a failed launch was not marked as an error")
	}
	if !strings.Contains(bad.Text, "Update") {
		t.Errorf("text = %q, want the command's name", bad.Text)
	}
	if bad.Detail != "no terminal found." {
		t.Errorf("detail = %q", bad.Detail)
	}
}

// The fallback matters when the CLI dies without a message. A row saying
// "Could not run Update." with nothing under it is still better than an exit
// code on its own.
func TestForRunFallsBackToTheMessage(t *testing.T) {
	st := ForRun(cli.Result{Code: -1, Message: "signal: killed"}, "Update")
	if st.Detail != "signal: killed" {
		t.Errorf("detail = %q", st.Detail)
	}
}

func TestForTerminalReportsBothOutcomes(t *testing.T) {
	ok := ForTerminal(cli.Result{Code: 0, Message: "terminal = Ghostty on darwin."})
	if ok.Error {
		t.Error("a saved terminal was marked as an error")
	}
	if !strings.Contains(ok.Text, "Ghostty") {
		t.Errorf("text = %q", ok.Text)
	}

	bad := ForTerminal(cli.Result{Code: 3, Detail: "open -a Ghostty: no such application"})
	if !bad.Error {
		t.Error("a failed terminal change was not marked as an error")
	}
	if bad.Detail == "" {
		t.Error("the reason was dropped")
	}
}

func TestForEditReportsBothOutcomes(t *testing.T) {
	ok := ForEdit(cli.Result{Code: 0, Message: "Editing /home/tester/.config/oshelly/oshelly.conf."})
	if ok.Error {
		t.Error("a successful edit was marked as an error")
	}
	if !strings.Contains(ok.Text, "Editing") {
		t.Errorf("text = %q", ok.Text)
	}

	bad := ForEdit(cli.Result{Code: 2, Detail: "no editor found."})
	if !bad.Error {
		t.Error("a failed edit was not marked as an error")
	}
	if bad.Detail != "no editor found." {
		t.Errorf("detail = %q", bad.Detail)
	}
}

func TestOneLineMakesAMenuItemOneLine(t *testing.T) {
	if got := oneLine("first\nsecond   line"); got != "first second line" {
		t.Errorf("oneLine = %q", got)
	}
}
