<h1 align="center">Fiach</h1>

<p align="center">
  A general-purpose mod manager for any game.
</p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.26.4-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.26.4"></a>
  <a href="https://v3.wails.io/"><img src="https://img.shields.io/badge/Wails-v3-DF0000?style=for-the-badge&logo=wails&logoColor=white" alt="Wails v3"></a>
  <a href="https://github.com/phergul/fiach/releases"><img src="https://img.shields.io/github/v/release/phergul/fiach?style=for-the-badge&logo=github&logoColor=white&label=release" alt="Release"></a>
</p>

## Features

- **Library** - Scans your Steam libraries, browse installed games
- **Mods** - Import from archives (zip, 7z, rar) or plain folders
- **Profiles** - Per-game profiles with enable/disable and load order
- **Deployment** - Review planned file changes from mods with a tree view before applying or restoring. Inspect files, resolve drift and conflicts
- **Incremental apply** - Re-apply an already active profile when you change mods or load order
- **ReShade** - Detection, preset and content management, apply per game
- **OptiScaler** - Detection, management, apply per game

## Install

Pre-built binaries are published to [GitHub Releases](https://github.com/phergul/fiach/releases).

1. Open [GitHub Releases](https://github.com/phergul/fiach/releases) and download the artifact for your OS.
2. **Windows** - Run `fiach_windows_amd64_installer.exe`, or use `fiach_windows_amd64.exe` directly.
3. **macOS** - Download `fiach_darwin_arm64.zip` or `fiach_darwin_amd64.zip` (same universal app), unzip, and open `fiach.app`.
4. **Linux** - Use `fiach_linux_amd64`, the AppImage, `.deb`, or `.rpm` as you prefer.

Fiach can check for updates from **Settings → About → Check for updates**. The in-app updater installs `fiach_windows_amd64.exe`, `fiach_darwin_*.zip`, or `fiach_linux_amd64` depending on your platform.

## Usage

1. **Library** - Scan your Steam games.
2. **Game details** - Import mods, manage tags and metadata, and see which profile is applied.
3. **Profiles** - Create a profile, add mods, set load order and enabled state.
4. **Deployment** - Open deployment preview for a profile. Browse the file tree, inspect changes, and resolve any drift or conflicts. Apply when ready. Open it again later to incrementally update an already-applied profile with more changes.
5. **Restore** - From game details, restore vanilla to remove applied mods and return the game files back to default.
6. **ReShade / OptiScaler** - Select the correct executable, run the wizard to manage.

## Platform support

| Platform | Mod Management | ReShade / OptiScaler |
| -------- | -------------- | -------------------- |
| Windows  | ✅             | ✅                   |
| Linux    | ✅             | ❌                   |
| macOS    | ✅             | ❌                   |

## Development

Fiach is built with [Wails v3](https://v3.wails.io/) (Go backend, React frontend).

**Prerequisites:** Go 1.26.4+, Wails v3 CLI `v3.0.0-alpha2.108`, Bun 1.3.x. On Linux, install GTK4/WebKitGTK 6.0 development packages (`libgtk-4-dev`, `libwebkitgtk-6.0-dev`, and related build tools).

```bash
wails3 task dev      # development
wails3 task build    # production build → bin/
wails3 task package  # platform installer/package

# tests
go test ./...
cd frontend && bun install && bun run test
```

## License

See [LICENSE](LICENSE) - [THIRD PARTY NOTICES](THIRD_PARTY_NOTICES).
