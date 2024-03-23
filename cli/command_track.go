package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"time"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/mathutil"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// TrackCommand is "track" command handler
func TrackCommand(args CommandArgs) int {
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

	interval := 3 // 3 seconds by default

	if args.Has(1) {
		interval, err = args.GetI(1)

		if err != nil {
			terminal.Error("Can't parse update interval: %v", err)
			return EC_ERROR
		}

		interval = mathutil.Between(interval, 1, 300)
	}

	return showInteractiveInfo(id, interval)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// showInteractiveInfo show interactive instance info
func showInteractiveInfo(id, interval int) int {
	fmtc.TPrintf("{s-}Preparation…{!}")

	for {
		state, err := CORE.GetInstanceState(id, true)

		if err != nil {
			fmtc.TPrintf("{r}%v{!}\n", err)
			return EC_ERROR
		}

		coloredState := getInstanceStateWithColor(state)

		if !state.IsWorks() {
			fmtc.TPrintf("State: " + coloredState)
			time.Sleep(time.Second * time.Duration(interval))
			continue
		}

		i1, err := CORE.GetInstanceInfo(id, time.Second, false)

		if err != nil {
			fmtc.TPrintf("{r}%v{!}\n", err)
			return EC_ERROR
		}

		time.Sleep(time.Second * time.Duration(interval))

		i2, err := CORE.GetInstanceInfo(id, time.Second, false)

		if err != nil {
			fmtc.TPrintf("{r}%v{!}\n", err)
			return EC_ERROR
		}

		usage := calculateInstanceCPUUsage(
			extractCPUUsageInfo(i1),
			extractCPUUsageInfo(i2),
			interval,
		)

		cpu := fmt.Sprintf("%g%%", mathutil.Between(fmtutil.Float(usage[0]+usage[1]), 0.0, 100.0))
		clients := i2.GetI("clients", "connected_clients")
		mem := i2.GetI("memory", "used_memory")
		memRSS := i2.GetI("memory", "used_memory_rss")
		memLua := i2.GetI("memory", "used_memory_lua")
		swap := i2.GetI("memory", "used_memory_swap")
		ops := i2.GetI("stats", "instantaneous_ops_per_sec")
		input := i2.GetF("stats", "instantaneous_input_kbps") * 1024
		output := i2.GetF("stats", "instantaneous_output_kbps") * 1024

		fmtc.TPrintf(
			"{*}State:{!} "+coloredState+" {s}|{!} {*}Clients:{!} %s {s}|{!} {*}Commands:{!} %s {s}|{!} {*}CPU:{!} %s {s}|{!} {*}Mem:{!} %s/%s {s}|{!} {*}Lua:{!} %s {s}|{!} {*}Swp:{!} %s {s}|{!} {*}In:{!} %s/s {s}|{!} {*}Out:{!} %s/s",
			fmtutil.PrettyNum(clients), fmtutil.PrettyNum(ops), cpu,
			fmtutil.PrettySize(mem), fmtutil.PrettySize(memRSS), fmtutil.PrettySize(memLua),
			fmtutil.PrettySize(swap), fmtutil.PrettySize(input), fmtutil.PrettySize(output),
		)
	}
}
