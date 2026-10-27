#!/usr/bin/env bash
# Build Oshelly.app.
#
# macOS treats a menu bar app differently from a plain binary: without a bundle
# it gets a Dock icon, a generic name, and no stable identity. This makes a
# bundle, with an ad-hoc signature so a rebuild does not change its identity
# more than it has to.
#
# Run this on the Mac. Nothing here works on Linux, because the bundle format
# and the toolchain are both macOS's.

set -euo pipefail

cd "$(dirname "$0")/.."

APP="build/Oshelly.app"
BIN="oshelly-mac"

if ! command -v go >/dev/null; then
	echo "go is not on PATH. Install Go 1.22 or newer." >&2
	exit 1
fi

echo "==> checking oshelly is where this will look for it"
found=""
for p in /usr/local/bin/oshelly "$HOME/.local/bin/oshelly" /opt/homebrew/bin/oshelly; do
	if [ -x "$p" ]; then
		found="$p"
		break
	fi
done
if [ -z "$found" ]; then
	echo "oshelly is not installed, so this will start and say so." >&2
	echo "    Build it from https://github.com/h1st0ry3D/oshelly-cli" >&2
else
	echo "    found $found"
fi

echo "==> running the tests"
# These run anywhere, including here, and cover everything except the drawing.
go test ./internal/...

echo "==> building"
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

# cgo is required: systray talks to AppKit through it.
CGO_ENABLED=1 GOOS=darwin go build -trimpath -ldflags "-s -w" \
	-o "$APP/Contents/MacOS/$BIN" ./cmd/oshelly-mac

# The icon, from the one master image. icns is a container of ten sizes and
# only macOS can write it, so this stays here rather than in the repository.
echo "==> building the icon"
if ! command -v iconutil >/dev/null || ! command -v sips >/dev/null; then
	echo "    sips or iconutil is missing, which should not happen on macOS." >&2
	echo "    The app will build without an icon." >&2
else
	iconset="$(mktemp -d)/AppIcon.iconset"
	mkdir -p "$iconset"
	for size in 16 32 128 256 512; do
		# Each of these is a point size and its retina double, which is why
		# 32, 128 and 512 appear twice: once as themselves and once as the
		# double of the size below.
		sips -z "$size" "$size" build/AppIcon.png --out "$iconset/icon_${size}x${size}.png" >/dev/null
		double=$((size * 2))
		sips -z "$double" "$double" build/AppIcon.png --out "$iconset/icon_${size}x${size}@2x.png" >/dev/null
	done
	iconutil -c icns "$iconset" -o "$APP/Contents/Resources/AppIcon.icns"
	rm -rf "$(dirname "$iconset")"
fi

cat > "$APP/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key>
	<string>Oshelly</string>
	<key>CFBundleDisplayName</key>
	<string>Oshelly</string>
	<key>CFBundleExecutable</key>
	<string>oshelly-mac</string>
	<key>CFBundleIdentifier</key>
	<string>com.github.h1st0ry3d.oshelly-mac</string>
	<key>CFBundleIconFile</key>
	<string>AppIcon</string>

	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>1.0.0</string>
	<key>CFBundleVersion</key>
	<string>1</string>
	<key>LSMinimumSystemVersion</key>
	<string>11.0</string>

	<!-- A menu bar app: no Dock icon, no window. -->
	<key>LSUIElement</key>
	<true/>

	<!-- Without this the icon and any text render blurry on a Retina display. -->
	<key>NSHighResolutionCapable</key>
	<true/>
</dict>
</plist>
PLIST

# Ad-hoc signed, so the bundle has one identity between a build and the next.
# There are no TCC grants for this app to lose, but an unsigned rebuild is
# still a different app to Launch Services.
echo "==> signing (ad-hoc)"
codesign --force --sign - --timestamp=none "$APP" 2>/dev/null \
	|| echo "    codesign not available; the app will run without a signature"

echo
echo "==> done: $APP"
echo
echo "Move it to /Applications and open it. The icon appears in the menu bar."
echo "If it says oshelly is not installed, build the CLI and put it in"
echo "~/.local/bin or /usr/local/bin, then choose Reload from the menu."
