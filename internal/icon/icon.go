// Package icon holds the menu bar glyph.
//
// A glyph rather than the character it stands for, because the menu bar draws
// its title in the system font and a private-use codepoint is a box in a font
// that does not have it. Rendering it here means the icon looks the same on a
// machine with no Nerd Font installed at all.
package icon

import _ "embed"

// terminal is fa-terminal, U+F120 in JetBrainsMono Nerd Font, which is the
// same glyph the Omarchy panel puts in the bar, so the two front ends are
// recognisably the same thing.
//
//go:embed terminal.png
var terminal []byte

// Terminal is the PNG to hand to systray. It is black with an alpha channel,
// which is what makes it a template image: macOS inverts it for a dark menu
// bar and leaves it alone for a light one, without a second file.
func Terminal() []byte { return terminal }
