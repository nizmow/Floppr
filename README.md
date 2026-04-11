# Floppr

Create DOS floppy disk images from a directory. I write this mostly just to scratch an itch -- lots of DOS programs are just available as raw files, which means you have to mess around a bit to get an image to use with 86box or a Gotek. This makes it pretty easy.

It supports 360k, 720k, 1.2MB and 1.44MB disk image sizes. It does no normalisation of filenames and will fail fast if your input includes names that aren't DOS friendly, or if the directory is too big for the disk size.

It can also extract an existing floppy image back out to a host directory, including multiple images via a glob pattern.

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

```sh
floppr extract ./bootdisk.img ./bootdisk
```

```sh
floppr extract './disks/*.img' ./extracted
```

```sh
floppr extract *.img ./extracted
```

## Options

```sh
floppr create <source> [output] [--label LABEL]
```

```sh
floppr extract <source-or-glob>... <destination>
```

- `source`: directory to package into the floppy image
- `output`: optional output path, defaults to `<dirname>.img`
- `--format`, `-f`: floppy size in KB, one of `360`, `720`, `1200`, `1440`
- `--label`, `-l`: optional DOS volume label, defaults from the directory name and is truncated to 11 DOS-safe characters
- `source-or-glob`: one or more floppy image paths, or glob patterns such as `./disks/*.img`
- `destination`: target directory; for a single image the contents are extracted directly here, and for multiple images each image is extracted into its own subdirectory named after the image file

## Notes

- Writes a FAT12 floppy image in the selected format.
- Fails fast if the contents will not fit.
- Defaults output to `<dirname>.img`.
- Defaults label from the directory name.
- Requires DOS 8.3 filenames for now.
- Both quoted globs and shell-expanded globs work for `extract`.

Development and release notes live in [docs/development.md](/Users/nizmow/Code/Floppr/docs/development.md).
