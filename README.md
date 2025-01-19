<h1>
    <img src="docs/pumping_station.svg" style="vertical-align:middle; width:8%" align="absmiddle"/>
    <span style="vertical-align:middle;">&nbsp;&nbsp;Simple sideloading tool for Kindle devices</span>
</h1>

[![GitHub Release](https://img.shields.io/github/release/rupor-github/sync2kindle.svg)](https://github.com/rupor-github/sync2kindle/releases)

### Purpose
This is CLI tool for day-to-day synchronization of kindle books between local
directory and directory on device over the wire - using either MTP or old USBMS
mount.

It was created to support day-to-day side loading usage scenario (based on my
multi-year experience owning various Kindle devices):

I have one or more local directories containing books in Kindle-supported
formats, possibly organized into subdirectories by authors or genres for easier
navigation. I would like to run a single command (not a tool with a UI or
additional complexity) from the terminal or console to send these books to my
device, while preserving the original directory structure.

Later, I may add new books to the local directories. At the same time, as I
finish reading books on the device, they may be removed there. When I run the
tool again, I want these changes to be synchronized bidirectionally: new or
updated books should be sent to the device, and completed (and deleted) books
should be removed locally.

The tool should maintain a history of actions performed. If a book is added to
the device outside this process, it should be ignored by the tool and left
untouched. Similarly, any additional directories or files created by the device
(e.g., Kindle-generated files) should not be affected.

The tool should have a minimal number of options and be simple to use. It
should support synchronization from the same local directory to multiple target
devices. The history it maintains should be per device and per target directory
on the device, allowing different target directories on the same device to be
synchronized at different intervals (e.g., syncing "fiction" frequently and
"nonfiction" less often). Simplicity and reliability should take priority over
performance and added flexibility.

### Installation:

Download from the [releases page](https://github.com/rupor-github/sync2kindle/releases) and unpack it in a convenient location.
You could use following public key and [minisign](https://github.com/jedisct1/minisign) tool to verify the authenticity of the release:

<p>
    <img src="docs/build_key.svg" style="vertical-align:middle; width:15%" align="absmiddle"/>
    <span style="vertical-align:middle;">&nbsp;&nbsp;RWTNh1aN8DrXq26YRmWO3bPBx4m8jBATGXt4Z96DF4OVSzdCBmoAU+Vq</span>
</p>

### Usage

To see details run any command with --help or -h.
```
EBooks> ./s2k
NAME:
   s2k - synchronizing local books with supported kindle device over MTP protocol or USBMS mount

USAGE:
   s2k [global options] command [command options]

VERSION:
   <<<will be current version>>>

COMMANDS:
   mtp         Synchronizes books between local source and target device over MTP protocol
   usb         Synchronizes books between local source and target device using USBMS mount
   history     Lists details for local history files
   dumpconfig  Dumps either default or active configuration (YAML)

GLOBAL OPTIONS:
   --config FILE, -c FILE  load configuration from FILE (YAML)
   --debug, -d             changes program behavior to help troubleshooting (default: false)
   --help, -h              show help
   --version, -v           print the version
```

**Or** to see default or currently active configuration run `s2k [--config <configuration file>] dumpconfig [--help] [--dry-run]`:

```
EBooks> ./s2k dumpconfig -h
NAME:
   s2k dumpconfig - Dumps either default or active configuration (YAML)

USAGE:
   s2k dumpconfig [command options] DESTINATION

OPTIONS:
   --dry-run   output active configuration to be used in actual operations, including values from --config file (default: false)
   --help, -h  show help

DESTINATION:
    file name to write configuration to, if absent - STDOUT

Produces file with default configuration values.
To see actual "active" configuration use dry-run mode.
```

**Or** to synchronize files use `s2k [--config <configuration file>] usb|mtp [--dry-run]`:

```
EBooks> ./s2k mtp -h
NAME:
   s2k mtp - Synchronizes books between local source and target device over MTP protocol

USAGE:
   s2k mtp [command options]

OPTIONS:
   --ignore-device-removals, -i  do not respect books removals on the device (default: false)
   --dry-run                     do not perform any actual changes (default: false)
   --help, -h                    show help

Using MTP protocol syncronizes books between 'source' local directory and 'target' path on the device.
Both could be specified in configuration file, otherwise 'source' is current working directory and 'target' is "documents/mybooks".
Kindle device is expected to be connected at the time of operation.

When 'ignore-device-removals' flag is set, books removed from the device are not removed from the local source.
```
and

```
EBooks> ./s2k usb -h
NAME:
   s2k usb - Synchronizes books between local source and target device using USBMS mount

USAGE:
   s2k usb [command options]

OPTIONS:
   --ignore-device-removals, -i  do not respect books removals on the device (default: false)
   --dry-run                     do not perform any actual changes (default: false)
   --unmount, -u                 Attempts to prepare device for safe disconnect (default: false)
   --help, -h                    show help

Using device storage mounted over USB syncronizes books between 'source' local directory and 'target' path on the device.
Both could be specified in configuration file, otherwise 'source' is current working directory and 'target' is "documents/mybooks".
Kindle device is expected to be mounted at the time of operation.

When 'ignore-device-removals' flag is set, books removed from the device are not removed from the local source.

With 'unmount' flag set, attempt is made to safely unmount storage after sync operation. Has no effect with 'dry-run'.
Results of this flag are very OS dependent, for example on Windows it may fail if not all buffers have been yet written
to storage and will fail if something still have device opened, on Linux it requires admin priviliges and will only
unmount filesystem after mount seases to be busy, etc. Since this is command line tool this flag mostly makes sense
on Windows, where standard way of unmounting USB media from the command line has been missing for years. On Linux
you could simply use 'eject' or 'udisksctl' commands.
```
**Or** to see what history has been accumulated use `s2k [--config <configuration file>] history`:

```
EBooks> ./s2k history -h
NAME:
   s2k history - Lists details for local history files

USAGE:
   s2k history [command options]

OPTIONS:
   --help, -h  show help

Lists local history databases specifying details for each of them.
```

Logging output levels, both in terminal and file are configurable independently (see "configuration") below. 

### Configuration

Configuration file is in YAML format and is fully described [here](https://github.com/rupor-github/sync2kindle/blob/main/config/config.yaml.tmpl)

Basically only values you need to define are "source" - one of your local books
directories and "target" - place on device which will be used for
synchronization. The rest is rarely needed.

I suggest having multiple configurations - per device and "target" directory,
rather than attempting to send and keep in sync humongous libraries all at
once. Main reason is rather obvious: Kindle storage is slow.

Synchronization logic is fully defined [at the source](https://github.com/rupor-github/sync2kindle/blob/main/sync/prepare.go).

### Troubleshooting

If you need help: there is "--debug" switch which will produce a zipped file
with details, hopefully sufficient for analysis. Its name and location could be
set in configuration file. Reproduce the problem in debug mode, create an
[issue](https://github.com/rupor-github/sync2kindle/issues) with description
and share the report.

I tried to be as careful as possible, but working USB and MTP devices on
different platforms is not straightforward.

### Supported platforms and devices

Kindle devices which mount as USBMS storage (**everything before latest Kindle
Scribe, Paperwhite 12 or Colorsoft**) are supported with **USB** subcommand (tested
with PW2, PW10 and Voyage) and later ones (**Scribe, Colorsoft and latest
Paperwhite**) are supported by **MTP** subcommand (tested with PW12).

At the moment program is built for Windows x64 and Linux x64. That all I have
access to. It was tested on fresh Windows 11 and KUbuntu 24.04 but should work on
most 64 bit Windows and Linuxes supported by current GoLang.

I tried to structure source code in such a way that it should be easy to port
to other Windows or Linux architectures and it could be relatively simple to
add drivers to support Darwin architectures too. Synchronization logic code is
platform independent.

Windows build does not require CGO at all, but COM support is very platform
dependent.

Linux build is using CGO and libmtp (which should also work for Darwin) but USB
discovery is very OS specific.

If you have a need to support something I have no way of supporting - say any
Macs, take a look at sources and drop a PR. We could work together to
incorporate your changes.

### TODO

- Add thumbnail support for KFX files
- Expand history reports with some useful statistics
