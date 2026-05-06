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


## Features

- automatically derives the next development version from git tag history and working-tree state
- bumps the released version and creates an annotated, optionally signed git tag


## ☝️ Is it any good?

[yes](https://news.ycombinator.com/item?id=3067434).
