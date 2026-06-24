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

- **Library** - Scan Steam libraries and browse installed games
- **Mods** - Import from archives (zip, 7z, rar) or plain folders
- **Profiles** - Per-game profiles with enable/disable and load order
- **Apply** - Preview and apply a profile, or restore vanilla state
- **ReShade** - Detection, preset and content management, apply per game
- **OptiScaler** - Detection, management, apply per game

## Install

Pre-built binaries are published to [GitHub Releases](https://github.com/phergul/fiach/releases).

1. Open [GitHub Releases](https://github.com/phergul/fiach/releases) and download the artifact for your OS.
2. **Windows** - Run the installer, or use the portable `.exe` directly.
3. **macOS** - Unzip and open `fiach.app`.
4. **Linux** - Use the AppImage, `.deb`, `.rpm`, or standalone binary as you prefer.

## Usage

1. **Library** - Scan your Steam games.
2. **Game details** - Import mods and manage them using tags and metadata.
3. **Profiles** - Create a profile, add mods, set load order and enabled/disable.
4. **Apply** - Preview planned changes (files/directories created/replaced), apply the profile to the game install, later restore files to vanilla to remove mods.
5. **ReShade / OptiScaler** - Select the correct executable, run the wizard to manage.

## Platform support

| Platform | Mod Management | ReShade / OptiScaler |
|----------|------------------|----------------------|
| Windows | ✅ | ✅ |
| Linux | ✅ | ❌ |
| macOS | ✅ | ❌ |

## Development

Fiach is built with [Wails v3](https://v3.wails.io/) (Go backend, React frontend).

**Prerequisites:** Go 1.26.4+, Wails v3 CLI `v3.0.0-alpha.77`, Bun 1.3.x. On Linux, install GTK/WebKit development packages (`libgtk-3-dev`, `libwebkit2gtk-4.1-dev`, and related build tools).@

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
