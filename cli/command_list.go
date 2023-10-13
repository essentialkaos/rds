package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/fmtutil/table"
	"github.com/essentialkaos/ek/v12/mathutil"
	"github.com/essentialkaos/ek/v12/system/process"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// ListCommand is "list" command handler
func ListCommand(args CommandArgs) int {
	if !CORE.HasInstances() {
		terminal.Warn("No instances are created")

		return EC_WARN
	}

	filter := args
	idList := CORE.GetInstanceIDList()

	t := table.NewTable("ID", "MEMORY", "OWNER", "DESCRIPTION")

	lastID := strconv.Itoa(idList[len(idList)-1])
	idColumnSize := mathutil.Between(len(lastID), 2, 4)

	t.SetSizes(idColumnSize, 10, 18)
	t.SetAlignments(table.ALIGN_RIGHT, table.ALIGN_RIGHT, table.ALIGN_RIGHT, table.ALIGN_LEFT)

	dataShown := listInstanceSearch(t, idList, filter, false)

	if !dataShown && len(filter) != 0 {
		dataShown = listInstanceSearch(t, idList, filter, true)
	}

	if !useRawOutput {
		if dataShown {
			t.Separator()

			if !fmtc.DisableColors {
				fmtc.Println("\n Legend: {s-}stopped{!} {s}∙{!} {r}dead{!} {s}∙{!} {y}idle{!} {s}∙{!} {g}active{!} {s}∙{!} {g*}saving{!} {s}∙{!} {g_}syncing{!} {s}∙{!} {c}loading{!} {s}∙{!} {m}hang{!} {s}∙{!} unknown")
			}
		} else {
			terminal.Warn("No instances found")
		}
	}

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

func listInstanceSearch(t *table.Table, idList []int, filter []string, fullText bool) bool {
	dataShown := false

	for _, id := range idList {
		state, err := CORE.GetInstanceState(id, true)

		if err != nil {
			state = CORE.INSTANCE_STATE_UNKNOWN
		}

		meta, err := CORE.GetInstanceMeta(id)

		if err != nil {
			state = CORE.INSTANCE_STATE_UNKNOWN
		}

		switch fullText {
		case true:
			if !isDescFit(filter, meta.Desc) {
				continue
			}
		case false:
			if !isFilterFit(filter, state, meta) {
				continue
			}
		}

		if useRawOutput {
			fmt.Println(id)
			dataShown = true
			continue
		}

		if fullText {
			t.Print(
				getInstanceIDWithColor(id, state),
				getInstanceMemoryUsageWithColor(id, state),
				getInstanceOwnerWithColor(meta, true),
				getInstanceDescWithTags(meta, filter),
			)
		} else {
			t.Print(
				getInstanceIDWithColor(id, state),
				getInstanceMemoryUsageWithColor(id, state),
				getInstanceOwnerWithColor(meta, true),
				getInstanceDescWithTags(meta, nil),
			)
		}

		dataShown = true
	}

	return dataShown
}

// isFilterFit return true if instance fit for filter
func isFilterFit(filter []string, state CORE.State, meta *CORE.InstanceMeta) bool {
	if len(filter) == 0 {
		return true
	}

	fit := true

	for _, filterValue := range filter {
		switch filterValue {
		case "my":
			fit = meta.Auth.User == CORE.User.RealName
		case "works", "on":
			fit = state.IsWorks()
		case "stopped", "stop", "off":
			fit = state.IsStopped()
		case "dead", "problems":
			fit = state.IsDead()
		case "hang":
			fit = state.IsHang()
		case "idle", "inactive":
			fit = state.IsWorks() && state.IsIdle()
		case "active":
			fit = state.IsWorks() && !state.IsIdle()
		case "syncing", "sync":
			fit = state.IsWorks() && state.IsSyncing()
		case "saving", "save":
			fit = state.IsWorks() && state.IsSaving()
		case "loading", "load":
			fit = state.IsWorks() && state.IsLoading()
		case "abandoned":
			fit = state.IsWorks() && state.IsAbandoned()
		case "master-up":
			fit = state.IsWorks() && state.IsMasterUp()
		case "master-down":
			fit = state.IsWorks() && state.IsMasterDown()
		case "no-replica":
			fit = state.IsWorks() && state.NoReplica()
		case "with-replica":
			fit = state.IsWorks() && state.WithReplica()
		case "with-errors":
			fit = state.IsWorks() && state.WithErrors()
		case "orphan":
			fit = isInstanceOwnerExist(meta.Auth.User) == false
		case "outdated":
			currentRedisVer, _ := CORE.GetRedisVersion()
			if state.IsStopped() {
				fit = false
			} else if currentRedisVer.String() != "" && meta.Compatible != "" {
				fit = currentRedisVer.String() != meta.Compatible
			}
		case "standby":
			fit = meta.Preferencies.ReplicationType == CORE.REPL_TYPE_STANDBY
		case "replica":
			fit = meta.Preferencies.ReplicationType == CORE.REPL_TYPE_REPLICA
		case "secure":
			fit = meta.Preferencies.ServicePassword != ""
		default:
			fit = meta.Auth.User == filterValue
		}

		if !fit && strings.HasPrefix(filterValue, "@") {
			fit = isMetaContainsTag(meta, strings.TrimLeft(filterValue, "@"))
		}

		if !fit {
			return false
		}
	}

	return fit
}

// isDescFit return true if
func isDescFit(filter []string, desc string) bool {
	for _, f := range filter {
		if strings.Contains(desc, f) {
			return true
		}
	}

	return false
}

// getInstanceMemoryUsageWithColor returns instance memory usage
func getInstanceMemoryUsageWithColor(id int, state CORE.State) string {
	if !state.IsWorks() {
		return "{s-}∙∙∙∙∙∙∙∙{!}"
	}

	pid := CORE.GetInstancePID(id)

	if pid == -1 {
		return "{y}????????{!}"
	}

	usage, err := process.GetMemInfo(pid)

	if err != nil {
		return "{y}????????{!}"
	}

	return fmtutil.PrettySize(usage.VmRSS)
}

// isMetaContainsTag return true if instance has given tag
func isMetaContainsTag(meta *CORE.InstanceMeta, tag string) bool {
	for _, instanceTag := range meta.Tags {
		rawTag, _ := CORE.ParseTag(instanceTag)

		if tag == rawTag {
			return true
		}
	}

	return false
}
