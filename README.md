# oshelly-mac

Your shell commands in the macOS menu bar.

Click a name and it opens in your terminal. The list is the same
`~/.config/oshelly/oshelly.conf` the [Omarchy panel][panel] reads, written one
command per line:

```sh
# ~/.config/oshelly/oshelly.conf

Restart audio: systemctl --user restart pipewire wireplumber
Update system: omarchy update
Tail the journal: journalctl -f
```

The app is only a front end. It runs [oshelly-cli][cli], which owns the config
format and the terminal launch, so the two front ends cannot disagree about
what a name means or which terminal opens.

## The menu

```
──────────────────────────
Run a command
--
Restart audio
Update system
Tail the journal
--
Terminal: Ghostty  ▸
--
Edit config
Reload
Quit
```

Clicking a name closes the menu and starts the command. `Terminal` opens a
submenu with the system default and the common terminal applications, with a
tick on the saved choice. Choose Ghostty once and every command opens there.
Choose System default to clear the choice.
`Edit config` opens the config in the default text editor; the CLI creates the
file with a commented header if it does not exist yet. `Reload` re-reads the
config after you edit it; the app also re-reads after every launch, so a command
added in a terminal shows up the next time the menu opens.

When nothing is configured, the menu says so and names the file to edit. When
the config will not parse, it shows the line number and reason instead of
pretending the list is empty, and `Edit config` is still there to fix it.

A launch reports what happened on the first line:

```
Launched "Restart audio".
```

or, when it could not start:

```
Could not run "Update".
no terminal found: set OSHELLY_TERMINAL, or install a terminal this knows about.
```

The second line is oshelly's own reason, kept as a line of its own because a
menu item is one line and the reason is the actionable part.

## Install

### What you need first

| | What | Why | Required |
|---|---|---|---|
| CLI | [`oshelly`](https://github.com/h1st0ry3D/oshelly-cli) | Reads the config and opens the terminal | **Yes** |
| Build | Go 1.22 or newer | Building both of the above | To build |

The app needs no permission dialogs. It does not send HID reports or read the
screen; it starts a terminal, which `open` does for any `.command` file.

### 1. The CLI

```sh
git clone https://github.com/h1st0ry3D/oshelly-cli
cd oshelly-cli
go build -trimpath -o ~/.local/bin/oshelly ./cmd/oshelly
```

`/usr/local/bin` and `/opt/homebrew/bin` work too. The app looks in all three,
so it is found with no configuration. Check it:

```sh
oshelly doctor
oshelly list
```

### 2. The config

```sh
mkdir -p ~/.config/oshelly
$EDITOR ~/.config/oshelly/oshelly.conf
```

One command per line, `name: command`. A line starting with `#` is a comment.
A name can contain spaces; a name cannot contain a colon. Everything after the
first colon is the command and is passed to the terminal unchanged.

### 3. The app

Build it on the Mac:

```sh
git clone https://github.com/h1st0ry3D/oshelly-mac
cd oshelly-mac
./build/build.sh
```

That produces `build/Oshelly.app`, ad-hoc signed with an icon. Move it to
`/Applications` and open it:

```sh
mv build/Oshelly.app /Applications
open /Applications/Oshelly.app
```

The icon appears in the menu bar. There is no Dock icon and no window, because
it is a menu bar app (`LSUIElement`).

### Updating

```sh
cd ~/github/oshelly-cli
git pull
go build -trimpath -o ~/.local/bin/oshelly ./cmd/oshelly

cd ~/github/oshelly-mac
git pull
./build/build.sh
rm -rf /Applications/Oshelly.app
mv build/Oshelly.app /Applications
```

Quit and reopen the app after replacing the bundle. The app picks up the CLI
update the next time the menu launches something.

### Uninstall

Quit the app and delete it:

```sh
rm -rf /Applications/Oshelly.app
```

Your commands live in `~/.config/oshelly/oshelly.conf`, not in the bundle, so
they survive. Delete that file too if you want them gone.

## Config

`$XDG_CONFIG_HOME/oshelly/oshelly.conf`, which is
`~/.config/oshelly/oshelly.conf` unless you set it. On macOS this is the same
`~/.config/...` path the Linux panel uses, not `~/Library/Application Support`,
so one file works on both machines.

```sh
Restart audio: systemctl --user restart pipewire wireplumber
Screenshot: screencapture -i ~/Desktop/shot.png
Update: brew upgrade
```

`OSHELLY_CONFIG` overrides the path and takes either the file, which must end
in `.conf`, or the directory holding it.

The file is executable code, by design. Keep it writable only by you.

## Which terminal

The Terminal submenu saves an application name with `oshelly terminal`, for
example `Ghostty`. The CLI writes it per platform to
`~/.config/oshelly/settings.json` and passes it to `open -a`. That works for
Ghostty because `open -a Ghostty <script>` hands the `.command` file to
Ghostty's file handler, which runs it. The same command works from a terminal:

```sh
oshelly terminal Ghostty
```

With no saved choice, the CLI asks macOS for the default handler for `.command`
files through `open`. That is usually Terminal, and it is whatever you have
registered if you use iTerm or another terminal that claims the type.

`OSHELLY_TERMINAL` is a session override and wins over the saved choice. It is
passed to `open -a`:

```sh
OSHELLY_TERMINAL=iTerm oshelly run "Update"
```

A GUI app started from Finder does not inherit exports from `.zshrc`, so
setting it there does not reach this app. To set it for the app, use
`launchctl setenv OSHELLY_TERMINAL iTerm` and reopen the app; use
`launchctl unsetenv OSHELLY_TERMINAL` to remove it.

The terminal stays open when the command finishes, with a line saying its exit
status and then an interactive shell. A failed script has to stay on screen
long enough to read the error, so the terminal does not close on its own.

There is no working directory to inherit from a menu bar click. Commands start
in the terminal's default directory, which is usually `$HOME`. Use absolute
paths in the config when it matters.

## Which editor

`Edit config` runs `oshelly edit`, which asks macOS for the default text editor
with `open -t`. `OSHELLY_EDITOR` overrides it with an application name or bundle
path, which the CLI passes to `open -a`:

```sh
OSHELLY_EDITOR=BBEdit oshelly edit
```

Like `OSHELLY_TERMINAL`, it has to be set in the app's environment to affect a
menu item: use `launchctl setenv OSHELLY_EDITOR BBEdit` and reopen the app.

## Troubleshooting

### The menu says oshelly is not installed

It looked in `/usr/local/bin/oshelly`, `~/.local/bin/oshelly` and
`/opt/homebrew/bin/oshelly`. Build the CLI to one of those and choose
`Reload settings` in the menu; the app does not need restarting.

### Clicking a command does nothing

The menu shows oshelly's reason on the line under the status. The usual causes
are a renamed command (reopen the menu, it re-reads the file), no terminal
found (`oshelly doctor` names the one it would use), or a terminal that opened
and closed because the command itself exited.

### The app does not appear in the menu bar

Open it from `/Applications`, not from the build directory. If it is already
running, macOS does not start a second copy. Quit it from the menu first, or
remove it from the menu bar by quitting and reopening.

### The config was edited but the menu is old

Choose `Reload`. The app also re-reads after every launch, and the CLI reads
the file every time it is asked to run something.

## Design notes

- The app runs one program and owns none of the behaviour. It reads a list with
  `list --json`, draws it, and sends a name with `run`. The config parser and
  the terminal selection live in one place, so this app and the Omarchy panel
  cannot drift.
- The app builds an argv array and runs no shell. A name with a space or a
  quote cannot become a second command on the way to the CLI.
- The menu is built once and edited in place. systray adds items rather than
  replacing them, so a rebuild happens only when the commands gain, lose or are
  reordered. A status line changing is a title update, not a rebuild.
- The command runs in a terminal, not inside the app. A command that asks a
  question is an ordinary thing to launch from a menu, and the app has no
  window to ask it in.
- The app icon and the menu bar glyph are both the terminal prompt from
  JetBrainsMono Nerd Font, the same glyph the Omarchy panel uses.
- The menu bar icon is a template image: black with an alpha channel, so macOS
  inverts it for a dark menu bar and leaves it alone for a light one, with no
  second file.

## License

MIT

[cli]: https://github.com/h1st0ry3D/oshelly-cli
[panel]: https://github.com/h1st0ry3D/oshelly
