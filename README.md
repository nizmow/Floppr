# Floppr

`floppr` creates DOS 1.44MB floppy disk images from a source directory.

Current behavior:

- Creates a FAT12 floppy image sized to 1,474,560 bytes.
- Fails before writing if the directory tree will not fit in FAT12 data space.
- Fails if the root directory would exceed FAT12's 224-entry limit.
- Accepts only DOS 8.3 filenames and DOS-safe volume labels for now.

## Requirements

- `mise`

## Setup

```sh
mise install
```

## Usage

```sh
mise run build
./bin/floppr create ./mydir ./disk.img --label MYDISK
```

## GitHub Builds

GitHub Actions builds binaries from `HEAD` for:

- Linux `amd64`
- macOS `amd64`
- macOS `arm64`
- Windows `amd64`

Artifacts are uploaded on pushes to `main` or `master`, and you can also trigger the workflow manually from the Actions tab.

## Tasks

```sh
mise run fmt
mise run test
mise run build
mise run run -- create ./mydir ./disk.img
```
