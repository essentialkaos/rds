package core

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/system"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// getTHPStatus returns current status of transparent huge pages (enabled, defrag, error)
func getTHPStatus() (bool, bool, error) {
	thpPath := fsutil.ProperPath("DR", []string{
		"/sys/kernel/mm/transparent_hugepage",
		"/sys/kernel/mm/redhat_transparent_hugepage",
	})

	thpEnabledInfo, err := os.ReadFile(thpPath + "/enabled")

	if err != nil {
		return false, false, fmt.Errorf("Can't read THP info: %v", err)
	}

	thpDefragInfo, err := os.ReadFile(thpPath + "/defrag")

	if err != nil {
		return false, false, fmt.Errorf("Can't read THP info: %v", err)
	}

	thpEnabled := !strings.Contains(string(thpEnabledInfo), "[never]")
	thpDefrag := !strings.Contains(string(thpDefragInfo), "[never]")

	return thpEnabled, thpDefrag, nil
}

// getSysctlSetting read setting value from sysctl
func getSysctlSetting(key string) (int, error) {
	cmd := exec.Command("sysctl", "-n", key)
	output, err := cmd.Output()

	if err != nil {
		return -1, fmt.Errorf("Error while sysctl execution: %v", err)
	}

	value, err := strconv.Atoi(strings.Trim(string(output), "\n\r"))

	if err != nil {
		return -1, fmt.Errorf("Can't parse sysctl output: %v", err)
	}

	return value, nil
}

// isSystemHasTHPIssues returns true if system has problems with THP
func isSystemHasTHPIssues() (bool, error) {
	thpEnabled, thpDefrag, err := getTHPStatus()

	if err != nil {
		return false, err
	}

	return thpEnabled || thpDefrag, nil
}

// isSystemHasKernelIssues returns true if system has problems with kernel configuration
func isSystemHasKernelIssues() (bool, error) {
	overcommitMem, err := getSysctlSetting("vm.overcommit_memory")

	if err != nil {
		return false, err
	}

	somaxconn, err := getSysctlSetting("net.core.somaxconn")

	if err != nil {
		return false, err
	}

	return overcommitMem == 0 || somaxconn < MIN_SOMAXCONN, nil
}

// isSystemHasLimitsIssues returns true if system has problems with limits for Redis user
func isSystemHasLimitsIssues() (bool, error) {
	if !fsutil.CheckPerms("FRX", BIN_RUNUSER) {
		return false, fmt.Errorf("%s can't be executed", BIN_RUNUSER)
	}

	user := Config.GetS(REDIS_USER)
	cmd := exec.Command(BIN_RUNUSER, "-s", "/bin/bash", user, "-c", "ulimit -n")
	output, err := cmd.Output()

	if err != nil {
		return false, fmt.Errorf("Error while ulimit execution: %v", err)
	}

	procNum, err := strconv.Atoi(strings.Trim(string(output), "\n\r"))

	if err != nil {
		return false, fmt.Errorf("Can't parse ulimit output: %v", err)
	}

	return procNum < MIN_PROCS, nil
}

// isSystemHasFSIssues returns true if system has problems with FS
func isSystemHasFSIssues(force bool) (bool, error) {
	if !force && Config.GetB(MAIN_DISABLE_FILESYSTEM_CHECK, false) {
		return false, nil
	}

	memUsage, err := system.GetMemUsage()

	if err != nil {
		return false, err
	}

	fsInfo, err := system.GetFSUsage()

	if err != nil {
		return false, err
	}

	dataDirInfo := findDirDeviceInfo(Config.GetS(PATH_DATA_DIR), fsInfo)

	if dataDirInfo == nil {
		return false, nil
	}

	return dataDirInfo.Total < (memUsage.MemTotal+memUsage.SwapTotal)*2, nil
}

// findDirDeviceInfo try to find device info for given path
func findDirDeviceInfo(path string, info map[string]*system.FSUsage) *system.FSUsage {
	for p, i := range info {
		if p == "/" {
			continue
		}

		if strings.HasPrefix(path, p) {
			return i
		}
	}

	return info["/"]
}
