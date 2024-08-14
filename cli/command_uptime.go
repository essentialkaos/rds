package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"strconv"
	"time"

	"github.com/essentialkaos/ek/v13/fmtutil/table"
	"github.com/essentialkaos/ek/v13/fsutil"
	"github.com/essentialkaos/ek/v13/mathutil"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/timeutil"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// UptimeCommand is "uptime" command handler
func UptimeCommand(args CommandArgs) int {
	if !CORE.HasInstances() {
		terminal.Warn("No instances are created")
		return EC_WARN
	}

	idList := CORE.GetInstanceIDList()
	lastID := strconv.Itoa(idList[len(idList)-1])
	idColumnSize := mathutil.Between(len(lastID), 2, 4)

	t := table.NewTable(
		"ID", "UPTIME", "LAST SAVE", "DESCRIPTION",
	).SetSizes(
		idColumnSize, 6, 20,
	).SetAlignments(
		table.AR, table.AR, table.AR, table.AL,
	)

	for _, id := range CORE.GetInstanceIDList() {
		uptime, lastSave := "{s-}∙∙∙∙∙∙{!}", "{s-}∙∙∙∙∙∙∙∙{!}"

		state, err := CORE.GetInstanceState(id, true)

		if err != nil {
			state = CORE.INSTANCE_STATE_UNKNOWN
		}

		meta, err := CORE.GetInstanceMeta(id)

		if err != nil {
			t.Print(
				getInstanceIDWithColor(id, state),
				uptime, lastSave, "{s-}—{!}",
			)

			continue
		}

		if state.IsWorks() {
			pidFile := CORE.GetInstancePIDFilePath(id)
			modTime, _ := fsutil.GetMTime(pidFile)

			if !modTime.IsZero() {
				uptime = timeutil.MiniDuration(time.Since(modTime))
			}
		}

		_, saveDate, _ := getInstanceDataInfo(id)

		if !saveDate.IsZero() {
			lastSave = timeutil.Format(saveDate, "%Y/%m/%d %H:%M:%S")
		}

		t.Print(
			getInstanceIDWithColor(id, state),
			uptime, lastSave,
			getInstanceDescWithTags(meta, state.IsWorks(), nil),
		)
	}

	t.Border()

	return EC_OK
}
