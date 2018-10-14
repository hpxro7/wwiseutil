# wwiseutil
`wwiseutil` is a tool for manipulating Wwise SoundBank files (`.bnk` or `.nbnk`) and File Packages (`.pck` or `.npck`). It currently support the following features with both a GUI or command line tool:

* __unpacking__: An input SoundBank or File Package can be unpacked, writing all of the embedded `.wem` files to a directory.
[ww2ogg](https://github.com/hcs64/ww2ogg/releases) can then be used to convert the `.wem` files to a playable Ogg Vorbis format. 

* __replacing__: The `.wem` files within a source can be replaced. All metadata stored within the file will be updated to support the replacement `.wem`s. Replacement `.wem` files are allowed to be larger or smaller than the original embedded `wem`.

* __loop editing__: Currently, loop editing of basic sound effects is supported. Support for different looping mechanisms will be supported in the future. Loop editing is currently only supported in the GUI.

![screenshot](screenshot.PNG?raw=true)

## Resources
* [Command Line Usage](https://github.com/hpxro7/wwiseutil/wiki/Command-Line-Usage)
* [MH:W Audio Modding Instructions](https://github.com/hpxro7/wwiseutil/wiki/Modding-MH:W)

## Limitations

1. This software has not been thoroughly tested yet and isn't gaurenteed to work with all SoundBank or File Package files. Do [file a bug](https://github.com/hpxro7/bnkutil/issues/new) on this github page if you encounter a problem.
