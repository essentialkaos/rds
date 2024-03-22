package support

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"

	"github.com/essentialkaos/ek/v12/support"
	"github.com/essentialkaos/ek/v12/support/deps"
	"github.com/essentialkaos/ek/v12/support/pkgs"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Print prints verbose info about application, system, dependencies and
// important environment
func Print(app, ver, gitRev string, gomod []byte) {
	support.Collect(app, ver).
		WithRevision(gitRev).
		WithDeps(deps.Extract(gomod)).
		WithPackages(pkgs.Collect("redis", "redis-cli", "rds", "rds-sync")).
		WithChecks(checkSystem()...).
		WithChecks(checkKeepalived()).
		WithApps(getRedisVersion()).
		Print()
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
		chk.Message = fmt.Sprintf("IP: %s (master)", virtualIP)
	case CORE.KEEPALIVED_STATE_BACKUP:
		chk.Message = fmt.Sprintf("IP: %s (backup)", virtualIP)
	default:
		chk.Message = fmt.Sprintf("IP: %s (unknown status)", virtualIP)
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
		chks = append(chks, support.Check{support.CHECK_OK, "THP", ""})
	}

	if status.HasKernelIssues {
		chks = append(chks, support.Check{support.CHECK_ERROR, "Kernel", "Kernel is not properly configured for Redis"})
	} else {
		chks = append(chks, support.Check{support.CHECK_OK, "Kernel", ""})
	}

	if status.HasLimitsIssues {
		chks = append(chks, support.Check{support.CHECK_OK, "Limits", "Limits are not set"})
	} else {
		chks = append(chks, support.Check{support.CHECK_OK, "Limits", ""})
	}

	if status.HasFSIssues {
		chks = append(chks, support.Check{support.CHECK_OK, "Filesystem", "Not enough free space on disk"})
	} else {
		chks = append(chks, support.Check{support.CHECK_OK, "Filesystem", ""})
	}

	return chks
}

// getRedisVersion returns current Redis version
func getRedisVersion() support.App {
	currentRedisVer, _ := CORE.GetRedisVersion()
	return support.App{"Redis", currentRedisVer.String()}
}
