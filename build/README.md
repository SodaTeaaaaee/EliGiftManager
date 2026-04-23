# Build Directory

The `build` directory contains Wails packaging assets and generated desktop binaries.

## Repository Convention

- `build/windows` and `build/darwin` stay under version control because they affect installer metadata and runtime packaging.
- `build/bin` is generated output and stays ignored by git.
- Generated binaries are useful for local verification, but they do not imply that the current UI is stable. The desktop UI is still in a prototype stage and is expected to change significantly.

## Structure

- `bin`: packaged output directory
- `darwin`: macOS-specific metadata
- `windows`: Windows-specific metadata and installer assets

## macOS

The `darwin` directory holds files used during macOS builds. If you want to restore the default generated state, delete the customized files and run `wails build` again.

- `Info.plist`: main plist used by `wails build`
- `Info.dev.plist`: development plist used by `wails dev`

## Windows

The `windows` directory contains the manifest, icon, and installer resources used during `wails build`. These files are committed because they affect the packaged application.

- `icon.ico`: application icon used by `wails build`
- `installer/*`: NSIS installer resources
- `info.json`: application metadata surfaced by the Windows installer and executable properties
- `wails.exe.manifest`: main application manifest
