# Development

## Local

```sh
mise install
mise run fmt
mise run test
mise run build
mise run run -- create ./mydir ./disk.img
```

## GitHub Actions

- [build.yml](/Users/nizmow/Code/Floppr/.github/workflows/build.yml) builds binaries from `HEAD` for Linux, macOS, and Windows and uploads workflow artifacts.
- [release.yml](/Users/nizmow/Code/Floppr/.github/workflows/release.yml) publishes a GitHub Release when a `v*` tag is pushed.

## Releases

```sh
git tag v0.1.0
git push origin v0.1.0
```

That creates a GitHub Release for the tagged commit and uploads packaged binaries for:

- Linux `amd64`
- macOS `amd64`
- macOS `arm64`
- Windows `amd64`
