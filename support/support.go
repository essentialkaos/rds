package support

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"

	"github.com/essentialkaos/ek/v13/support"
	"github.com/essentialkaos/ek/v13/support/deps"
	"github.com/essentialkaos/ek/v13/support/kernel"
	"github.com/essentialkaos/ek/v13/support/network"
	"github.com/essentialkaos/ek/v13/support/pkgs"
	"github.com/essentialkaos/ek/v13/support/resources"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Print prints verbose info about application, system, dependencies and
// important environment
func Print(app, ver, gitRev string, gomod []byte) {
	support.Collect(app, ver).
		WithRevision(gitRev).
		WithDeps(deps.Extract(gomod)).
		WithPackages(pkgs.Collect("redis,redis62,redis70,redis72,redis74")).
		WithPackages(pkgs.Collect("redis-cli,redis62-cli,redis70-cli,redis72-cli,redis74-cli")).
		WithPackages(pkgs.Collect("rds", "rds-sync", "systemd", "tuned")).
		WithChecks(checkSystem()...).
		WithChecks(checkSyncDaemon()).
		WithChecks(checkKeepalived()).
		WithApps(getRedisVersion()).
		WithNetwork(network.Collect()).
		WithResources(resources.Collect()).
		WithKernel(kernel.Collect(
			"vm.swappiness",
			"vm.overcommit_memory",
			"net.core.somaxconn",
			"vm.nr_hugepages",
			"vm.nr_overcommit_hugepages",
		)).Print()
}

// checkKeepalived checks status of keepalived
func checkKeepalived() support.Check {
	virtualIP := CORE.Config.GetS(CORE.KEEPALIVED_VIRTUAL_IP)

	if virtualIP == "" {
		return support.Check{}
	}

	chk := support.Check{Status: support.CHECK_OK, Title: "Virtual IP"}

	switch CORE.GetKeepalivedState() {
	case CORE.KEEPALIVED_STATE_MASTER:
		chk.Message = fmt.Sprintf("%s (master)", virtualIP)
	case CORE.KEEPALIVED_STATE_BACKUP:
		chk.Status = support.CHECK_SKIP
		chk.Message = fmt.Sprintf("%s (backup)", virtualIP)
	default:
		chk.Message = fmt.Sprintf("%s (unknown status)", virtualIP)
		chk.Status = support.CHECK_WARN
	}

	return chk
}

// checkSystem checks system for problems
func checkSystem() []support.Check {
	var chks []support.Check

	currentRedisVer, err := CORE.GetRedisVersion()

	if err != nil {
		chks = append(chks, support.Check{support.CHECK_ERROR, "Redis", "Can't check Redis version"})
	}

	if currentRedisVer.IsZero() {
		chks = append(chks, support.Check{support.CHECK_ERROR, "Redis", "Can't extract or parse Redis version"})
	}

	status, err := CORE.GetSystemConfigurationStatus(true)

	if err != nil {
		chks = append(chks, support.Check{support.CHECK_ERROR, "System", "Can't check system for problems"})
		return chks
	}

	if status.HasTHPIssues {
		chks = append(chks, support.Check{support.CHECK_ERROR, "THP", "Transparent hugepages are not disabled"})
	} else {
		chks = append(chks, support.Check{support.CHECK_OK, "THP", "No issues"})
	}

	if status.HasKernelIssues {
		chks = append(chks, support.Check{support.CHECK_ERROR, "Kernel", "Kernel is not properly configured for Redis"})
	} else {
		chks = append(chks, support.Check{support.CHECK_OK, "Kernel", "No issues"})
	}

	if status.HasLimitsIssues {
		chks = append(chks, support.Check{support.CHECK_OK, "Limits", "Limits are not set"})
	} else {
		chks = append(chks, support.Check{support.CHECK_OK, "Limits", "No issues"})
	}

	if status.HasFSIssues {
		chks = append(chks, support.Check{support.CHECK_OK, "Filesystem", "Not enough free space on disk"})
	} else {
		chks = append(chks, support.Check{support.CHECK_OK, "Filesystem", "No issues"})
	}

	return chks
}

// checkSyncDaemon checks for sync daemon status
func checkSyncDaemon() support.Check {
	if !CORE.IsSyncDaemonInstalled() {
		return support.Check{}
	}

	chk := support.Check{Status: support.CHECK_OK, Title: "Sync Daemon"}

	switch CORE.IsSyncDaemonActive() {
	case true:
		chk.Message = "Sync daemon works"
	default:
		chk.Status, chk.Message = support.CHECK_SKIP, "Sync daemon is stopped"
	}

	return chk
}

// getRedisVersion returns current Redis version
func getRedisVersion() support.App {
	currentRedisVer, _ := CORE.GetRedisVersion()
	return support.App{"Redis", currentRedisVer.String()}
}
