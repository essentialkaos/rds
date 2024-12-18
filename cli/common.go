package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fmtutil/panel"
	"github.com/essentialkaos/ek/v13/fmtutil/table"
	"github.com/essentialkaos/ek/v13/fsutil"
	"github.com/essentialkaos/ek/v13/strutil"
	"github.com/essentialkaos/ek/v13/system"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/terminal/input"
	"github.com/essentialkaos/ek/v13/timeutil"
	"github.com/essentialkaos/ek/v13/version"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type CommandArgs []string

type CommandHandler func(CommandArgs) int

// ////////////////////////////////////////////////////////////////////////////////// //

var userExistenceCache map[string]bool

// ////////////////////////////////////////////////////////////////////////////////// //

// Check checks command arguments
func (a CommandArgs) Check(mustWorks bool) error {
	if len(a) == 0 {
		return errors.New("You must define instance ID for this command")
	}

	id, _, err := CORE.ParseIDDBPair(a[0])

	if err != nil {
		return err
	}

	if !CORE.IsInstanceExist(id) {
		return fmt.Errorf("Instance with ID %d does not exist", id)
	}

	if !mustWorks {
		return nil
	}

	state, err := CORE.GetInstanceState(id, false)

	if err != nil {
		return errors.New("Can't get instance state")
	}

	if !state.IsWorks() {
		return errors.New("Instance must work for executing this command")
	}

	return nil
}

// Has returns true if arguments contains argument with given index
func (a CommandArgs) Has(index int) bool {
	return index < len(a) && a[index] != ""
}

// Get returns argument with given index
func (a CommandArgs) Get(index int) string {
	if index >= len(a) {
		return ""
	}

	return a[index]
}

// GetI returns argument with given index as int
func (a CommandArgs) GetI(index int) (int, error) {
	v := a.Get(index)
	return strconv.Atoi(v)
}

// GetF returns argument with given index as float
func (a CommandArgs) GetF(index int) (float64, error) {
	v := a.Get(index)
	return strconv.ParseFloat(v, 64)
}

// GetB returns argument with given index as boolean
func (a CommandArgs) GetB(index int) (bool, error) {
	v := strings.ToLower(a.Get(index))
	return v == "true" || v == "yes", nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// getForceArg checks and return forced action flag
func getForceArg(a string) (bool, error) {
	if a == "force" || a == "true" {
		return true, nil
	}

	return false, errors.New(`You should use "force" or "true" as a flag of forced action`)
}

// isAllRedisCompatible checks if instances with given ID's are compatible with
// current Redis version
func isAllRedisCompatible(ids []int) bool {
	if len(ids) == 0 {
		return true
	}

	currentRedisVer, err := CORE.GetRedisVersion()

	if err != nil || currentRedisVer.String() == "" {
		return true
	}

	for _, id := range ids {
		isCompatible, _, _ := isRedisCompatible(id, currentRedisVer)

		if !isCompatible {
			return false
		}
	}

	return true
}

// isRedisCompatible checks if instance with given ID is compatible with given
// Redis version
func isRedisCompatible(id int, currentVersion version.Version) (bool, version.Version, version.Version) {
	var err error

	if !CORE.IsInstanceExist(id) {
		return true, version.Version{}, version.Version{}
	}

	if currentVersion.IsZero() {
		currentVersion, err = CORE.GetRedisVersion()

		if err != nil || currentVersion.IsZero() {
			return true, version.Version{}, version.Version{}
		}
	}

	meta, err := CORE.GetInstanceMeta(id)

	if err != nil || meta.Compatible == "" {
		return true, version.Version{}, version.Version{}
	}

	compatVersion, err := version.Parse(meta.Compatible)

	if err != nil {
		return true, version.Version{}, version.Version{}
	}

	if compatVersion.Major() != currentVersion.Major() {
		return false, compatVersion, currentVersion
	}

	return true, version.Version{}, version.Version{}
}

// isSomeConfigUpdated checks if some config was updated after start
func isSomeConfigUpdated(ids []int) bool {
	if len(ids) == 0 {
		return true
	}

	for _, id := range ids {
		if isConfigUpdated(id) {
			return true
		}
	}

	return false
}

// isConfigUpdated checks if instance config was updated after start
func isConfigUpdated(id int) bool {
	if !CORE.IsInstanceExist(id) {
		return false
	}

	meta, err := CORE.GetInstanceMeta(id)

	if err != nil {
		return false
	}

	mtime, _ := fsutil.GetMTime(CORE.GetInstanceConfigFilePath(id))

	if mtime.Unix() == meta.Config.Date {
		return false
	}

	hash, err := CORE.GetInstanceConfigHash(id)

	if err != nil {
		return false
	}

	if meta.Config.Hash != hash {
		return true
	}

	return false
}

// getStateColorTag returns color tag for given state
func getStateColorTag(state CORE.State) string {
	if state.IsStopped() {
		return "{s-}"
	}

	if state.IsDead() {
		return "{r}"
	}

	if state.IsHang() {
		return "{m}"
	}

	if state.IsLoading() {
		return "{c}"
	}

	var modificator string

	if state.IsSaving() {
		modificator += "*"
	}

	if state.IsSyncing() {
		modificator += "_"
	}

	if state.IsIdle() {
		return "{y" + modificator + "}"
	}

	if state.IsWorks() {
		return "{g" + modificator + "}"
	}

	return ""
}

// getStateName returns name for given state
func getStateName(state CORE.State) string {
	switch {
	case state.IsStopped():
		return "Stopped"
	case state.IsDead():
		return "Dead"
	case state.IsHang():
		return "Hang"
	case state.IsLoading():
		return "Loading"
	case state.IsSaving():
		return "Saving"
	case state.IsSyncing():
		return "Syncing"
	case state.IsIdle():
		return "Idle"
	case state.IsWorks():
		return "Active"
	default:
		return "Unknown"
	}
}

// getInstanceStateWithColor returns state with color tags
func getInstanceStateWithColor(state CORE.State) string {
	return getStateColorTag(state) + getStateName(state) + "{!}"
}

// getInstanceIDWithColor returns ID with color tags
func getInstanceIDWithColor(id int, state CORE.State) string {
	return getStateColorTag(state) + strconv.Itoa(id) + "{!}"
}

// getInstanceOwnerWithColor returns owner name with color tags
func getInstanceOwnerWithColor(meta *CORE.InstanceMeta, before bool) string {
	if meta == nil {
		return "{s-}??????{!}"
	}

	owner := meta.Auth.User

	if isInstanceOwnerExist(owner) {
		if CORE.User.RealName == owner {
			switch before {
			case true:
				return "{g}•{!} " + owner
			default:
				return owner + " {g}•{!}"
			}
		}

		return owner
	}

	return "{s-}" + owner + "{!}"
}

// isInstanceOwnerExist returns true if instance owner user exist on the system
func isInstanceOwnerExist(owner string) bool {
	if userExistenceCache == nil {
		userExistenceCache = make(map[string]bool)
	}

	_, found := userExistenceCache[owner]

	if !found {
		userExistenceCache[owner] = system.IsUserExist(owner)
	}

	return userExistenceCache[owner]
}

// getInstanceDescWithTags returns instance description with rendered tags
func getInstanceDescWithTags(meta *CORE.InstanceMeta, isWorks bool, highlights []string) string {
	if meta == nil {
		return "{s-}—{!}"
	}

	desc := meta.Desc

	if len(highlights) != 0 {
		desc = applyHighlights(desc, highlights)
	}

	if !isWorks {
		desc = "{s-}" + desc + "{!}"
	}

	return desc + " " + renderTags(meta.Tags...)
}

// renderTags render all tags with colors
func renderTags(tags ...string) string {
	if len(tags) == 0 {
		return ""
	}

	var result []string

	for _, tag := range tags {
		tagName, tagColor := CORE.ParseTag(tag)

		switch tagColor {
		case "r", "red":
			result = append(result, "{r}#"+tagName+"{!}")
		case "b", "blue":
			result = append(result, "{b}#"+tagName+"{!}")
		case "g", "green":
			result = append(result, "{g}#"+tagName+"{!}")
		case "y", "yellow":
			result = append(result, "{y}#"+tagName+"{!}")
		case "c", "cyan":
			result = append(result, "{c}#"+tagName+"{!}")
		case "m", "magenta":
			result = append(result, "{m}#"+tagName+"{!}")
		default:
			result = append(result, "{s}#"+tagName+"{!}")
		}
	}

	return strings.Join(result, " ")
}

// showInstanceBasicInfo prints basic instance info
func showInstanceBasicInfoCard(id int, state CORE.State) error {
	meta, err := CORE.GetInstanceMeta(id)

	if err != nil {
		return err
	}

	t := table.NewTable().SetSizes(14, 96)

	t.Border()
	fmtc.Println(" ▾ {*}INSTANCE INFO{!}")
	t.Border()

	t.Print("ID", id)
	t.Print("Owner", getInstanceOwnerWithColor(meta, false))
	t.Print("Description", meta.Desc)
	t.Print("State", getInstanceStateWithColor(state))
	t.Print("Created", timeutil.Format(time.Unix(meta.Created, 0), "%Y/%m/%d %H:%M:%S"))

	t.Border()

	fmtc.NewLine()

	return nil
}

// warnAboutUnsafeAction warns the user about unsafe action
func warnAboutUnsafeAction(id int, message string) bool {
	state, err := CORE.GetInstanceState(id, true)

	if err != nil {
		terminal.Error(err)
		return false
	}

	err = showInstanceBasicInfoCard(id, state)

	if err != nil {
		terminal.Error(err)
		return false
	}

	isCompatible, compatibleVer, currentVer := isRedisCompatible(id, version.Version{})

	for i := 0; i < 6; i++ {
		switch i {
		case 0:
			if isCompatible {
				continue
			}

			if compatibleVer.Major() > currentVer.Major() {
				terminal.Warn(
					"Redis was downgraded (%s → %s). Old versions of Redis can not read data saved",
					compatibleVer.String(), currentVer.String(),
				)
				terminal.Warn("by the new ones. Also, instance configuration is not checked for compatibility with")
				terminal.Warn("the newly installed version of Redis.")
			} else {
				terminal.Warn(
					"Redis was updated (%s → %s). Instance is not checked for configuration file",
					compatibleVer.String(), currentVer.String(),
				)
				terminal.Warn("compatibility with the newly installed version of Redis.")
			}

		case 1:
			if !isConfigUpdated(id) {
				continue
			}

			terminal.Warn("Instance configuration file was changed and can be incompatible.")

		case 2:
			if !state.IsSyncing() {
				continue
			}

			terminal.Warn("Instance currently syncing with master or slave.")

		case 3:
			if !state.IsSaving() {
				continue
			}

			terminal.Warn("Instance currently saving data to disk.")

		case 4:
			if !state.IsLoading() {
				continue
			}

			terminal.Warn("Instance currently loading data from disk.")

		case 5:
			ops, err := getCurrentInstanceTraffic(id)

			switch {
			case ops > 10:
				terminal.Warn("There is some traffic on the instance (%d ops/s).", ops)
			case err != nil:
				terminal.Warn("Can't measure the number of operations on the instance.")
			default:
				continue
			}
		}

		fmtc.NewLine()
	}

	ok, err := input.ReadAnswer(message, "N")

	if err != nil || !ok {
		return false
	}

	return true
}

// isEnoughMemoryToCreate checks system for free memory and ask user if there is
// not enough memory
func isEnoughMemoryToCreate() bool {
	memUsage := getMemUsage()

	if memUsage < CORE.Config.GetI(CORE.MAIN_WARN_USED_MEMORY) {
		return true
	}

	panel.Warn(
		fmt.Sprintf(
			"Less than %d%% of free system memory",
			CORE.Config.GetI(CORE.MAIN_WARN_USED_MEMORY),
		),
		fmt.Sprintf("More than %d%% of available memory in use on this system. We highly recommend to do not create new Redis instances.", memUsage),
	)

	fmtc.NewLine()

	ok, err := input.ReadAnswer("Do you want to create new instances?", "N")

	if err != nil || !ok {
		return false
	}

	return true
}

// isSystemConfigured checks system configuration and print error message
// if the system is not configured
func isSystemConfigured() bool {
	status, err := CORE.GetSystemConfigurationStatus(false)

	if err == nil && !status.HasProblems {
		return true
	}

	if err != nil {
		terminal.Warn("Warning! Can't check system configuration: %v", err)
	} else {
		terminal.Warn("Warning! System is unconfigured.\n")

		if status.HasTHPIssues {
			terminal.Warn("• You should disable THP (Transparent Huge Pages) on this system.")
		}

		if status.HasLimitsIssues {
			terminal.Warn("• You should increase the maximum number of open file descriptors for the Redis user.")
		}

		if status.HasKernelIssues {
			terminal.Warn("• You should set vm.overcommit_memory setting and increase net.core.somaxconn")
			terminal.Warn("  setting in your sysctl configuration file.")
		}

		if status.HasFSIssues {
			terminal.Warn("• You should increase the size of the partition used for Redis data. Size of the")
			terminal.Warn("  partition should be at least twice bigger than the size of available")
			terminal.Warn("  memory (physical + swap).")
		}
	}

	fmtc.NewLine()

	ok, err := input.ReadAnswer("Do you want to proceed (it highly unrecommended)?", "N")

	if err != nil || !ok {
		return false
	}

	return true
}

// checkVirtualIP checks and warns the user about keepalived virtual IP
func checkVirtualIP() bool {
	if !CORE.IsFailoverMethod(CORE.FAILOVER_METHOD_STANDBY) ||
		!CORE.IsMaster() || CORE.Config.Is(CORE.KEEPALIVED_VIRTUAL_IP, "") {
		return true
	}

	switch CORE.GetKeepalivedState() {
	case CORE.KEEPALIVED_STATE_MASTER:
		return true
	case CORE.KEEPALIVED_STATE_UNKNOWN:
		terminal.Error("Can't check keepalived virtual IP status")
	case CORE.KEEPALIVED_STATE_BACKUP:
		terminal.Error(
			"This server has no keepalived virtual IP (%s). No longer a master?",
			CORE.Config.GetS(CORE.KEEPALIVED_VIRTUAL_IP),
		)
	}

	fmtc.NewLine()

	ok, err := input.ReadAnswer("Do you want to proceed (it highly unrecommended)?", "N")

	if err != nil || !ok {
		return false
	}

	return true
}

// getCurrentInstanceTraffic returns the number of commands processed by the instance
// in the last second
func getCurrentInstanceTraffic(id int) (uint64, error) {
	info, err := CORE.GetInstanceInfo(id, 3*time.Second, false)

	if err != nil {
		return 0, err
	}

	return info.GetU("stats", "instantaneous_ops_per_sec"), nil
}

// isSystemWasRebooted returns true if system was rebooted some time ago
func isSystemWasRebooted() (bool, error) {
	idList := CORE.GetInstanceIDList()

	if len(idList) == 0 || !fsutil.IsExist(CORE.GetStatesFilePath()) {
		return false, nil
	}

	uptime, err := system.GetUptime()

	if err != nil {
		return false, fmt.Errorf("Can't check system uptime: %w", err)
	}

	statesModTime, err := fsutil.GetMTime(CORE.GetStatesFilePath())

	if err != nil {
		return false, fmt.Errorf("Can't check states file modification date: %w", err)
	}

	if uint64(time.Since(statesModTime).Seconds()) > uptime {
		return true, nil
	}

	return false, nil
}

// parseFieldsLine parses line with many fields
func parseFieldsLine(line string, separator rune) map[string]string {
	result := make(map[string]string)

	for i := 0; i < 32; i++ {
		item := strutil.ReadField(line, i, false, separator)

		if item == "" {
			break
		}

		name, value, ok := strings.Cut(item, "=")

		if ok {
			result[name] = value
		}
	}

	return result
}

// applyHighlights applies highlights to given string
func applyHighlights(data string, highlights []string) string {
	if len(highlights) == 0 {
		return data
	}

	// Reverse sort by highlight token length
	sort.Slice(highlights, func(i, j int) bool {
		return len(highlights[i]) > len(highlights[j])
	})

	highlightRe := regexp.MustCompile(fmt.Sprintf("(?i)(%s)", strings.Join(highlights, "|")))

	data = highlightRe.ReplaceAllStringFunc(data, func(s string) string {
		return "{_}" + s + "{!_}"
	})

	closeRe := regexp.MustCompile(`\{!\}([^\{]*)\{!\}`)

	// Deduplicate all closing tags
	for closeRe.MatchString(data) {
		data = closeRe.ReplaceAllStringFunc(data, func(s string) string {
			return strutil.Substr(s, 3, 999) // Remove leading tag
		})
	}

	return data
}

// getInstanceDataInfo returns info about instance data (RDB/AOF)
func getInstanceDataInfo(id int) (int64, time.Time, error) {
	var err error
	var size int64
	var modTime time.Time

	dumpFile := CORE.GetInstanceRDBPath(id)

	if fsutil.IsExist(dumpFile) {
		size = fsutil.GetSize(dumpFile)
		modTime, err = fsutil.GetMTime(dumpFile)
	} else {
		aofFile := CORE.GetInstanceAOFPath(id)

		if fsutil.IsExist(aofFile) {
			size = fsutil.GetSize(dumpFile)
			modTime, err = fsutil.GetMTime(aofFile)
		}
	}

	if err != nil {
		return size, modTime, fmt.Errorf("Can't check data modification date: %w", err)
	}

	return size, modTime, nil
}
