package main

import (
	"testing"

	"github.com/h1st0ry3D/oshelly-mac/internal/cli"
	"github.com/h1st0ry3D/oshelly-mac/internal/panel"
)

func settings(names ...string) cli.Settings {
	s := cli.Settings{Path: "/home/tester/.config/oshelly/oshelly.conf", Exists: true}
	for _, n := range names {
		s.Commands = append(s.Commands, cli.Command{Name: n, Command: "true"})
	}
	return s
}

func shapeOfMenu(s cli.Settings, st panel.Status) []string {
	return shapeOf(panel.Build(s, st, false).Items)
}

// The menu is rebuilt only when its shape changes. A status message changes on
// every launch, so including it would rebuild the menu under the pointer for
// no reason; a command being added or renamed is a real change.
func TestTheShapeChangesOnlyWhenTheMenuDoes(t *testing.T) {
	cases := []struct {
		name          string
		before, after []string
		wantSame      bool
	}{
		{"nothing changed",
			shapeOfMenu(settings("one", "two"), panel.Idle),
			shapeOfMenu(settings("one", "two"), panel.Idle), true},
		{"only the status text changed",
			shapeOfMenu(settings("one", "two"), panel.Status{Text: "Run a command"}),
			shapeOfMenu(settings("one", "two"), panel.Status{Text: "Launched one."}), true},
		{"a command was added",
			shapeOfMenu(settings("one"), panel.Idle),
			shapeOfMenu(settings("one", "two"), panel.Idle), false},
		{"a command was removed",
			shapeOfMenu(settings("one", "two"), panel.Idle),
			shapeOfMenu(settings("one"), panel.Idle), false},
		{"the commands were reordered",
			shapeOfMenu(settings("one", "two"), panel.Idle),
			shapeOfMenu(settings("two", "one"), panel.Idle), false},
	}
	for _, c := range cases {
		if got := sameShape(c.before, c.after); got != c.wantSame {
			t.Errorf("%s: sameShape = %v, want %v", c.name, got, c.wantSame)
		}
	}
}

// A reason is a line of its own, so one appearing does change the shape, and
// its text does not.
func TestAReasonAddsALineWithoutItsTextChangingTheShape(t *testing.T) {
	plain := shapeOfMenu(settings("one"), panel.Status{Text: "Could not run one.", Error: true})
	withDetail := shapeOfMenu(settings("one"), panel.Status{
		Text: "Could not run one.", Error: true, Detail: "no terminal found.",
	})
	if sameShape(plain, withDetail) {
		t.Error("a reason line did not change the shape, so it would never be drawn")
	}

	other := shapeOfMenu(settings("one"), panel.Status{
		Text: "Could not run one.", Error: true, Detail: "line 4 has no colon",
	})
	if !sameShape(withDetail, other) {
		t.Error("changing the reason's text rebuilt the menu")
	}
}
