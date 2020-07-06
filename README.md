<!-- omit in toc -->
# check-cert

Go-based tooling to check/verify certs (e.g., as part of a Nagios service check)

[![Latest Release](https://img.shields.io/github/release/atc0005/check-cert.svg?style=flat-square)](https://github.com/atc0005/check-cert/releases/latest)
[![GoDoc](https://godoc.org/github.com/atc0005/check-cert?status.svg)](https://godoc.org/github.com/atc0005/check-cert)
![Validate Codebase](https://github.com/atc0005/check-cert/workflows/Validate%20Codebase/badge.svg)
![Validate Docs](https://github.com/atc0005/check-cert/workflows/Validate%20Docs/badge.svg)

<!-- omit in toc -->
## Table of Contents

- [Project home](#project-home)
- [Overview](#overview)
  - [check_certs](#check_certs)
  - [lscert](#lscert)
- [Features](#features)
- [Changelog](#changelog)
- [Requirements](#requirements)
  - [Building source code](#building-source-code)
  - [Running](#running)
- [Installation](#installation)
- [Configuration options](#configuration-options)
  - [Threshold calculations](#threshold-calculations)
  - [Command-line arguments](#command-line-arguments)
    - [`check_cert`](#check_cert)
    - [`lscert`](#lscert-1)
  - [Configuration file](#configuration-file)
- [Examples](#examples)
  - [`check_cert` Nagios plugin](#check_cert-nagios-plugin)
    - [OK results](#ok-results)
    - [WARNING results](#warning-results)
  - [`lscert` CLI tool](#lscert-cli-tool)
    - [OK results](#ok-results-1)
    - [WARNING results](#warning-results-1)
- [License](#license)
- [References](#references)

## Project home

See [our GitHub repo](https://github.com/atc0005/check-cert) for the latest code,
to file an issue or submit improvements for review and potential inclusion
into the project.

## Overview

This repo contains various tools used to monitor/validate certificates.

| Tool Name     | Status | Description                                                                            |
| ------------- | ------ | -------------------------------------------------------------------------------------- |
| `check_certs` | Alpha  | Nagios plugin used to monitor certificate chains                                       |
| `lscert`      | Alpha  | Small CLI app used to generate a summary of certificate metadata and expiration status |

### check_certs

Nagios plugin used to monitor certificate chains. In addition to the features
shared with `lscert`, this app also validates the provided hostname against
the certificate Common Name *or* one of the available SANs entries.

The output for this application is designed to provide the one-line summary
needed by Nagios for quick identification of a problem while providing longer,
more detailed information for use in email and Teams notifications
([atc0005/send2teams](https://github.com/atc0005/send2teams)).

### lscert

Small CLI tool to print a *very* basic summary of certificate metadata
provided by a remote service at the specified fully-qualified domain name
(e.g., www.github.com) and port (e.g., 443) or via a local certificate
"bundle" or standalone leaf certificate file

This tool is intended to quickly review the results of replacing a certificate
and/or troubleshoot why connections to a certificate-enabled service may be
failing.

## Features

- Two tools for validating certificates
  - `lscert` CLI tool
    - verify certificate used by specified service
    - verify local certificate "bundle" or standalone leaf certificate file
  - `check_cert` Nagios plugin
    - verify certificate used by specified service

- Check expiration of all certificates in the *provided* certificate chain for
  cert-enabled services
  - not expired
  - expiring "soon"
    - warning threshold
    - critical threshold

- Validate provided hostname against Common Name *or* one of the available
  SANs entries
  - the expected hostname can be supplied by the `--server` flag *or* the
    `--dns-name` flag

- Optional support for verifying SANs entries on a certificate against a
  provided list
  - if `SKIPSANSCHECKS` keyword is supplied as the value no SANs entry checks
    will be performed; this keyword is useful for defining a shared Nagios
    check command and service check where some hosts may not use a certificate
    which has SANs entries defined

- Detailed "report" of findings
  - certificate order
  - certificate type
  - status (OK, CRITICAL, WARNING)
  - SANs entries
  - serial number
  - issuer

- Optional generation of OpenSSL-like text output from target cert-enabled
  service or filename
  - thanks to the `grantae/certinfo` package

- Optional, leveled logging using `rs/zerolog` package
  - JSON-format output (to `stderr`)
  - choice of `disabled`, `panic`, `fatal`, `error`, `warn`, `info` (the
    default), `debug` or `trace`.

- Optional, user-specified timeout value for TCP connection attempt

- Go modules support (vs classic `GOPATH` setup)

## Changelog

See the [`CHANGELOG.md`](CHANGELOG.md) file for the changes associated with
each release of this application. Changes that have been merged to `master`,
but not yet an official release may also be noted in the file under the
`Unreleased` section. A helpful link to the Git commit history since the last
official release is also provided for further review.

## Requirements

The following is a loose guideline. Other combinations of Go and operating
systems for building and running tools from this repo may work, but have not
been tested.

### Building source code

- Go 1.13+
- GCC
  - if building with custom options (as the provided `Makefile` does)
- `make`
  - if using the provided `Makefile`

### Running

- Windows 7, Server 2008R2 or later
  - per official [Go install notes][go-docs-install]
- Windows 10 Version 1909
  - tested
- Ubuntu Linux 16.04, 18.04

## Installation

1. [Download][go-docs-download] Go
1. [Install][go-docs-install] Go
   - NOTE: Pay special attention to the remarks about `$HOME/.profile`
1. Clone the repo
   1. `cd /tmp`
   1. `git clone https://github.com/atc0005/check-cert`
   1. `cd check-cert`
1. Install dependencies (optional)
   - for Ubuntu Linux
     - `sudo apt-get install make gcc`
   - for CentOS Linux
     - `sudo yum install make gcc`
   - for Windows
     - Emulated environments (*easier*)
       - Skip all of this and build using the default `go build` command in
         Windows (see below for use of the `-mod=vendor` flag)
       - build using Windows Subsystem for Linux Ubuntu environment and just
         copy out the Windows binaries from that environment
       - If already running a Docker environment, use a container with the Go
         tool-chain already installed
       - If already familiar with LXD, create a container and follow the
         installation steps given previously to install required dependencies
     - Native tooling (*harder*)
       - see the StackOverflow Question `32127524` link in the
         [References](references.md) section for potential options for
         installing `make` on Windows
       - see the mingw-w64 project homepage link in the
         [References](references.md) section for options for installing `gcc`
         and related packages on Windows
1. Build binaries
   - for the current operating system, explicitly using bundled dependencies
         in top-level `vendor` folder
     - `go build -mod=vendor ./cmd/check_cert/`
     - `go build -mod=vendor ./cmd/lscert/`
   - for all supported platforms (where `make` is installed)
      - `make all`
   - for use on Windows
      - `make windows`
   - for use on Linux
     - `make linux`
1. Copy the newly compiled binary from the applicable `/tmp` subdirectory path
   (based on the clone instructions in this section) below and deploy where
   needed.
   - if using `Makefile`
     - look in `/tmp/check-cert/release_assets/check_cert/`
     - look in `/tmp/check-cert/release_assets/lscert/`
   - if using `go build`
     - look in `/tmp/check-cert/`

## Configuration options

### Threshold calculations

The behavior of the `check_cert`plugin differs somewhat from
`check_http` `v2.1.2`; this plugin triggers a whole day
*later* than `check_http` does for the same `WARNING` and
`CRITICAL` threshold values.

For example, if we use the default values of 30 days for `WARNING` threshold
and 15 days for the `CRITICAL` threshold:

1. The thresholds are calculated
    - `WARNING`: Now (exact time in UTC) + 30 days
    - `CRITICAL`: Now (exact timein UTC) + 15 days
1. The certificate expiration date is checked and the very first match (in
   order) determines the status of the service check
    1. if the certificate expires *before* the current time, the status is
       `EXPIRED`
    1. if the certificate expires *before* the CRITICAL threshold, the status
       is `CRITICAL`
    1. if the certificate expires *before* the WARNING threshold, the status
       is `WARNING`
    1. otherwise, the certificate is assumed to have a status of `OK`

No rounding is performed.

See GH-32 for additional info.

### Command-line arguments

- Use the `-h` or `--help` flag to display current usage information.
- Flags marked as **`required`** must be set via CLI flag.
- Flags *not* marked as required are for settings where a useful default is
  already defined, but may be overridden if desired.

#### `check_cert`

| Flag                 | Required | Default | Repeat | Possible                                                                | Description                                                                                                                                                                                                                                                                                                                                                           |
| -------------------- | -------- | ------- | ------ | ----------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `h`, `help`          | No       | `false` | No     | `h`, `help`                                                             | Show Help text along with the list of supported flags.                                                                                                                                                                                                                                                                                                                |
| `v`, `version`       | No       | `false` | No     | `v`, `version`                                                          | Whether to display application version and then immediately exit application.                                                                                                                                                                                                                                                                                         |
| `branding`           | No       | `false` | No     | `branding`                                                              | Toggles emission of branding details with plugin status details. This output is disabled by default.                                                                                                                                                                                                                                                                  |
| `c`, `age-critical`  | No       | 15      | No     | *positive whole number of days*                                         | The threshold for the certificate check's `CRITICAL` state. If the certificate expires before this number of days then the service check will be considered in a `CRITICAL` state.                                                                                                                                                                                    |
| `w`, `age-warning`   | No       | 30      | No     | *positive whole number of days*                                         | The threshold for the certificate check's `WARNING` state. If the certificate expires before this number of days, but not before the `age-critical` value, then the service check will be considered in a `WARNING` state.                                                                                                                                            |
| `ll`, `log-level`    | No       | `info`  | No     | `disabled`, `panic`, `fatal`, `error`, `warn`, `info`, `debug`, `trace` | Log message priority filter. Log messages with a lower level are ignored.                                                                                                                                                                                                                                                                                             |
| `p`, `port`          | No       | `443`   | No     | *positive whole number between 1-65535, inclusive*                      | TCP port of the remote certificate-enabled service. This is usually 443 (HTTPS) or 636 (LDAPS).                                                                                                                                                                                                                                                                       |
| `t`, `timeout`       | No       | `10`    | No     | *positive whole number*                                                 | Timeout value in seconds allowed before the connection attempt to a remote certificate-enabled service is abandoned and an error returned.                                                                                                                                                                                                                            |
| `se`, `sans-entries` | No       |         |        | *comma-separated list of values*                                        | One or many Subject Alternate Names (SANs) expected for the certificate used by the remote service. If provided, this list of comma-separated (optional) values is required for the certificate to pass validation. If the case-insensitive SKIPSANSCHECKS keyword is provided this validation will be skipped, effectively turning the use of this flag into a NOOP. |
| `s`, `server`        | **Yes**  |         |        | *fully-qualified domain name or IP Address*                             | The fully-qualified domain name or IP Address of the remote system whose cert(s) will be monitored. The value provided will be validated against the Common Name and Subject Alternate Names fields *unless* the `dns-name` flag is also specified, in which case *this* value is only used for making the initial connection.                                        |
| `dn`, `dns-name`     | No       |         |        | *fully-qualified domain name or IP Address*                             | The fully-qualified domain name of the remote system to be used for hostname verification. This option can be used for cases where the initial connection is made using a name or IP Address not associated with the certificate.                                                                                                                                     |

#### `lscert`

| Flag                 | Required | Default | Repeat | Possible                                                                | Description                                                                                                                                                                                                                                                                                                                                                           |
| -------------------- | -------- | ------- | ------ | ----------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `h`, `help`          | No       | `false` | No     | `h`, `help`                                                             | Show Help text along with the list of supported flags.                                                                                                                                                                                                                                                                                                                |
| `v`, `version`       | No       | `false` | No     | `v`, `version`                                                          | Whether to display application version and then immediately exit application.                                                                                                                                                                                                                                                                                         |
| `c`, `age-critical`  | No       | 15      | No     | *positive whole number of days*                                         | The threshold for the certificate check's `CRITICAL` state. If the certificate expires before this number of days then the service check will be considered in a `CRITICAL` state.                                                                                                                                                                                    |
| `w`, `age-warning`   | No       | 30      | No     | *positive whole number of days*                                         | The threshold for the certificate check's `WARNING` state. If the certificate expires before this number of days, but not before the `age-critical` value, then the service check will be considered in a `WARNING` state.                                                                                                                                            |
| `f`, `filename`      | No       | `false` | No     | *valid file name characters*                                            | Fully-qualified path to a file containing one or more certificates.                                                                                                                                                                                                                                                                                                   |
| `text`               | No       | `false` | No     | `true`, `false`                                                         | Toggles emission of x509 TLS certificates in an OpenSSL-inspired text format. This output is disabled by default.                                                                                                                                                                                                                                                     |
| `ll`, `log-level`    | No       | `info`  | No     | `disabled`, `panic`, `fatal`, `error`, `warn`, `info`, `debug`, `trace` | Log message priority filter. Log messages with a lower level are ignored.                                                                                                                                                                                                                                                                                             |
| `p`, `port`          | No       | `443`   | No     | *positive whole number between 1-65535, inclusive*                      | TCP port of the remote certificate-enabled service. This is usually 443 (HTTPS) or 636 (LDAPS).                                                                                                                                                                                                                                                                       |
| `t`, `timeout`       | No       | `10`    | No     | *positive whole number*                                                 | Timeout value in seconds allowed before the connection attempt to a remote certificate-enabled service is abandoned and an error returned.                                                                                                                                                                                                                            |
| `se`, `sans-entries` | No       |         |        | *comma-separated list of values*                                        | One or many Subject Alternate Names (SANs) expected for the certificate used by the remote service. If provided, this list of comma-separated (optional) values is required for the certificate to pass validation. If the case-insensitive SKIPSANSCHECKS keyword is provided this validation will be skipped, effectively turning the use of this flag into a NOOP. |
| `s`, `server`        | **Yes**  |         |        | *fully-qualified domain name or IP Address*                             | The fully-qualified domain name or IP Address of the remote system whose cert(s) will be monitored. The value provided will be validated against the Common Name and Subject Alternate Names fields *unless* the `dns-name` flag is also specified, in which case *this* value is only used for making the initial connection.                                        |
| `dn`, `dns-name`     | No       |         |        | *fully-qualified domain name or IP Address*                             | The fully-qualified domain name of the remote system to be used for hostname verification. This option can be used for cases where the initial connection is made using a name or IP Address not associated with the certificate.                                                                                                                                     |

### Configuration file

Not currently supported. This feature may be added later if there is
sufficient interest.

## Examples

### `check_cert` Nagios plugin

#### OK results

This example shows using the Nagios plugin to manually check a remote
certificate-enabled port on www.google.com. We override the default `WARNING`
and `CRITICAL` age threshold values with somewhat arbitrary numbers.

```ShellSession
.\check_cert.exe --server www.google.com --port 443 --age-critical 50 --age-warning 55
OK: leaf cert "www.google.com" expires next (on 2020-08-12 12:08:31 +0000 UTC)

**ERRORS**

* None

**DETAILED INFO**

Certificate 1 of 2 (leaf):
        Name: CN=www.google.com,O=Google LLC,L=Mountain View,ST=California,C=US
        SANs entries: [www.google.com]
        KeyID: D:94:9F:90:8A:5C:E:B5:B5:DB:B7:79:7F:6A:9:42:3A:4D:CC:D4
        Issuer: CN=GTS CA 1O1,O=Google Trust Services,C=US
        IssuerKeyID: 98:D1:F8:6E:10:EB:CF:9B:EC:60:9F:18:90:1B:A0:EB:7D:9:FD:2B
        Serial: 41166161160297429311704478035915443513
        Expiration: 2020-08-12 12:08:31 +0000 UTC
        Status: [OK] 66d 23h remaining

Certificate 2 of 2 (intermediate):
        Name: CN=GTS CA 1O1,O=Google Trust Services,C=US
        SANs entries: []
        KeyID: 98:D1:F8:6E:10:EB:CF:9B:EC:60:9F:18:90:1B:A0:EB:7D:9:FD:2B
        Issuer: CN=GlobalSign,OU=GlobalSign Root CA - R2,O=GlobalSign
        IssuerKeyID: 9B:E2:7:57:67:1C:1E:C0:6A:6:DE:59:B4:9A:2D:DF:DC:19:86:2E
        Serial: 149699596615803609916394524856
        Expiration: 2021-12-15 00:00:42 +0000 UTC
        Status: [OK] 556d 11h remaining
```

See the `WARNING` example output for additional details.

#### WARNING results

Here we do the same thing again, but using the expiration date values returned
earlier as a starting point, we intentionally move the threshold values in
order to trigger a `WARNING` state for the leaf certificate: if the leaf
certificate is good for 66 days and 23 hours more, we indicate that warnings
that should triggered once the cert has fewer than 67 days left.

```ShellSession
.\check_cert.exe --server www.google.com --port 443 --age-critical 50 --age-warning 67
{"level":"warn","version":"x.y.z","logging_level":"info","server":"www.google.com","port":443,"age_warning":67,"age_critical":50,"expected_sans_entries":"","error":"1 certificates expired or expiring","expiring_certs":1,"caller":"T:/github/check-cert/cmd/check_cert/main.go:218","message":"expired certs present in chain"}
WARNING: Invalid certificate chain for "www.google.com" [EXPIRED: 0, EXPIRING: 1, OK: 1]

**ERRORS**

* 1 certificates expired or expiring

**DETAILED INFO**

Certificate 1 of 2 (leaf):
        Name: CN=www.google.com,O=Google LLC,L=Mountain View,ST=California,C=US
        SANs entries: [www.google.com]
        KeyID: D:94:9F:90:8A:5C:E:B5:B5:DB:B7:79:7F:6A:9:42:3A:4D:CC:D4
        Issuer: CN=GTS CA 1O1,O=Google Trust Services,C=US
        IssuerKeyID: 98:D1:F8:6E:10:EB:CF:9B:EC:60:9F:18:90:1B:A0:EB:7D:9:FD:2B
        Serial: 41166161160297429311704478035915443513
        Expiration: 2020-08-12 12:08:31 +0000 UTC
        Status: [WARNING] 66d 23h remaining

Certificate 2 of 2 (intermediate):
        Name: CN=GTS CA 1O1,O=Google Trust Services,C=US
        SANs entries: []
        KeyID: 98:D1:F8:6E:10:EB:CF:9B:EC:60:9F:18:90:1B:A0:EB:7D:9:FD:2B
        Issuer: CN=GlobalSign,OU=GlobalSign Root CA - R2,O=GlobalSign
        IssuerKeyID: 9B:E2:7:57:67:1C:1E:C0:6A:6:DE:59:B4:9A:2D:DF:DC:19:86:2E
        Serial: 149699596615803609916394524856
        Expiration: 2021-12-15 00:00:42 +0000 UTC
        Status: [OK] 556d 11h remaining
```

Some items to note (in order of appearance):

1. JSON output providing structured logging information
   - this is sent to `stderr`
   - Nagios ignores `stderr` output from plugins; `stdout` is for Nagios,
     `stderr` is for humans
1. The one-line status output on the second line
   - this is used by Nagios for display in an overview view for all service
     checkout for a host
   - this is used by Nagios for text, email and whatever else notifications
     (if configured)
1. The `ERRORS` section notes briefly what is wrong with the cert
1. The `DETAILED INFO` section contains an overview of the certificate chain
   - this is used by Nagios for display on the detailed service check-specific
     page (e.g., shows last check time, frequency, current state, etc)
   - as for the one-line output, this is used by Nagios for text, email and
     whatever other notifications may be configured
1. The `Status` field for the leaf certificate changed from `OK` to `WARNING`
   and this plugin set the appropriate exit code to let Nagios know of the
   state change.

### `lscert` CLI tool

#### OK results

This example shows using the CLI app to perform the same initial check that we
performed earlier using the Nagios plugin.

```ShellSession
.\lscert.exe --server www.google.com --port 443 --age-critical 50 --age-warning 55

Connecting to remote server "www.google.com" at port 443


======================
CERTIFICATES | SUMMARY
======================

- OK: 2 certs found for service running on www.google.com at port 443
- OK: Provided hostname matches discovered certificate
- FYI: leaf cert "www.google.com" expires next (on 2020-08-12 12:08:31 +0000 UTC)

============================
CERTIFICATES | CHAIN DETAILS
============================

Certificate 1 of 2 (leaf):
        Name: CN=www.google.com,O=Google LLC,L=Mountain View,ST=California,C=US
        SANs entries: [www.google.com]
        KeyID: D:94:9F:90:8A:5C:E:B5:B5:DB:B7:79:7F:6A:9:42:3A:4D:CC:D4
        Issuer: CN=GTS CA 1O1,O=Google Trust Services,C=US
        IssuerKeyID: 98:D1:F8:6E:10:EB:CF:9B:EC:60:9F:18:90:1B:A0:EB:7D:9:FD:2B
        Serial: 41166161160297429311704478035915443513
        Expiration: 2020-08-12 12:08:31 +0000 UTC
        Status: [OK] 66d 23h remaining

Certificate 2 of 2 (intermediate):
        Name: CN=GTS CA 1O1,O=Google Trust Services,C=US
        SANs entries: []
        KeyID: 98:D1:F8:6E:10:EB:CF:9B:EC:60:9F:18:90:1B:A0:EB:7D:9:FD:2B
        Issuer: CN=GlobalSign,OU=GlobalSign Root CA - R2,O=GlobalSign
        IssuerKeyID: 9B:E2:7:57:67:1C:1E:C0:6A:6:DE:59:B4:9A:2D:DF:DC:19:86:2E
        Serial: 149699596615803609916394524856
        Expiration: 2021-12-15 00:00:42 +0000 UTC
        Status: [OK] 556d 11h remaining
```

#### WARNING results

```ShellSession
.\lscert.exe --server www.google.com --port 443 --age-critical 50 --age-warning 67

Connecting to remote server "www.google.com" at port 443


======================
CERTIFICATES | SUMMARY
======================

- OK: 2 certs found for service running on www.google.com at port 443
- OK: Provided hostname matches discovered certificate
- WARNING: 1 certificates expiring soon
- FYI: leaf cert "www.google.com" expires next (on 2020-08-12 12:08:31 +0000 UTC)

============================
CERTIFICATES | CHAIN DETAILS
============================

Certificate 1 of 2 (leaf):
        Name: CN=www.google.com,O=Google LLC,L=Mountain View,ST=California,C=US
        SANs entries: [www.google.com]
        KeyID: D:94:9F:90:8A:5C:E:B5:B5:DB:B7:79:7F:6A:9:42:3A:4D:CC:D4
        Issuer: CN=GTS CA 1O1,O=Google Trust Services,C=US
        IssuerKeyID: 98:D1:F8:6E:10:EB:CF:9B:EC:60:9F:18:90:1B:A0:EB:7D:9:FD:2B
        Serial: 41166161160297429311704478035915443513
        Expiration: 2020-08-12 12:08:31 +0000 UTC
        Status: [WARNING] 66d 23h remaining

Certificate 2 of 2 (intermediate):
        Name: CN=GTS CA 1O1,O=Google Trust Services,C=US
        SANs entries: []
        KeyID: 98:D1:F8:6E:10:EB:CF:9B:EC:60:9F:18:90:1B:A0:EB:7D:9:FD:2B
        Issuer: CN=GlobalSign,OU=GlobalSign Root CA - R2,O=GlobalSign
        IssuerKeyID: 9B:E2:7:57:67:1C:1E:C0:6A:6:DE:59:B4:9A:2D:DF:DC:19:86:2E
        Serial: 149699596615803609916394524856
        Expiration: 2021-12-15 00:00:42 +0000 UTC
        Status: [OK] 556d 11h remaining
```

In general, the differences between the `OK` and `WARNING` output for the two
tools is minor. However, unlike the `check_cert` Nagios plugin where we are
limited to one line of summary output, the `lscert` CLI tool doesn't share the
same output requirements and can be more expressive (e.g., such as the summary
section to highlight particular items of interest).

## License

From the [LICENSE](LICENSE) file:

```license
MIT License

Copyright (c) 2020 Adam Chalkley

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## References

- <https://github.com/grantae/certinfo>
- <https://github.com/rs/zerolog>
- <https://github.com/atc0005/go-nagios>
- <https://nagios-plugins.org/doc/guidelines.html>

- badssl.com
  - <https://github.com/chromium/badssl.com>
  - <https://expired.badssl.com/>
    - useful test target to demo output of tools, confirm expiration
      validation works as intended

<!-- Footnotes here  -->

[repo-url]: <https://github.com/atc0005/check-cert>  "This project's GitHub repo"

[go-docs-download]: <https://golang.org/dl>  "Download Go"

[go-docs-install]: <https://golang.org/doc/install>  "Install Go"

<!-- []: PLACEHOLDER "DESCRIPTION_HERE" -->
