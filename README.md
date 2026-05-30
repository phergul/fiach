# Fiach

Fiach is a Wails v3 desktop app for managing game mods.

## Development

Run the app in development mode:

```powershell
wails3 dev
```

Build the app for the current platform:

```powershell
wails3 task build
```

Build outputs are written to `bin/`.

## Windows Builds

Build the Windows executable:

```powershell
wails3 task windows:build
```

Create the default unsigned NSIS installer:

```powershell
wails3 task windows:package
```

The Windows executable is written to `bin/fiach.exe`. The NSIS installer is written to `bin/fiach-<arch>-installer.exe`.

To create an unsigned MSIX package instead:

```powershell
wails3 task windows:package FORMAT=msix
```

The MSIX package is written to `bin/fiach-<arch>.msix`.

## Releases

The GitHub release workflow builds unsigned artifacts for Windows, macOS, and Linux, then creates a draft GitHub Release.

Create a release from a tag:

```powershell
git tag v0.1.0
git push origin v0.1.0
```

You can also run the `Release` workflow manually in GitHub Actions and provide a version such as `0.1.0`. Manual runs create or update the matching `v<version>` tag at the selected commit.

Release artifacts are uploaded to the draft release for review before publishing.
