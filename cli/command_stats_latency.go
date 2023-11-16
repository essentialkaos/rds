package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v12/fmtutil/table"
	"github.com/essentialkaos/ek/v12/strutil"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
	REDIS "github.com/essentialkaos/rds/redis"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// StatsLatencyCommand is "stats-latency" command handler
func StatsLatencyCommand(args CommandArgs) int {
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

	printLatencyStatsInfo(info)

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// printLatencyStatsInfo prints latency stats info from instance info
func printLatencyStatsInfo(info *REDIS.Info) {
	section := info.Sections["latencystats"]

	if section == nil || len(section.Fields) == 0 {
		terminal.Warn("There is no info about latency")
		return
	}

	t := table.NewTable().SetHeaders(
		"COMMAND", "p50", "p99", "p99.9",
	).SetAlignments(
		table.ALIGN_LEFT, table.ALIGN_RIGHT, table.ALIGN_RIGHT,
	)

	for _, v := range section.Fields {
		cmdName := strings.ToUpper(strutil.Exclude(v, "latency_percentiles_usec_"))
		cmdLat := parseFieldsLine(section.Values[v], ",")

		cmdName = strings.ReplaceAll(cmdName, "|", " ")

		t.Add(
			cmdName,
			fmt.Sprintf("%s μs", cmdLat["p50"]),
			fmt.Sprintf("%s μs", cmdLat["p99"]),
			fmt.Sprintf("%s μs", cmdLat["p99.9"]),
		)
	}

	t.Render()
}
