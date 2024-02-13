package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"time"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/terminal"
	"github.com/essentialkaos/ek/v12/timeutil"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// StatusCommand is "status" command handler
func StatusCommand(args CommandArgs) int {
	err := args.Check(false)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	id, _, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	state, err := CORE.GetInstanceState(id, true)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	switch {
	case state.IsStopped():
		fmtc.Printf("Instance {*}%d{!} is {y}stopped{!}\n", id)
		return EC_OK
	case state.IsDead():
		terminal.Error("PID file for instance %d exist, but service doesn't work", id)
		return EC_ERROR
	}

	if state.IsWorks() {
		info, _ := CORE.GetInstanceInfo(id, 5*time.Second, false)

		if state.IsSaving() {
			switch {
			case info != nil && info.Get("persistence", "rdb_bgsave_in_progress") == "1":
				fmtc.Printf(
					"Instance {*}%d{!} is {g}works{!} and {g*}saving data{!} {s-}(started %s ago){!}\n",
					id, timeutil.PrettyDuration(info.GetI("persistence", "rdb_current_bgsave_time_sec")),
				)

			case info != nil && info.Get("persistence", "aof_rewrite_in_progress") == "1":
				fmtc.Printf(
					"Instance {*}%d{!} is {g}works{!} and {g*}saving data{!} {s-}(started %s ago){!}\n",
					id, timeutil.PrettyDuration(info.GetI("persistence", "aof_current_rewrite_time_sec")),
				)

			default:
				fmtc.Printf("Instance {*}%d{!} is {g}works{!} and {g*}saving data{!}\n", id)
			}

			return EC_OK
		}

		if state.IsLoading() {
			if info != nil {
				fmtc.Printf(
					"Instance {*}%d{!} is {g}works{!} and {c}loading{!} {s-}(%s%% loaded){!}\n",
					id, info.Get("persistence", "loading_loaded_perc"),
				)
				return EC_OK
			}

			fmtc.Printf("Instance {*}%d{!} is {g}works{!} and {c}loading{!}\n", id)
			return EC_OK
		}

		switch {
		case state.IsHang():
			fmtc.Printf("Instance {*}%d{!} is {m}hang{!}\n", id)
			return EC_OK
		case state.IsSyncing():
			fmtc.Printf("Instance {*}%d{!} is {g}works{!} and {g_}syncing{!}\n", id)
			return EC_OK
		default:
			fmtc.Printf("Instance {*}%d{!} is {g}works{!}\n", id)
			return EC_OK
		}
	}

	return EC_OK
}
