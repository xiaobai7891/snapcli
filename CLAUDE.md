# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SnapCLI is a cross-platform screenshot utility for CLI AI tools. It captures screenshots with one hotkey and automatically copies the file path to clipboard for easy pasting into terminal.

## Build Commands

```bash
# Windows (primary)
build.bat              # Quick build for Windows
make build-windows     # Windows x64 GUI (no console window)

# macOS
make build-darwin-amd64   # Intel Mac
make build-darwin-arm64   # Apple Silicon

# All platforms
make build-all
```

Build output goes to `build/` directory.

## Architecture

```
cmd/snapcli/main.go          # Entry point, workflow orchestration
internal/
  capture/                   # Screenshot capture
    capture.go               # Interfaces (Capturer, Selector, Region, Display)
    capture_windows.go       # Windows: GDI32 BitBlt + CreateDIBSection
    capture_darwin.go        # macOS: screencapture command (incomplete)
    selector_windows.go      # Selection UI: Win32 popup window
  clipboard/                 # Path copying
    clipboard_windows.go     # Windows API: GlobalAlloc + SetClipboardData
  config/                    # JSON config at %APPDATA%/snapcli/config.json
  hotkey/                    # Global hotkey via golang.design/x/hotkey
  notify/                    # Toast notifications via go-toast
  storage/                   # PNG/JPEG saving with timestamp filenames
  tray/                      # System tray via getlantern/systray
```

## Key Patterns

**Platform-specific code**: Uses Go build tags (`//go:build windows`). Each module has interface + platform implementations.

**Windows API calls**: Direct syscall to user32.dll, gdi32.dll, kernel32.dll. No CGO or heavy frameworks.

**Screenshot workflow** (in `onHotkeyPressed()`):
1. Capture full screen via BitBlt → memory DIB
2. Show selector popup for region/window selection
3. Crop selected region
4. Save to disk with timestamp filename
5. Copy path to clipboard
6. Show notification

## Important Implementation Details

**Color format**: Windows GDI uses BGRA, Go image uses RGBA. Conversion happens in `capture_windows.go:CaptureRegion()`.

**Selector rendering**: Uses pre-computed dark/bright pixel arrays for overlay effect. Caches DIB section for performance.

**Window detection**: `WindowFromPoint` → `GetAncestor(GA_ROOT)` → filter desktop classes (Progman, WorkerW, Shell_TrayWnd).

## Configuration

Default hotkey: `alt+1`
Config file: `%APPDATA%\snapcli\config.json` (Windows), `~/.config/snapcli/config.json` (macOS)

```bash
snapcli --set-hotkey ctrl+shift+x   # Change hotkey
snapcli --config                     # Show config path
```

## Known Limitations

- macOS `capture_darwin.go` and selector are incomplete stubs
- `playSound` and `autoStart` config options not implemented
