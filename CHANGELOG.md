# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2022-02-20

- Enable dependabot
- Fix refresh access token request body [GH-44](https://github.com/mnencia/mchfuse/issues/44)

## [0.3.2] - 2021-01-30

- Switch to godaemon to demonize [GH-15](https://github.com/mnencia/mchfuse/issues/15)
- Store the version information inside the executable.
  The build version is printed in the log during the startup.
- Add -v/--version command line switches

## [0.3.1] - 2020-12-13

- Fix "operation not supported" error on write ([GH-18](https://github.com/mnencia/mchfuse/issues/18)
  and [GH-19](https://github.com/mnencia/mchfuse/issues/19))

## [0.3.0] - 2020-11-22

- Bump github.com/relvacode/iso8601 from 1.0.0 to 1.1.0
- Fix login error ([GH-17](https://github.com/mnencia/mchfuse/issues/17))
- Close file descriptors when going background (tentative fix for [GH-15](https://github.com/mnencia/mchfuse/issues/17))

## [0.2.0] - 2020-06-15

- Send the process in the background automatically
- Support mounting using `mount -t fuse.mchfuse` and `/etc/fstab`

## [0.1.0] - 2020-06-04

- Reuse http.Client to improve performances. ([GH-1](https://github.com/mnencia/mchfuse/issues/1))
- Detect whether the device is reachable locally

  When connecting to a device, mchfuse try to determine if it is locally
  reachable, otherwise it uses the external endpoint.

  The reachability check runs every 30 seconds (to allow transition from
  External to Internal endpoint in case the device become reachable
  locally) or when an API call returns a connection error (to allow a
  quick transition from Internal to External endpoint if the device
  become unreachable locally).
- Always use SSL to access a device

## [0.0.1] - 2020-04-30

- Initial release

[Unreleased]: https://github.com/mnencia/mchfuse/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/mnencia/mchfuse/releases/tag/v0.4.0
[0.3.2]: https://github.com/mnencia/mchfuse/releases/tag/v0.3.2
[0.3.1]: https://github.com/mnencia/mchfuse/releases/tag/v0.3.1
[0.3.0]: https://github.com/mnencia/mchfuse/releases/tag/v0.3.0
[0.2.0]: https://github.com/mnencia/mchfuse/releases/tag/v0.2.0
[0.1.0]: https://github.com/mnencia/mchfuse/releases/tag/v0.1.0
[0.0.1]: https://github.com/mnencia/mchfuse/releases/tag/v0.0.1
