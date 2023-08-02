package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"time"

	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/fmtutil/table"
	"github.com/essentialkaos/ek/v12/mathutil"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
	REDIS "github.com/essentialkaos/rds/redis"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// CPUCommand is "cpu" command handler
func CPUCommand(args CommandArgs) int {
	err := args.Check(true)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	id, _, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	period := 3 // 3 seconds by default

	if args.Has(1) {
		period, err = args.GetI(1)

		if err != nil {
			terminal.Error("Can't parse period: %v", err)
			return EC_ERROR
		}

		period = mathutil.Between(period, 1, 3600)
	}

	return getInstanceCPUUsage(id, period)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// getInstanceCPUUsage calculate instance cpu usage for given period
func getInstanceCPUUsage(id, period int) int {
	pid := CORE.GetInstancePID(id)

	if pid == -1 {
		terminal.Error("Can't get instance PID")
		return EC_ERROR
	}

	i1, err := CORE.GetInstanceInfo(id, 3*time.Second, false)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	time.Sleep(time.Second * time.Duration(period))

	i2, err := CORE.GetInstanceInfo(id, 3*time.Second, false)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	u1 := extractCPUUsageInfo(i1)
	u2 := extractCPUUsageInfo(i2)

	usage := calculateInstanceCPUUsage(u1, u2, period)

	printInstanceCPUUsage(usage)

	return EC_OK
}

// calculateInstanceCPUUsage calculate instance cpu usage
func calculateInstanceCPUUsage(u1, u2 []float64, period int) []float64 {
	fp := float64(period)

	sys := u2[0] - u1[0]
	usr := u2[1] - u1[1]
	sysCh := u2[2] - u1[2]
	usrCh := u2[3] - u1[3]

	sys = (sys * 100) / fp
	usr = (usr * 100) / fp
	sysCh = (sysCh * 100) / fp
	usrCh = (usrCh * 100) / fp

	return []float64{sys, usr, sysCh, usrCh}
}

// printInstanceCPUUsage print info about cpu usage
func printInstanceCPUUsage(usage []float64) {
	sysStr := fmtutil.PrettyPerc(mathutil.BetweenF(fmtutil.Float(usage[0]), 0.0, 100.0))
	usrStr := fmtutil.PrettyPerc(mathutil.BetweenF(fmtutil.Float(usage[1]), 0.0, 100.0))
	sysChStr := fmtutil.PrettyPerc(mathutil.BetweenF(fmtutil.Float(usage[2]), 0.0, 100.0))
	usrChStr := fmtutil.PrettyPerc(mathutil.BetweenF(fmtutil.Float(usage[3]), 0.0, 100.0))

	t := table.NewTable("TYPE", "VALUE").SetSizes(16)

	t.Add("Sys", sysStr)
	t.Add("User", usrStr)
	t.Add("Sys (Children)", sysChStr)
	t.Add("User (Children)", usrChStr)

	t.Render()
}

// extractCPUUsageInfo extrtacts cpu usage from redis info
func extractCPUUsageInfo(info *REDIS.Info) []float64 {
	return []float64{
		info.GetF("cpu", "used_cpu_sys"),
		info.GetF("cpu", "used_cpu_user"),
		info.GetF("cpu", "used_cpu_sys_children"),
		info.GetF("cpu", "used_cpu_user_children"),
	}
}
