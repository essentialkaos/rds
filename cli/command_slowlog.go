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
	"time"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil/table"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/pager"
	"github.com/essentialkaos/ek/v12/terminal"
	"github.com/essentialkaos/ek/v12/timeutil"

	CORE "github.com/essentialkaos/rds/core"
	REDIS "github.com/essentialkaos/rds/redis"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// SlowlogGetCommand is "slowlog-get" command handler
func SlowlogGetCommand(args CommandArgs) int {
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

	num := 10

	if args.Has(1) {
		num, err = args.GetI(1)

		if err != nil {
			terminal.Error("Can't parse number of recent entries: %v", err)
			return EC_ERROR
		}
	}

	if (options.GetB(OPT_PAGER) || prefs.AutoPaging) && !useRawOutput {
		if pager.Setup() == nil {
			defer pager.Complete()
		}
	}

	return slowlogGet(id, num)
}

// SlowlogResetCommand is "slowlog-reset" command handler
func SlowlogResetCommand(args CommandArgs) int {
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

	return slowlogReset(id)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// slowlogGet run SLOWLOG GET command
func slowlogGet(id, num int) int {
	resp, err := CORE.ExecCommand(
		id, &REDIS.Request{
			Command: []string{"SLOWLOG", "GET", strconv.Itoa(num)},
		},
	)

	if err != nil {
		terminal.Error("Can't execute SLOWLOG command on Redis instance")
		return EC_ERROR
	}

	entries, err := resp.Array()

	if len(entries) == 0 || err != nil {
		terminal.Warn("Slow log is empty")
		return EC_OK
	}

	t := table.NewTable("#", "DATE", "TIME", "CLIENT", "COMMAND")

	t.SetSizes(0, 20, 6)
	t.SetAlignments(table.ALIGN_RIGHT, table.ALIGN_RIGHT, table.ALIGN_RIGHT, table.ALIGN_RIGHT)

	for _, entry := range entries {
		elements, err := entry.Array()

		if len(elements) < 4 || err != nil {
			continue
		}

		entryID, _ := elements[0].Int()
		timestamp, _ := elements[1].Int64()
		execMs, _ := elements[2].Int64()
		cmd, _ := elements[3].List()
		client := "—"

		if len(elements) > 5 {
			client, _ = elements[4].Str()
		}

		date := timeutil.Format(time.Unix(timestamp, 0), "%Y/%m/%d %H:%M:%S")
		cmdFull := strconv.Quote(strings.Join(cmd, " "))

		t.Add(entryID, date, formatSlowlogTime(execMs), client, cmdFull[1:len(cmdFull)-1])
	}

	t.Render()

	return EC_OK
}

// slowlogReset run SLOWLOG RESET command
func slowlogReset(id int) int {
	resp, err := CORE.ExecCommand(
		id, &REDIS.Request{
			Command: []string{"SLOWLOG", "RESET"},
		},
	)

	if err != nil {
		terminal.Error("Can't execute SLOWLOG command on Redis instance")
		return EC_ERROR
	}

	_, err = resp.Str()

	if err != nil {
		terminal.Error("Can't parse SLOWLOG command response")
		return EC_ERROR
	}

	logger.Info(id, "Slow log reset")
	fmtc.Printf("{g}Slow log successfully cleared for instance %d{!}\n", id)

	return EC_OK
}

// formatSlowlogTime format execution time
func formatSlowlogTime(ms int64) string {
	if ms == 0 {
		return "0s"
	}

	d := time.Microsecond * time.Duration(ms)

	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.3gs", float64(d)/float64(time.Second))
	case d >= time.Millisecond:
		return fmt.Sprintf("%.3gms", float64(d)/float64(time.Millisecond))
	default:
		return fmt.Sprintf("%.3gµs", float64(d)/float64(time.Microsecond))
	}
}
