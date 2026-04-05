# Floppr

Create DOS 1.44MB floppy disk images from a directory.

## Usage

```sh
floppr create ./mydir ./disk.img --label MYDISK
```

## Notes

- Writes a 1.44MB FAT12 floppy image.
- Fails fast if the contents will not fit.
- Requires DOS 8.3 filenames for now.

Development and release notes live in [docs/development.md](/Users/nizmow/Code/Floppr/docs/development.md).
