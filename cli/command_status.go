package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"time"

	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/timeutil"

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
		fmtc.Printfn("Instance {*}%d{!} is {y}stopped{!}", id)
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
				fmtc.Printfn(
					"Instance {*}%d{!} is {g}works{!} and {g*}saving data{!} {s-}(started %s ago){!}",
					id, timeutil.PrettyDuration(info.GetI("persistence", "rdb_current_bgsave_time_sec")),
				)

			case info != nil && info.Get("persistence", "aof_rewrite_in_progress") == "1":
				fmtc.Printfn(
					"Instance {*}%d{!} is {g}works{!} and {g*}saving data{!} {s-}(started %s ago){!}",
					id, timeutil.PrettyDuration(info.GetI("persistence", "aof_current_rewrite_time_sec")),
				)

			default:
				fmtc.Printfn("Instance {*}%d{!} is {g}works{!} and {g*}saving data{!}", id)
			}

			return EC_OK
		}

		if state.IsLoading() {
			if info != nil {
				fmtc.Printfn(
					"Instance {*}%d{!} is {g}works{!} and {c}loading{!} {s-}(%s%% loaded){!}",
					id, info.Get("persistence", "loading_loaded_perc"),
				)
				return EC_OK
			}

			fmtc.Printfn("Instance {*}%d{!} is {g}works{!} and {c}loading{!}", id)
			return EC_OK
		}

		switch {
		case state.IsHang():
			fmtc.Printfn("Instance {*}%d{!} is {m}hang{!}", id)
			return EC_OK
		case state.IsSyncing():
			fmtc.Printfn("Instance {*}%d{!} is {g}works{!} and {g_}syncing{!}", id)
			return EC_OK
		default:
			fmtc.Printfn("Instance {*}%d{!} is {g}works{!}", id)
			return EC_OK
		}
	}

	return EC_OK
}
