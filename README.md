# verify-links

CLI tool that helps verify current status of URIs in text files

# Table of Contents

- [Prerequisites](#prerequisites)
- [Installing](#installing)
- [Usage](#usage)
  - [Common usage](#common-usage)
  - [Flags](#flags)
  - [Running with docker](#running-with-docker)
- [Screenshots](#screenshots)
  - [Terminal output](#terminal-output)
  - [Github Action](#github-action)
- [Running in a Github Action](#running-in-a-github-action)

# Prerequisites

- Golang 1.16

# Installing

```
go install github.com/ellisonleao/vl/cmd/vl@latest
```

# Usage

## Common usage

```
$ vl FILE
```

## Flags

```
-a (Skip status codes list)
```

Example:

```sh
$ vl README.md -a 500,404
```

All `500` and `404` errors will be ignored and not considered as errors

```
-t (timeout for each request)
```

Example:

```sh
$ vl README.md -t 30s
```

Each request that takes more than 30s will be considered as an error. The values
accepted must be under the durations values. Some examples
[here](https://golang.org/pkg/time/#ParseDuration)

```
-w Whitelist URIs
```

Example:

```sh
$ vl README.md -w server1.com,server2.com
```

Adds a list of whitelisted links that shouldn't be verified. Links must be exactly
passed as they are in the text file

## Running with docker

```
$ docker run -it --rm -v $PWD:/vl ellisonleao/vl /vl/yourfile.md
```

# Screenshots

## Terminal output

_terminal colors are only working in linux_

![](https://i.postimg.cc/xqD8YDfz/Screenshot-from-2021-03-18-17-42-31.png)

## Github Action

![](https://i.postimg.cc/VNpd4bxg/Screenshot-from-2021-03-18-18-29-21.png)

# Running in a Github Action

An example of a workflow file:

```yaml
---
name: CI
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-go@v3
        id: go
        with:
          go-version: '>=1.17.0'

      - uses: actions/checkout@v3

      - run: go install github.com/ellisonleao/vl/cmd/vl@latest
      - run: vl README.md
```
