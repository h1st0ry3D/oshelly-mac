// Package cli talks to the oshelly program.
//
// This is the same contract the Omarchy panel uses: `list --json` to read the
// list of commands, and `run` to launch one. Nothing here parses the config or
// chooses a terminal, because those belong to oshelly and are the things two
// front ends must not disagree about.
//
// Everything in this package runs on any platform, so it is all testable
// without a Mac. The menu bar around it is not.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

// candidates are the directories the binary is looked for in, in order.
//
// The first two are the same ones the Omarchy panel uses, so one install
// serves both front ends. The third is where Homebrew puts things on a Mac.
// These are directories, not binary names: the name is appended.
var candidates = []string{
	"/usr/local/bin",
	"",
	"/opt/homebrew/bin",
}

// ErrMissing reports that the oshelly program is not installed, which is a
// different situation from oshelly reporting that a command could not run.
var ErrMissing = errors.New("oshelly is not installed")

// Timeout is the longest a single call may take.
//
// `run` starts a terminal and returns, so it is quick. `list` reads one small
// file. This is generous because a machine under load can be slow to fork a
// terminal, and a menu that says "did not answer" while the terminal is
// opening would be wrong.
const Timeout = 15 * time.Second

// Command is one entry from the config.
type Command struct {
	Name string `json:"name"`
	// Command is the shell text. It is carried for a UI that wants to show
	// what a row will run; the name is what this client launches.
	Command string `json:"command"`
	// Close says the terminal should close when this command finishes.
	Close bool `json:"close"`
}

// Settings is what `list --json` prints.
type Settings struct {
	Version  int    `json:"version"`
	Path     string `json:"path"`
	Exists   bool   `json:"exists"`
	Platform string `json:"platform"`
	// Terminal is the program or launcher in use here, for a label.
	Terminal string `json:"terminal"`
	// TerminalChoice is the saved choice for this Mac, or empty for the system
	// default. It is what the Terminal submenu ticks.
	TerminalChoice string    `json:"terminalChoice"`
	Commands       []Command `json:"commands"`
}

// Empty reports whether there is nothing to offer.
func (s Settings) Empty() bool { return len(s.Commands) == 0 }

// Client is a handle on the oshelly program.
type Client struct {
	// Path is the binary to run, once found.
	Path string
}

// New finds oshelly and returns a client for it, or ErrMissing.
func New() (*Client, error) {
	for _, p := range Candidates() {
		if info, err := os.Stat(p); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return &Client{Path: p}, nil
		}
	}
	return nil, fmt.Errorf("%w: looked in %s", ErrMissing, strings.Join(Candidates(), ", "))
}

// Candidates is every path New will look, in order, for a message that says
// where it looked.
func Candidates() []string {
	home := os.Getenv("HOME")
	if home == "" {
		if u, err := user.Current(); err == nil {
			home = u.HomeDir
		}
	}
	out := make([]string, 0, len(candidates))
	for _, dir := range candidates {
		if dir == "" {
			dir = filepath.Join(home, ".local", "bin")
		}
		out = append(out, filepath.Join(dir, "oshelly"))
	}
	return out
}

// Result is what one oshelly call said.
type Result struct {
	// Code is the exit code, which is the contract.
	Code int
	// Raw is stdout exactly as printed. List parses this, because flattening
	// JSON would also collapse whitespace inside its strings.
	Raw string
	// Message is the one line oshelly wrote for a person.
	Message string
	// Detail is what it wrote underneath, usually the reason a command could
	// not be launched.
	Detail string
}

// OK reports whether the call did what it was asked.
func (r Result) OK() bool { return r.Code == 0 }

// run calls oshelly and reads back both of its answers.
//
// The program and its arguments go across as an argv array with no shell
// between, so a name with spaces in it stays one argument and a name with a
// quote in it cannot become a second command. The environment is inherited
// rather than stripped: the CLI has to hand a terminal the session it belongs
// to, and the command in the config is the user's own code.
func (c *Client) run(ctx context.Context, args ...string) (Result, error) {
	if c == nil || c.Path == "" {
		return Result{}, ErrMissing
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.Path, args...)
	cmd.Env = environment()

	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf

	err := cmd.Run()
	raw := out.String()
	res := Result{
		Raw:     raw,
		Message: flatten(raw),
		Detail:  flatten(errBuf.String()),
	}
	if cmd.ProcessState != nil {
		res.Code = cmd.ProcessState.ExitCode()
	}
	if err != nil && res.Code == 0 {
		// Killed by a signal, or never started. ExitCode reports -1 there,
		// which is not an exit code oshelly ever uses.
		res.Code = -1
	}
	return res, nil
}

// environment is the session's own, with HOME guaranteed.
//
// A menu bar app started by a launch agent can have almost no environment. HOME
// is the one variable oshelly cannot do without: it is how the config file is
// found. Everything else is passed through so a terminal opened from here is
// in the same session as one opened from the Dock.
func environment() []string {
	env := os.Environ()
	if os.Getenv("HOME") == "" {
		// os.UserHomeDir reads HOME, so it cannot help when HOME is what is
		// missing. The account database can.
		if u, err := user.Current(); err == nil && u.HomeDir != "" {
			env = append(env, "HOME="+u.HomeDir)
		}
	}
	return env
}

// List reads the settings. A non-zero exit is an error rather than a Result,
// because there is no useful fallback and the caller has to show something.
func (c *Client) List(ctx context.Context) (Settings, error) {
	res, err := c.run(ctx, "list", "--json")
	if err != nil {
		return Settings{}, err
	}
	if res.Code != 0 {
		return Settings{}, fmt.Errorf("oshelly could not read its settings (exit %d): %s",
			res.Code, firstNonEmpty(res.Detail, res.Message))
	}
	var s Settings
	if err := json.Unmarshal([]byte(res.Raw), &s); err != nil {
		return Settings{}, fmt.Errorf("oshelly printed settings this does not understand: %w", err)
	}
	if s.Commands == nil {
		s.Commands = []Command{}
	}
	return s, nil
}

// Run asks oshelly to launch a named command.
func (c *Client) Run(ctx context.Context, name string) Result {
	res, err := c.run(ctx, "run", name)
	if err != nil {
		return Result{Code: -1, Message: err.Error(), Detail: err.Error()}
	}
	return res
}

// Edit asks oshelly to open the config in the default text editor.
func (c *Client) Edit(ctx context.Context) Result {
	res, err := c.run(ctx, "edit")
	if err != nil {
		return Result{Code: -1, Message: err.Error(), Detail: err.Error()}
	}
	return res
}

// SetClose records whether a command closes its terminal when it finishes.
func (c *Client) SetClose(ctx context.Context, name string, close bool) Result {
	word := "off"
	if close {
		word = "on"
	}
	res, err := c.run(ctx, "set", "close", name, word)
	if err != nil {
		return Result{Code: -1, Message: err.Error(), Detail: err.Error()}
	}
	return res
}

// SetTerminal records the terminal to use on this Mac. An empty name means the
// system default.
func (c *Client) SetTerminal(ctx context.Context, name string) Result {
	if name == "" {
		name = "default"
	}
	res, err := c.run(ctx, "terminal", name)
	if err != nil {
		return Result{Code: -1, Message: err.Error(), Detail: err.Error()}
	}
	return res
}

// flatten makes oshelly's output one line, because a menu item is one line.
func flatten(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.Join(strings.Fields(s), " ")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
