---
title: "Installation"
description: "Install discord2110 from a release, with go install, or from source."
weight: 20
---

## Prebuilt binaries

Every [release](https://github.com/tamnd/discord2110-cli/releases) carries archives for Linux, macOS,
and Windows on amd64 and arm64, plus deb, rpm, and apk packages for Linux.
Download, unpack, put `discord2110` on your `PATH`, done. The `checksums.txt`
on each release is signed with keyless [cosign](https://docs.sigstore.dev/) if
you want to verify before running.

## With Go

```bash
go install github.com/tamnd/discord2110-cli/cmd/discord2110@latest
```

That puts `discord2110` in `$(go env GOPATH)/bin`, which is `~/go/bin` unless
you moved it. Make sure that directory is on your `PATH`.

## From source

```bash
git clone https://github.com/tamnd/discord2110-cli
cd discord2110-cli
make build        # produces ./bin/discord2110
./bin/discord2110 version
```

## Container image

```bash
docker run --rm ghcr.io/tamnd/discord2110:latest --help
```

## Checking the install

```bash
discord2110 version
```

prints the version and exits.
