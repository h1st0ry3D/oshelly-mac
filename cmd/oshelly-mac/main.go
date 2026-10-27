// Command oshelly-mac puts your shell commands in the macOS menu bar.
//
// It is a front end. Everything that matters happens in oshelly: which commands
// exist, how a name is parsed, and which terminal opens. This program decides
// how to draw a flat list and nothing else, so it cannot disagree with the
// Omarchy panel about what a click does.
//
// One click is one call to `oshelly run`. The menu is built once and then
// edited in place; systray adds items rather than replacing them, so a rebuild
// happens only when the shape changes, which is when the commands gain, lose
// or are reordered. A status line changing is not a rebuild.
package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"fyne.io/systray"

	"github.com/h1st0ry3D/oshelly-mac/internal/cli"
	"github.com/h1st0ry3D/oshelly-mac/internal/icon"
	"github.com/h1st0ry3D/oshelly-mac/internal/panel"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("oshelly-mac: ")
	systray.Run(onReady, func() {})
}

var state = &app{status: panel.Idle}

func onReady() {
	// systray draws only the image when the title is empty, so this is the
	// terminal glyph rather than a word next to it. It is a template image:
	// black with an alpha channel, which macOS inverts for a dark menu bar and
	// leaves alone for a light one, so one file works on both.
	systray.SetTemplateIcon(icon.Terminal(), icon.Terminal())
	systray.SetTitle("")
	systray.SetTooltip("Oshelly: run a shell command")

	state.reload()
}

// app is the live state.
//
// systray runs its callbacks on its own goroutine, and a launch can take the
// length of a CLI timeout, so the state and the drawing are kept apart by one
// mutex rather than by hoping. renderMu is separate because render is called
// from both a click and a reload, and it owns the menu items themselves.
type app struct {
	mu       sync.Mutex
	renderMu sync.Mutex

	client   *cli.Client
	settings cli.Settings
	status   panel.Status
	// readError is set when the binary is there but its config will not parse.
	// It is kept apart from an empty config, because the two need different
	// words and one of them has a line number in it.
	readError string
	busy      bool

	// drawn is the shape of the model that is on screen. The menu is rebuilt
	// only when that changes, because a rebuild closes whatever is open, and a
	// status text changes on every click.
	drawn []string
	// roots are the items a rebuild takes down, and all is every item in the
	// order the model walks, which is how a render finds one again to edit it.
	roots []*systray.MenuItem
	all   []*systray.MenuItem
}

// reload re-reads the settings and updates the menu.
//
// It runs at start, after every launch, and from the Reload item, so a hand
// edit of the config takes effect without restarting anything.
func (a *app) reload() {
	client, err := cli.New()
	if err != nil {
		log.Printf("%v", err)
		a.mu.Lock()
		a.client = nil
		a.readError = ""
		a.mu.Unlock()
		a.render()
		return
	}

	settings, err := client.List(context.Background())
	if err != nil {
		log.Printf("could not read the settings: %v", err)
		a.mu.Lock()
		a.client = client
		a.settings = cli.Settings{}
		a.readError = err.Error()
		a.mu.Unlock()
		a.setStatus(panel.Status{
			Text:   "Could not read the oshelly config.",
			Error:  true,
			Detail: err.Error(),
		})
		return
	}

	a.mu.Lock()
	a.client = client
	a.settings = settings
	a.readError = ""
	a.mu.Unlock()
	a.render()
}

// setStatus records the last result and puts the menu back with it.
func (a *app) setStatus(s panel.Status) {
	a.mu.Lock()
	a.status = s
	a.mu.Unlock()
	a.render()
}

// render puts the current state on screen.
func (a *app) render() {
	a.renderMu.Lock()
	defer a.renderMu.Unlock()

	a.mu.Lock()
	settings, status, busy := a.settings, a.status, a.busy
	installed := a.client != nil
	readError := a.readError
	a.mu.Unlock()

	var model *panel.Model
	switch {
	case !installed:
		model = panel.NotInstalled(cli.Candidates())
	case readError != "":
		model = panel.Unreadable(status)
	default:
		model = panel.Build(settings, status, busy)
	}

	want := shapeOf(model.Items)
	if !sameShape(a.drawn, want) {
		for _, it := range a.roots {
			it.Remove()
		}
		a.draw(model)
		a.drawn = want
	}
	apply(model, a.all)
}

// draw puts the model on screen in one walk, recording every item it makes so
// a later render can find it again.
func (a *app) draw(model *panel.Model) {
	a.roots, a.all = nil, nil
	for _, it := range model.Items {
		if it.SeparatorBefore {
			// The blank is a real item on screen and has to be in all as
			// well, or every title after the first separator would be set on
			// the item above it.
			blank := a.blank()
			a.roots = append(a.roots, blank)
			a.all = append(a.all, blank)
		}
		item := a.create(it)
		a.roots = append(a.roots, item)
		a.all = append(a.all, item)
		a.wire(item, callOf(it))
	}
	log.Printf("menu rebuilt: %s", model.Summary())
}

// create makes one item from the model.
func (a *app) create(it *panel.Item) *systray.MenuItem {
	item := systray.AddMenuItem(it.Title, it.Tooltip)
	setEnabled(item, !it.Disabled)
	return item
}

// blank is the line between blocks. A menu bar has no separator of its own for
// this, so it is an empty item that cannot be clicked.
func (a *app) blank() *systray.MenuItem {
	item := systray.AddMenuItem("", "")
	setEnabled(item, false)
	return item
}

// callOf is what clicking an item does. The model carries every argument the
// action needs, which is why this is a copy and not a lookup by name.
func callOf(it *panel.Item) call {
	return call{action: it.Action, name: it.Name}
}

// apply gives every item on screen what the model says it should have.
//
// The two are walked the same way, so the nth item of the walk is the nth on
// screen, which the shape check is what keeps true. A title is set on the item
// that is there rather than the item replaced, so a new status message does
// not rebuild the menu under the pointer.
func apply(model *panel.Model, live []*systray.MenuItem) {
	at := 0
	for _, it := range model.Items {
		if it.SeparatorBefore {
			at++ // the blank item above has nothing to set
		}
		if at >= len(live) {
			return
		}
		item := live[at]
		at++
		item.SetTitle(it.Title)
		item.SetTooltip(it.Tooltip)
		setEnabled(item, !it.Disabled)
	}
}

// shapeOf is the structure of a menu: one line per item, saying where it sits
// and what it is.
//
// What it is means what clicking it does and what it does that to, so that
// renaming or reordering two commands is a different shape. A title is
// deliberately not part of it: a title changes on every launch and every
// status message, and a menu rebuilt for one would close under the pointer.
// That is what apply is for.
func shapeOf(items []*panel.Item) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, fmt.Sprintf("%d %s %t %t",
			int(it.Action), it.Name, it.SeparatorBefore, it.StatusDetail))
	}
	return out
}

func sameShape(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// call is everything a menu item needs to say what it does when clicked.
type call struct {
	action panel.Action
	name   string
}

// wire sends every click on an item to run.
func (a *app) wire(item *systray.MenuItem, c call) {
	go func() {
		for range item.ClickedCh {
			a.run(c)
		}
	}()
}

// run does one thing, then brings the menu up to date.
func (a *app) run(c call) {
	// The two that work with nothing driving the menu. The menu shown when
	// oshelly is missing offers a reload, because nothing else re-reads once
	// it appears, and quit has to work in that state too.
	switch c.action {
	case panel.ActionQuit:
		log.Println("quitting")
		systray.Quit()
		return
	case panel.ActionReload:
		a.setStatus(panel.Idle)
		a.reload()
		return
	}

	a.mu.Lock()
	if a.busy || a.client == nil {
		a.mu.Unlock()
		return
	}
	a.busy = true
	client := a.client
	a.mu.Unlock()

	a.render()

	var status panel.Status
	switch c.action {
	case panel.ActionEdit:
		log.Println("opening the config")
		status = panel.ForEdit(client.Edit(context.Background()))
	default:
		log.Printf("launching %q", c.name)
		status = panel.ForRun(client.Run(context.Background(), c.name), c.name)
	}

	a.mu.Lock()
	a.busy = false
	a.mu.Unlock()

	a.setStatus(status)

	// Whatever happened, re-read: a command removed from the file must not
	// stay in the menu as though it were still launchable.
	a.reload()
}

func setEnabled(it *systray.MenuItem, enabled bool) {
	if enabled {
		it.Enable()
	} else {
		it.Disable()
	}
}
