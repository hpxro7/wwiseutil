# bnkutil
`bnkutil` is a tool for manipulating Wwise SoundBank files (`.bnk` or `.nbnk`). It currently supports two modes:

__unpack__: This takes an input SoundBank file and writes all of the embedded `.wem` files to a directory.
[ww2ogg](https://github.com/hcs64/ww2ogg/releases) can then be used to convert the `.wem` files to a playable Ogg Vorbis format. 

__replace__: This takes an input SoundBank file, along with target `.wem` files and writes a new SoundBank with target wems replaced. All metadata stored within the SoundBank will be updated to support the replacement `.wem`s. Replacement `.wem` files are allowed to be larger or smaller than the original embedded `wem`. 

## Usage

You may run the following command to find a list of flags and shortcuts that the tool supports:
```bash
bnkutil.exe -h
```

### Unpack

To unpack a SoundBank file use the following command:

```bash
bnkutil.exe -unpack -bnkpath em118_vo.nbnk -output em118\
```

`-b(nkpath)` specifies the path to the `.bnk` or `.nbnk` file to unpack.

`-o(utput)` specifies the new directory that will be created to unpack the `.wem` files into.

### Replace

To replace the `wems` within a SoundBank file use the following command:

```bash
bnkutil.exe -replace -bnkpath em118_vo.nbnk -target wems\ -output newbank.nbnk
```
`-b(nkpath)` specifies the path to the `.bnk` or `.nbnk` file to use as a source; replacement wems will replace the wems inside of this SoundBank.

`-t(arget)` specifies a path to a directory that will be used to gather wem replacements. This directory will be searched for all compatible `.wem` files. Their filenames will be used as an index to the target wems to replace, starting at 1. For example, a `002.wem` file in this directory will replace the second wem in the original source SoundBank with the contents of that file. To file out what the indexes are for each `.wem` file in a SoundBank, first run the unpack command. These `.wem` files must not be manually padded; the tool will automatically add padding as needed.

`-o(utput)` specifies the path to the new SoundBank file to create, with replacement wems and updates applied.

## Converting `.wav` to `.wem`

Audiokinetic's [Wwise](https://www.audiokinetic.com/download/) tool can be used to convert any `.wav` of your choice into the `.wem` format for audio replacement. You can follow the instructions found in [this video](https://www.youtube.com/watch?v=39Oeb4GvxEc) to convert any `.wav` file into a `.wem`.

## Limitations

1. This software has not been thoroughly tested yet and isn't gaurenteed to work with all SoundBank files. Do [file a bug](https://github.com/hpxro7/bnkutil/issues/new) on this github page if you encounter a problem.
