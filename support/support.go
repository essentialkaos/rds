package support

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/hash"
	"github.com/essentialkaos/ek/v12/strutil"
	"github.com/essentialkaos/ek/v12/system"
	"github.com/essentialkaos/ek/v12/system/container"

	"github.com/essentialkaos/depsy"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Pkg contains basic package info
type Pkg struct {
	Name    string
	Version string
}

// Pkgs is slice with packages
type Pkgs []Pkg

// ////////////////////////////////////////////////////////////////////////////////// //

// Print prints verbose info about application, system, dependencies and
// important environment
func Print(app, ver, gitRev string, gomod []byte) {
	fmtutil.SeparatorTitleColorTag = "{s-}"
	fmtutil.SeparatorFullscreen = false
	fmtutil.SeparatorColorTag = "{s-}"
	fmtutil.SeparatorSize = 80

	showApplicationInfo(app, ver, gitRev)
	showOSInfo()
	showConfigurationInfo()
	showRedisVersionInfo()
	showKeepalivedInfo()
	showDepsInfo(gomod)

	fmtutil.Separator(false)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// showOSInfo shows verbose information about system
func showOSInfo() {
	osInfo, err := system.GetOSInfo()

	if err == nil {
		fmtutil.Separator(false, "OS INFO")

		printInfo(12, "Name", osInfo.ColoredName())
		printInfo(12, "Pretty Name", osInfo.ColoredPrettyName())
		printInfo(12, "Version", osInfo.Version)
		printInfo(12, "ID", osInfo.ID)
		printInfo(12, "ID Like", osInfo.IDLike)
		printInfo(12, "Version ID", osInfo.VersionID)
		printInfo(12, "Version Code", osInfo.VersionCodename)
		printInfo(12, "Platform ID", osInfo.PlatformID)
		printInfo(12, "CPE", osInfo.CPEName)
	}

	systemInfo, err := system.GetSystemInfo()

	if err != nil {
		return
	} else if osInfo == nil {
		fmtutil.Separator(false, "SYSTEM INFO")
		printInfo(12, "Name", systemInfo.OS)
	}

	printInfo(12, "Arch", systemInfo.Arch)
	printInfo(12, "Kernel", systemInfo.Kernel)

	containerEngine := "No"

	switch container.GetEngine() {
	case container.DOCKER:
		containerEngine = "Yes (Docker)"
	case container.PODMAN:
		containerEngine = "Yes (Podman)"
	case container.LXC:
		containerEngine = "Yes (LXC)"
	}

	fmtc.NewLine()

	printInfo(12, "Container", containerEngine)
}

// showApplicationInfo shows verbose information about application
func showApplicationInfo(app, ver, gitRev string) {
	fmtutil.Separator(false, "APPLICATION INFO")

	printInfo(7, "Name", app)
	printInfo(7, "Version", fmtc.Sprintf("%s{s}/%s{!}", ver, CORE.VERSION))

	printInfo(7, "Go", fmtc.Sprintf(
		"%s {s}(%s/%s){!}",
		strings.TrimLeft(runtime.Version(), "go"),
		runtime.GOOS, runtime.GOARCH,
	))

	if gitRev == "" {
		gitRev = extractGitRevFromBuildInfo()
	}

	if gitRev != "" {
		if !fmtc.DisableColors && fmtc.IsTrueColorSupported() {
			printInfo(7, "Git SHA", gitRev+getHashColorBullet(gitRev))
		} else {
			printInfo(7, "Git SHA", gitRev)
		}
	}

	bin, _ := os.Executable()
	binSHA := hash.FileHash(bin)

	if binSHA != "" {
		binSHA = strutil.Head(binSHA, 7)
		if !fmtc.DisableColors && fmtc.IsTrueColorSupported() {
			printInfo(7, "Bin SHA", binSHA+getHashColorBullet(binSHA))
		} else {
			printInfo(7, "Bin SHA", binSHA)
		}
	}
}

// showConfigurationInfo shows info about system configuration
func showConfigurationInfo() {
	fmtutil.Separator(false, "CONFIGURATION INFO")

	status, err := CORE.GetSystemConfigurationStatus(true)

	if err != nil {
		fmtc.Printf("  {r}Unable to check system: %v{!}\n", err.Error())
		return
	}

	fmtFlag := func(v bool) string {
		switch v {
		case true:
			return fmtc.Sprintf("{r~}Has issues{!}")
		default:
			return fmtc.Sprintf("No issues")
		}
	}

	printInfo(10, "THP", fmtFlag(status.HasTHPIssues))
	printInfo(10, "Kernel", fmtFlag(status.HasKernelIssues))
	printInfo(10, "Limits", fmtFlag(status.HasLimitsIssues))
	printInfo(10, "Filesystem", fmtFlag(status.HasFSIssues))
}

// showRedisVersionInfo shows info about redis version
func showRedisVersionInfo() {
	fmtutil.Separator(false, "REDIS INFO")

	currentRedisVer, _ := CORE.GetRedisVersion()

	printInfo(12, "Redis Server", currentRedisVer.String())
}

// showDepsInfo shows information about all dependencies
func showDepsInfo(gomod []byte) {
	deps := depsy.Extract(gomod, false)

	if len(deps) == 0 {
		return
	}

	fmtutil.Separator(false, "DEPENDENCIES")

	for _, dep := range deps {
		if dep.Extra == "" {
			fmtc.Printf(" {s}%8s{!}  %s\n", dep.Version, dep.Path)
		} else {
			fmtc.Printf(" {s}%8s{!}  %s {s-}(%s){!}\n", dep.Version, dep.Path, dep.Extra)
		}
	}
}

// showKeepalivedInfo shows info about keepalived virtual IP
func showKeepalivedInfo() {
	fmtutil.Separator(false, "KEEPALIVED INFO")

	virtualIP := CORE.Config.GetS(CORE.KEEPALIVED_VIRTUAL_IP)

	if virtualIP == "" {
		printInfo(10, "Virtual IP", "")
		return
	}

	switch CORE.GetKeepalivedState() {
	case CORE.KEEPALIVED_STATE_MASTER:
		printInfo(10, "Virtual IP", fmtc.Sprintf("%s {g}(master){!}", virtualIP))
	case CORE.KEEPALIVED_STATE_BACKUP:
		printInfo(10, "Virtual IP", fmtc.Sprintf("%s {s}(backup){!}", virtualIP))
	default:
		printInfo(10, "Virtual IP", fmtc.Sprint("{r}check error{!}"))
	}
}

// extractGitRevFromBuildInfo extracts git SHA from embedded build info
func extractGitRevFromBuildInfo() string {
	info, ok := debug.ReadBuildInfo()

	if !ok {
		return ""
	}

	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && len(s.Value) > 7 {
			return s.Value[:7]
		}
	}

	return ""
}

// getHashColorBullet return bullet with color from hash
func getHashColorBullet(v string) string {
	if len(v) > 6 {
		v = strutil.Head(v, 6)
	}

	return fmtc.Sprintf(" {#" + strutil.Head(v, 6) + "}● {!}")
}

// printInfo formats and prints info record
func printInfo(size int, name, value string) {
	name += ":"
	size++

	if value == "" {
		fm := fmt.Sprintf("  {*}%%-%ds{!}  {s-}—{!}\n", size)
		fmtc.Printf(fm, name)
	} else {
		fm := fmt.Sprintf("  {*}%%-%ds{!}  %%s\n", size)
		fmtc.Printf(fm, name, value)
	}
}

// ////////////////////////////////////////////////////////////////////////////////// //
