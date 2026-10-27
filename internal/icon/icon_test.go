package icon

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

// The menu bar icon has to be square, and this is not cosmetic.
//
// systray forces every icon to 16 by 16 points before it hands it to AppKit:
//
//	[image setSize:NSMakeSize(16, 16)];
//
// That is a stretch, not a fit. A glyph rendered into a canvas that is wider
// than it is tall comes out squashed on screen.
func TestTheIconIsSquare(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(Terminal()))
	if err != nil {
		t.Fatalf("terminal.png is not a PNG: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != b.Dy() {
		t.Errorf("the icon is %dx%d, and systray would stretch it to a square", b.Dx(), b.Dy())
	}
}

// A template image is its alpha channel as far as macOS is concerned. Without
// one it cannot be drawn as a template, and it stays black on a dark menu bar
// where it cannot be seen at all.
func TestTheIconCarriesAlpha(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(Terminal()))
	if err != nil {
		t.Fatal(err)
	}
	if _, hasAlpha := img.(*image.NRGBA); !hasAlpha {
		t.Errorf("the icon decodes as %T, want a type with an alpha channel", img)
	}
}

// Something has to be drawn, and some of it has to be transparent, or the icon
// is either blank or a solid block.
func TestTheIconHasBothInkAndSpace(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(Terminal()))
	if err != nil {
		t.Fatal(err)
	}
	var opaque, clear int
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			switch {
			case a > 0x8000:
				opaque++
			case a == 0:
				clear++
			}
		}
	}
	if opaque == 0 {
		t.Error("the icon is entirely transparent")
	}
	if clear == 0 {
		t.Error("the icon is entirely opaque, so it would be a solid block")
	}
}

// The icon is embedded at build time, so a missing or unreadable file is a
// build error rather than something that shows up as an empty menu bar.
func TestTheIconIsEmbedded(t *testing.T) {
	if len(Terminal()) == 0 {
		t.Fatal("no icon bytes")
	}
	if _, err := png.DecodeConfig(bytes.NewReader(Terminal())); err != nil {
		t.Fatalf("the embedded icon is not a readable image: %v", err)
	}
}
