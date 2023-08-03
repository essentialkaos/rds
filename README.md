<p align="center"><a href="#readme"><img src="https://gh.kaos.st/rds.svg"/></a></p>

<p align="center">
  <a href="https://kaos.sh/r/rds"><img src="https://kaos.sh/r/rds.svg" alt="GoReportCard" /></a>
  <a href="https://kaos.sh/l/rds"><img src="https://kaos.sh/l/b1568323e77e3a605a24.svg" alt="Code Climate Maintainability" /></a>
  <a href="https://kaos.sh/b/rds"><img src="https://kaos.sh/b/b9119bdd-79a1-46e8-8f31-238843410ad8.svg" alt="codebeat badge" /></a>
  <a href="https://kaos.sh/w/rds/ci"><img src="https://kaos.sh/w/rds/ci.svg" alt="GitHub Actions CI Status" /></a>
  <a href="https://kaos.sh/w/rds/codeql"><img src="https://kaos.sh/w/rds/codeql.svg" alt="GitHub Actions CodeQL Status" /></a>
  <a href="#license"><img src="https://gh.kaos.st/apache2.svg"></a>
</p>

<p align="center"><a href="#usage-demo">Usage demo</a> • <a href="#installation">Installation</a> • <a href="#usage">Usage</a> • <a href="#ci-status">CI Status</a> • <a href="#contributing">Contributing</a> • <a href="#license">License</a></p>

`RDS` is a tool for Redis orchestration.

### Usage demo

[![demo](https://gh.kaos.st/demo.gif)](#usage-demo)

### Installation

#### From [ESSENTIAL KAOS YUM/DNF Repository](https://pkgs.kaos.st)

```bash
sudo yum install -y https://pkgs.kaos.st/kaos-repo-latest.el$(grep 'CPE_NAME' /etc/os-release | tr -d '"' | cut -d':' -f5).noarch.rpm
sudo yum install rds rds-sync
```

```bash
sudo rds go
```

### Usage

```
Usage: rds {options} {command}

Instances commands

  create                       Create new Redis instance
  destroy id                   Destroy (delete) Redis instance
  edit id                      Edit metadata for instance
  start id                     Start Redis instance
  stop id force                Stop Redis instance
  restart id                   Restart Redis instance
  kill id                      Kill Redis instance
  status id                    Show current status of Redis instance
  cli id:db command            Run CLI connected to Redis instance
  cpu id period                Calculate instance CPU usage
  memory id                    Show instance memory usage
  info id section              Show system info about Redis instance
  stats-command id             Show statistics based on the command type
  stats-latency id             Show latency statistics based on the command type
  stats-error id               Show error statistics
  clients id filter            Show list of connected clients
  track id interval            Show interactive info about Redis instance
  conf id property             Show configuration of Redis instance
  list filter                  Show list of all Redis instances
  stats                        Show overall statistics
  top field num                Show instances top
  top-diff file field num      Compare current and dumped top data
  top-dump file                Dump top data to file
  slowlog-get id num           Show last entries from slow log
  slowlog-reset id             Clear slow log
  check                        Check for dead instances
  tag-add id tag               Add tag to instance
  tag-remove id tag            Remove tag from instance

Superuser commands

  go                           Generate superuser access credentials
  batch-create csv-file        Create many instances at once
  batch-edit id                Edit many instances at once
  stop-all                     Stop all instances
  start-all                    Start all instances
  restart-all                  Restart all instances
  reload id                    Reload configuration for one or all instances
  regen id                     Regenerate configuration file for one or all instances
  state-save file              Save state of all instances
  state-restore file           Restore state of all instances

Replication commands

  replication                  Show replication info
  replication-role-set         Reconfigure node after changing the role

Sentinel commands

  sentinel-start               Start Redis Sentinel daemon
  sentinel-stop                Stop Redis Sentinel daemon
  sentinel-status              Show status of Redis Sentinel daemon
  sentinel-enable id           Enable Sentinel monitoring for instance
  sentinel-disable id          Disable Sentinel monitoring for instance
  sentinel-info id             Show info from Sentinel for some instance
  sentinel-master id           Show IP of master instance
  sentinel-check id            Check Sentinel configuration
  sentinel-reset               Reset state in Sentinel for all instances
  sentinel-switch-master id    Switch instance to master role

Common commands

  help command                 Show command usage info
  settings option…             Show settings from global configuration file
  gen-token                    Generate authentication token for sync daemon

Options

  --secure, -s                  Create secure Redis instance with auth support
  --disable-saves, -ds          Disable saves for created instance
  --private, -p                 Force access to private data
  --tags, -t tag                List of tags
  --format, -f text|json|xml    Data output format
  --yes, -y                     Automatically answer yes for all questions
  --simple, -S                  Simplify output (useful for copy-paste)
  --no-color, -nc               Disable colors in output
  --help, -h                    Show this help message
  --version, -v                 Show information about version
  --verbose-version, -vv        Show verbose information about version
```

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
