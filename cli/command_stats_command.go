package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v13/fmtutil"
	"github.com/essentialkaos/ek/v13/fmtutil/table"
	"github.com/essentialkaos/ek/v13/mathutil"
	"github.com/essentialkaos/ek/v13/options"
	"github.com/essentialkaos/ek/v13/pager"
	"github.com/essentialkaos/ek/v13/strutil"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/timeutil"

	CORE "github.com/essentialkaos/rds/core"
	REDIS "github.com/essentialkaos/rds/redis"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// StatsCommandCommand is "stats-command" command handler
func StatsCommandCommand(args CommandArgs) int {
	err := args.Check(true)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	id, _, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	info, err := CORE.GetInstanceInfo(id, 5*time.Second, true)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	if (options.GetB(OPT_PAGER) || prefs.AutoPaging) && !useRawOutput {
		if pager.Setup() == nil {
			defer pager.Complete()
		}
	}

	printCommandStatsInfo(info)

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// printCommandStatsInfo prints commands stats info from instance info
func printCommandStatsInfo(info *REDIS.Info) {
	section := info.Sections["commandstats"]

	if section == nil || len(section.Fields) == 0 {
		terminal.Warn("There is no info about commands")
		return
	}

	t := table.NewTable().SetHeaders(
		"COMMAND", "CALLS", "TIME TOTAL", "TIME PER CALL", "REJECTED", "FAILED",
	).SetAlignments(
		table.ALIGN_LEFT, table.ALIGN_RIGHT, table.ALIGN_RIGHT,
		table.ALIGN_RIGHT, table.ALIGN_RIGHT,
	)

	for _, v := range section.Fields {
		cmdName := strings.ToUpper(strutil.Exclude(v, "cmdstat_"))
		cmdName = strings.ReplaceAll(cmdName, "|", " ")
		cmdStats := parseFieldsLine(section.Values[v], ',')
		cmdUSec, _ := strconv.Atoi(cmdStats["usec"])
		cmdTime := time.Microsecond * time.Duration(mathutil.Max(cmdUSec, 1))
		cmdCalls, _ := strconv.Atoi(cmdStats["calls"])
		cmdRejected, _ := strconv.Atoi(strutil.Q(cmdStats["rejected_calls"], "0"))
		cmdFailed, _ := strconv.Atoi(strutil.Q(cmdStats["failed_calls"], "0"))

		t.Add(
			cmdName,
			fmtutil.PrettyNum(cmdCalls),
			timeutil.PrettyDuration(cmdTime),
			fmt.Sprintf("%s μs", cmdStats["usec_per_call"]),
			fmtutil.PrettyNum(cmdRejected),
			fmtutil.PrettyNum(cmdFailed),
		)
	}

	t.Render()
}
