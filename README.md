# My Cloud Home FUSE file system

MCHFuse is a FUSE file system for mounting [Western Digital My Cloud Home](https://www.mycloud.com) devices.

It exposes the main storage area of your device using the
[WD My Cloud Home Off-Device API](https://developer.westerndigital.com/develop/wd-my-cloud-home/api.html).

## Installing

* Install the Go compiler suite and make; e.g. on Ubuntu:

  ``` sh
  sudo apt-get install golang-go ca-certificates make
  ```

* Then check out MCHFuse project, and run

  ``` sh
  make
  ```

  After the build, you find a `mchfuse` executable in the project root.

## Quickstart

You can mount your device using the following command:

``` sh
cat > mchfuse.conf < 'EOF'
username = YOURUSERNAME
password = PASSWORD
EOF

chmod 600 mchfuse.conf

./mchfuse -c mchfuse.conf DEVICE_NAME[:device/path] /mount/point &
```

You can specify all the command line options in a config file with the following format:

``` sh
umount /mount/point
```

## Usage

``` plain
Usage: mchfuse [flags] deviceName:devicePath mountpoint
  -a, --allow-other       allow other users
  -c, --config string     config file path
  -d, --debug             activate debug output
  -G, --gid int           set the group of the files in the filesystem (default -1)
  -p, --password string   mycloud.com password
  -U, --uid int           set the owner of the files in the filesystem (default -1)
  -u, --username string   mycloud.com username
```

All the options can be specified in a config file with the format

``` ini
flag-name = value
```

If you do not specify a UID and a GID, it inherits them from the context.

## Maturity

This project is in alpha state. I've made it to access my device from Linux,
and it works quite well for me. There are many things to improve starting
from performances.

## Known Limits

* For the moment, you must be on the same network of the device

## TODO

* Device locality detection to allow accessing the device remotely
* Mount helper, to support mounting from `/etc/fstab`
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
