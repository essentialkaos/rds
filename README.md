<p align="center"><a href="#readme"><img src=".github/images/card.svg"/></a></p>

<p align="center">
  <a href="https://kaos.sh/l/rds"><img src="https://kaos.sh/l/b1568323e77e3a605a24.svg" alt="Code Climate Maintainability" /></a>
  <a href="https://kaos.sh/y/rds"><img src="https://kaos.sh/y/e22a4319c08b42b5923e9d5ee85ae4d8.svg" alt="Codacy badge" /></a>
  <a href="https://kaos.sh/w/rds/ci"><img src="https://kaos.sh/w/rds/ci.svg" alt="GitHub Actions CI Status" /></a>
  <a href="https://kaos.sh/w/rds/codeql"><img src="https://kaos.sh/w/rds/codeql.svg" alt="GitHub Actions CodeQL Status" /></a>
  <a href="#license"><img src=".github/images/license.svg"/></a>
</p>

<p align="center"><a href="#usage-demo">Usage demo</a> • <a href="#installation">Installation</a> • <a href="#usage">Usage</a> • <a href="#ci-status">CI Status</a> • <a href="#contributing">Contributing</a> • <a href="#license">License</a></p>

`RDS` is a tool for Redis orchestration.

### Usage demo

[![demo](https://gh.kaos.st/rds-100a.gif)](#usage-demo)

### Installation

> [!IMPORTANT]
> We highly recommend you checkout [Requirements](https://github.com/essentialkaos/rds/wiki/Requirements) before RDS installation. It can save you from useless work.

#### From [ESSENTIAL KAOS DNF Repository](https://pkgs.kaos.st)

```bash
sudo dnf install -y https://pkgs.kaos.st/kaos-repo-latest.el$(grep 'CPE_NAME' /etc/os-release | tr -d '"' | cut -d':' -f5).noarch.rpm
sudo dnf install rds rds-sync redis70
```

Run `sudo rds go` command and follow the instructions. Check out the [FAQ section](https://kaos.sh/rds/w/FAQ) of our wiki for common questions about using RDS.

<details><summary><b>About Redis versions</b></summary><p>

RDS supports the next versions of Redis and Sentinel:

* `6.2.x`
* `7.0.x`
* `7.2.x` **← ʀᴇᴄᴏᴍᴍᴇɴᴅᴇᴅ**
* `7.4.x`

RDS packages do not have Redis as a dependency, so you can install it from any source (_package, sources, prebuilt binaries…_).

[ESSENTIAL KAOS YUM/DNF Repository](https://pkgs.kaos.st) provides pinned (_pinned to a specific version, for example, 7.2.x_) and unpinned versions of the Redis package:

* `redis`
* `redis62`
* `redis70`
* `redis72` **← ʀᴇᴄᴏᴍᴍᴇɴᴅᴇᴅ**
* `redis74`

</p></details>

### Usage

<p align="center"><img src=".github/images/usage.svg"/></p>

### CI Status

| Branch | Status |
|--------|--------|
| `master` | [![CI](https://kaos.sh/w/rds/ci.svg?branch=master)](https://kaos.sh/w/rds/ci?query=branch:master) |
| `develop` | [![CI](https://kaos.sh/w/rds/ci.svg?branch=develop)](https://kaos.sh/w/rds/ci?query=branch:develop) |

### Contributing

Before contributing to this project please read our [Contributing Guidelines](https://github.com/essentialkaos/contributing-guidelines#contributing-guidelines).

### License

[Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0)

<p align="center"><a href="https://essentialkaos.com"><img src="https://gh.kaos.st/ekgh.svg"/></a></p>
