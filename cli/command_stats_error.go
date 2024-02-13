package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"strconv"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/fmtutil/table"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/pager"
	"github.com/essentialkaos/ek/v12/strutil"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
	REDIS "github.com/essentialkaos/rds/redis"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// StatsErrorCommand is "stats-error" command handler
func StatsErrorCommand(args CommandArgs) int {
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

	printErrorStatsInfo(info)

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// printErrorStatsInfo prints latency stats info from instance info
func printErrorStatsInfo(info *REDIS.Info) {
	section := info.Sections["errorstats"]

	if section == nil || len(section.Fields) == 0 {
		terminal.Warn("There is no info about errors")
		return
	}

	t := table.NewTable().SetHeaders("ERROR", "COUNT")

	for _, v := range section.Fields {
		errName := strings.ToUpper(strutil.Exclude(v, "errorstat_"))
		errInfo := parseFieldsLine(section.Values[v], ',')
		errCount, _ := strconv.Atoi(errInfo["count"])

		t.Add(errName, fmtutil.PrettyNum(errCount))
	}

	t.Render()
}
