<h1 align="center"><code>semverkzeug</code></h1>
<br>
Automatic semantic version strings derived from your git tag history. Language-agnostic, single binary, no project config required.

```console
$ semverkzeug describe
v0.0.1-dev.260506T10351400Z
```
The version above was derived automatically from the latest git tag plus a timestamp from your repository's working-tree state.

Comparable in spirit to [setuptools_scm](https://github.com/pypa/setuptools_scm/), but language-agnostic and shipped as a single [Go](https://go.dev/) binary, easy to install locally, trivial to drop into any CI environment.


## Goals

Semverkzeug aims to help you stop thinking about version numbers. It tells you the current version from your git tag history and bumps it — major, minor, or patch level — when you ask, following [Semantic Versioning 2.0.0](https://semver.org/#semantic-versioning-200) throughout. No commit-message convention required, no version file, no project config.


### Non-Goals

* forcing a commit message convention on you
* generating a changelog for you
* triggering your build-system for you
* uploading tagged releases anywhere


## Installation

```console
foo@bar:~ $ go install github.com/0x5a17ed/semverkzeug/cmd/semverkzeug@latest
```


## Usage

### Inspecting the current version

```console
foo@bar:~/git/myproject $ semverkzeug describe 
v0.0.1-dev.260506T10351400Z
```

### Bumping the current version

```console
foo@bar:~/git/myproject $ semverkzeug bump patch
==> Creating annotated tag [v0.0.1]
 -> Target: e6f3fa7 (initial code import)
 -> Running: git tag -a -F - v0.0.1 e6f3fa7127fd385f44ed28346cbab27a4f9148be
   :: git may pause here waiting for signing (touch your security key if prompted)
==> Created tag [v0.0.1]
```


### Checking a bump before tagging

To see the tag that a bump would create without creating it, pass `--dry-run`. It runs every check, including the check for a clean working tree. To make sure the bump yields the version you expect, pass `--expect`, which also accepts the forms `v1.2.3` and `mod/v1.2.3`:

```console
foo@bar:~/git/myproject $ semverkzeug bump minor --expect 0.2.0 --dry-run
==> Would create annotated tag [v0.2.0]
 -> Target: e6f3fa7 (release 0.2.0)
```

When the bump yields another version, the command fails and creates nothing. A tool that writes its own record of versions, such as a changelog, can use this to catch a disagreement before it commits.

### Using your own tag message

By default, the tag message names the bump, such as `bump version v0.1.0 -> v0.2.0`. To use your own message instead, such as release notes, pass `--message-file PATH`, or `--message-file -` to read it from standard input. Lines that start with `#`, such as Markdown headings, stay in the message.


## Features

- automatically derives the next development version from git tag history and working-tree state
- bumps the released version and creates an annotated, optionally signed git tag
- checks a bump against an expected version, and previews it without creating the tag
- takes the tag message from a file, such as release notes


## ☝️ Is it any good?

[yes](https://news.ycombinator.com/item?id=3067434).
