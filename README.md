# My Cloud Home FUSE file system

MCHFuse is a FUSE file system for mounting [Western Digital My Cloud Home](https://www.mycloud.com) devices.

It exposes the main storage area of your device using the
[WD My Cloud Home Off-Device API](https://developer.westerndigital.com/develop/wd-my-cloud-home/api.html).

## Prerequisites

To compile MCHFuse, you need at least go 1.13.

To run MCHFuse on OSX, you need `osxfuse` extension. You can install it with Homebrew
using the  command:

``` sh
brew cask install osxfuse
```

## Installing the latest release

To quickly install the latest pre-built binary of MCHFuse, you can execute the following command:

``` sh
curl -sSfL https://github.com/mnencia/mchfuse/raw/master/install.sh | sudo sh -s -- -b /usr/local/bin
```

## Installing from source

* Install the Go compiler suite and make; e.g. on Ubuntu:

  ``` sh
  sudo apt-get install git golang-go ca-certificates make
  ```

* Then check out MCHFuse project

  ``` sh
  git clone https://github.com/mnencia/mchfuse.git
  ```

* Then change directory to the just-checked-out work tree and build it

  ``` sh
  cd mchfuse
  make
  ```

  After the build, you find a `mchfuse` executable in the project root.

* If you want to make `mchfuse` available as a system command, install it

  ``` sh
  sudo make install
  ```

## Quickstart

You can mount your device using the following command:

``` sh
cat > mchfuse.conf << 'EOF'
username = EMAIL
password = PASSWORD
EOF

chmod 600 mchfuse.conf

mchfuse -c mchfuse.conf DEVICE_NAME MOUNT_POINT
```

The `EMAIL` and `PASSWORD` are the ones used to access <https://home.mycloud.com/>.

The `DEVICE_NAME` is the name assigned to the device during the initial configuration.
If you happen to use a wrong name, the resulting error message contain the list
of valid discovered device names.

Replace `MOUNT_POINT` with the actual path where you want to see the content
of the device. (e.g. `/mnt/mydevice`)

After the last command, you should see the content of the mounted device available
in the mount point folder.

> **NOTE:** the filesystem will be only accessible from the user who executed
> the `mchfuse` command unless you specify the flag `--allow-other`
> either on the command line or in the configuration file (i.e. `allow-other = true`)

You can unmount the device using the usual `umount` command:

``` sh
umount MOUNT_POINT
```

## Usage

``` plain
Usage: mchfuse [flags] deviceName[:devicePath] mountpoint
  -c, --config string     config file path
  -u, --username string   mycloud.com username
  -p, --password string   mycloud.com password
  -a, --allow-other       allow other users
  -U, --uid int           set the owner of the files in the filesystem (default disabled)
  -G, --gid int           set the group of the files in the filesystem (default disabled)
  -f, --foreground        do not demonize
  -d, --debug             activate debug output (implies --foreground)
  -h, --help              display this help and exit
```

All the options can be specified in a configuration file with the format:

``` ini
flag-name = value
```

You can pass the configuration using the `--config` flag, otherwise `mchfuse`
loads the options from `/etc/mchfuse.conf` if it exists and is readable.

If you do not specify a UID or a GID, it inherits the missing setting from the
user that runs the command.

The `deviceName` is the name assigned to the device during the initial configuration.
If you happen to use a wrong name, the resulting error message contain the list of
valid discovered device names.

By default, MCHFuse mounts the root of the device, but you can append a `devicePath`
after a `:` separator, to start from a subdirectory.
(e.g. `myDevice:linuxData` uses the `linuxData` folder inside `myDevice` root)

The `mountpoint` is any directory accessible from the current user.
If the path doesn't exist, MCHFuse tries to create it.

> **NOTE:** `mchfuse` demonize itself, eventual errors raised by the background
> process will end up in the syslog with priority NOTICE and tag "mchfuse".

## Persistent Mounts

To keep your volume mounted on your system through reboots, create
a persistent mount. This is accomplished by updating
your system's `/etc/fstab` [file](https://wiki.archlinux.org/index.php/fstab).

### Update `fstab`

On a new line, add a mount directive to your `/etc/fstab` file which matches
the following syntax:

```
deviceName[:devicePath] mountpoint fuse.mchfuse noauto,x-systemd.automount,_netdev,allow_other 0 0
```

You can specify any option available on the command line adding it in the field 
containing `noauto,x-systemd.automount,_netdev,allow_other` separated by commas.
Please avoid using explicit `username` and `password` parameters, because
they will be readable both in `/etc/fstab` file and in the system process list.
Use the default configuration file `/etc/mchfuse.conf` or specify one 
using `config` option instead.

> **NOTE** You will need to use sudo privileges to edit this file from your
> limited user.

After setting the line in `/etc/fstab`, reboot your system.
Then, list the contents of the mounted directory. You should see the content
of your device.

## Maturity

This project is in alpha state. I've made it to access my device from Linux,
and it works quite well for me. There are many things to improve, starting
from performances.

## Known Limits

* Write performances need improvements

## TODO

* Performance tests
* Device list command
* Support for extended attributes
* More documentation
* More testing

## Feedback

You can send your feedback through the [issue tracker](https://github.com/mnencia/mchfs)

## License

Copyright 2020 Marco Nenciarini <mnencia@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  <http://www.apache.org/licenses/LICENSE-2.0>

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## Disclaimer

I'm not affiliated in any way with Western Digital.
