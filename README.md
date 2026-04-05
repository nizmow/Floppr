# Floppr

Create DOS 1.44MB floppy disk images from a directory.

## Usage

```sh
floppr create ./mydir
```

```sh
floppr create ./mydir ./bootdisk.img
```

```sh
floppr create ./mydir --label MYDISK
```

```sh
floppr create ./mydir --format 720
```

## Options

```sh
floppr create <source> [output] [--label LABEL]
```

- `source`: directory to package into the floppy image
- `output`: optional output path, defaults to `<dirname>.img`
- `--format`, `-f`: floppy size in KB, one of `360`, `720`, `1200`, `1440`
- `--label`, `-l`: optional DOS volume label, defaults from the directory name and is truncated to 11 DOS-safe characters

## Notes

- Writes a FAT12 floppy image in the selected format.
- Fails fast if the contents will not fit.
- Defaults output to `<dirname>.img`.
- Defaults label from the directory name.
- Requires DOS 8.3 filenames for now.

Development and release notes live in [docs/development.md](/Users/nizmow/Code/Floppr/docs/development.md).
