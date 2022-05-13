# semverkzeug 🛠️

Another stab at the old problem of handling semantic versioning.

What makes *semverkzeug* compelling is its automatic behavior of generating versions similar to what [setuptools_scm](https://github.com/pypa/setuptools_scm/) does while being easy to install and portable thanks to *semverkzeug* being written in [Go](https://go.dev/).


## 🎯 Goals

Semverkzeug aims to be a **simple** tool for automatically handling [Semantic Versioning 2.0.0](https://semver.org/#semantic-versioning-200) compliant version strings for any kind of software project in a language and packaging method agnostic way.


### Non-Goals

* generating a changelog for you
* triggering your build-system for you
* uploading tagged releases anywhere


## 📦 Installation

```console
foo@bar:~ $ go install github.com/0x5a17ed/semverkzeug@latest
```


## 🤔 Usage

```console
foo@bar:~/git/myproject $ semverkzeug describe 
v0.3.1-dev.0.20220513192209
```


## 💡 Features

- provides floating versions based on the state of a git repository
- supports bumping your software's version and creating a git tag


## ☝️ Is it any good?

[yes](https://news.ycombinator.com/item?id=3067434).
