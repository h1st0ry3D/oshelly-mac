// Package panel turns the list of commands into the shape of the menu.
//
// It is deliberately free of any drawing. What a row says, what clicking one
// would do, and where the separators fall are all decided here and can be
// checked without a menu bar, a screen or a Mac. The systray code does nothing
// but put these on screen.
//
// The shape is deliberately simpler than the Omarchy panel's: a menu bar has
// one column, so a command is one row and there are no submenus to keep in
// step with it.
package panel

import (
	"fmt"
	"strings"

	"github.com/h1st0ry3D/oshelly-mac/internal/cli"
)

// Action is what an item does when it is clicked.
type Action int

const (
	// ActionNone is a label, a heading, or a separator.
	ActionNone Action = iota
	// ActionRun launches one command in the terminal. This is one call and
	// oshelly owns the terminal selection.
	ActionRun
	// ActionEdit opens the config in the default text editor.
	ActionEdit
	// ActionReload re-reads the config, for after a hand edit.
	ActionReload
	// ActionQuit leaves.
	ActionQuit
)

// Item is one thing in the menu.
type Item struct {
	// Title is the text.
	Title string
	// Tooltip is the longer explanation. A menu has room for it where a bar
	// button does not.
	Tooltip string
	// Disabled greys it out and makes it do nothing. Headings and the last
	// result line are like this.
	Disabled bool
	// SeparatorBefore draws a line above.
	SeparatorBefore bool
	// Action is what clicking does. Ignored when Disabled.
	Action Action
	// Name is the command an ActionRun will launch.
	Name string
	// StatusDetail marks the line carrying the reason an action failed, which
	// is not a command's row and has no action of its own.
	StatusDetail bool
}

// Status is the line at the top, which reports the last action.
type Status struct {
	Text  string
	Error bool
	// Detail is the reason underneath, in oshelly's own words. It is shown as
	// its own line rather than dropped, because the reason is the actionable
	// part: "no terminal found" is a fixable thing, and "could not run" is not.
	Detail string
}

// Idle is the status before anything has happened in this session.
var Idle = Status{Text: "Run a command"}

// Model is the whole menu.
type Model struct {
	Status Status
	Items  []*Item
}

// Build turns settings into the menu.
//
// busy marks an action in flight. Commands are refused while it is set, so two
// clicks cannot start two terminals in the time the first CLI call takes to
// return.
func Build(s cli.Settings, status Status, busy bool) *Model {
	m := &Model{Status: status}
	m.Items = append(m.Items, &Item{
		Title:    oneLine(status.Text),
		Tooltip:  tooltipFor(status),
		Disabled: true,
	})
	if d := oneLine(status.Detail); d != "" {
		m.Items = append(m.Items, &Item{
			Title:        d,
			Tooltip:      "What oshelly said",
			Disabled:     true,
			StatusDetail: true,
		})
	}

	if s.Empty() {
		m.Items = append(m.Items, &Item{
			Title:           "No commands configured.",
			Tooltip:         "One command per line: name: command",
			Disabled:        true,
			SeparatorBefore: true,
		})
		where := s.Path
		if where == "" {
			where = "the config file"
		}
		m.Items = append(m.Items, &Item{
			Title:    "Edit " + where,
			Tooltip:  "Add a line like: Restart audio: systemctl --user restart pipewire",
			Disabled: true,
		})
	}

	for i, c := range s.Commands {
		m.Items = append(m.Items, &Item{
			Title:           c.Name,
			Tooltip:         "Run this in your default terminal",
			SeparatorBefore: i == 0,
			Action:          ActionRun,
			Name:            c.Name,
			Disabled:        busy,
		})
	}

	m.Items = append(m.Items,
		&Item{
			Title:           "Edit config",
			Tooltip:         "Open the config file in your default text editor",
			SeparatorBefore: true,
			Action:          ActionEdit,
			Disabled:        busy,
		},
		&Item{
			Title:    "Reload",
			Tooltip:  "Read the config again after editing it",
			Action:   ActionReload,
			Disabled: busy,
		},
		&Item{
			Title:    "Quit",
			Tooltip:  "Leave. Nothing is launched on the way out.",
			Action:   ActionQuit,
			Disabled: false,
		},
	)

	return m
}

// Unreadable is the menu when the CLI is there but its config will not parse.
//
// It is kept apart from an empty config on purpose. "No commands configured"
// and "line 4 has no colon" are different situations, and showing the first
// for the second would hide the line number that is the whole fix.
func Unreadable(status Status) *Model {
	m := &Model{Status: status}
	m.Items = append(m.Items, &Item{
		Title:    oneLine(status.Text),
		Disabled: true,
	})
	if d := oneLine(status.Detail); d != "" {
		m.Items = append(m.Items, &Item{
			Title:        d,
			Tooltip:      "What oshelly said",
			Disabled:     true,
			StatusDetail: true,
		})
	}
	m.Items = append(m.Items,
		&Item{
			Title:           "Edit config",
			Tooltip:         "Open the config in your default text editor and fix that line",
			SeparatorBefore: true,
			Action:          ActionEdit,
		},
		&Item{
			Title:   "Reload",
			Tooltip: "Read the config again after fixing it",
			Action:  ActionReload,
		},
		&Item{
			Title:   "Quit",
			Tooltip: "Leave. Nothing is launched on the way out.",
			Action:  ActionQuit,
		},
	)
	return m
}

// NotInstalled is the whole menu when there is nothing to drive.
//
// It says where it looked, because "nothing happens when I click" is the
// failure this state is for.
//
// This is the one menu that keeps a Reload: with nothing driving it there is
// nothing that re-reads on its own, so installing oshelly while this is open
// would otherwise need the app restarting.
func NotInstalled(where []string) *Model {
	m := &Model{
		Status: Status{
			Text:  "oshelly is not installed.",
			Error: true,
		},
		Items: []*Item{{
			Title:    "Put the oshelly binary in one of these:",
			Disabled: true,
		}},
	}
	// The paths are listed rather than summarized. A menu that says nothing
	// happens with no place to look is not actionable, and this is the state
	// where the paths are the whole story.
	for _, p := range where {
		m.Items = append(m.Items, &Item{
			Title:    p,
			Disabled: true,
		})
	}

	m.Items = append(m.Items,
		&Item{
			Title:           "Reload settings",
			Tooltip:         "Look again, after putting it there",
			SeparatorBefore: true,
			Action:          ActionReload,
		},
		&Item{
			Title:   "Quit",
			Tooltip: "Leave. Nothing is launched on the way out.",
			Action:  ActionQuit,
		},
	)
	return m
}

// ForRun builds the line that reports a finished launch.
func ForRun(res cli.Result, name string) Status {
	return forResult(res, "Launched "+name+".", "Could not run "+name+".")
}

// ForEdit builds the line that reports opening the config.
//
// A failure keeps oshelly's reason because "no editor found" is the part that
// can be fixed.
func ForEdit(res cli.Result) Status {
	return forResult(res, "Editing the config.", "Could not open the config.")
}

// forResult is the shape every completed action shares: a success uses the
// CLI's own message, and a failure keeps the CLI's reason under a line saying
// what failed.
func forResult(res cli.Result, fallback, failure string) Status {
	if res.OK() {
		text := oneLine(res.Message)
		if text == "" {
			text = fallback
		}
		return Status{Text: text}
	}

	detail := oneLine(res.Detail)
	if detail == "" {
		detail = oneLine(res.Message)
	}
	return Status{Text: failure, Error: true, Detail: detail}
}

// tooltipFor is the longer explanation of the status line.
func tooltipFor(s Status) string {
	if s.Error {
		return "The last launch did not finish. Read the line below, then try again."
	}
	return ""
}

// oneLine makes a string fit a menu item.
//
// A menu item is one line by definition, and text that does not fit gets its
// middle replaced by an ellipsis, which would cut out the part explaining what
// happened. Runs of whitespace become one space, so a wrapped reason reads as
// the sentence it is rather than as one run-together word.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// Describe renders the model as text, which is what the tests check and what a
// person reads first when something is wrong.
func (m *Model) Describe() string {
	var b strings.Builder
	mark := func(st Status) string {
		if st.Error {
			return st.Text + "  (error)"
		}
		return st.Text
	}
	b.WriteString(mark(m.Status))
	for _, it := range m.Items {
		if it.SeparatorBefore {
			b.WriteString("\n--")
		}
		flags := ""
		if it.Disabled {
			flags += " [disabled]"
		}
		b.WriteString("\n  " + it.Title + flags)
	}
	return b.String()
}

// Summary is a one-line count, used in the log when the menu is rebuilt, so a
// menu that came out empty says why.
func (m *Model) Summary() string {
	return fmt.Sprintf("%d items", len(m.Items))
}
