package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil/table"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/pager"
	"github.com/essentialkaos/ek/v12/spinner"
	"github.com/essentialkaos/ek/v12/terminal"

	API "github.com/essentialkaos/rds/api"
	CORE "github.com/essentialkaos/rds/core"
	SC "github.com/essentialkaos/rds/sync/client"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// SentinelStartCommand is "sentinel-start" command handler
func SentinelStartCommand(args CommandArgs) int {
	if CORE.IsSentinelActive() {
		terminal.Warn("Redis Sentinel already works")
		return EC_WARN
	}

	spinner.Show("Starting Sentinel")
	errs := CORE.SentinelStart()
	spinner.Done(len(errs) == 0)

	if len(errs) != 0 {
		fmtc.NewLine()

		terminal.Error("\nErrors while starting Sentinel:")

		for _, err := range errs {
			terminal.Error("  %v", err)
		}

		return EC_ERROR
	}

	err := SC.PropagateCommand(API.COMMAND_SENTINEL_START, -1, "")

	if err != nil {
		fmtc.NewLine()
		terminal.Error(err)
		return EC_ERROR
	}

	logger.Info(-1, "Started Sentinel")

	return EC_OK
}

// SentinelStopCommand is "sentinel-stop" command handler
func SentinelStopCommand(args CommandArgs) int {
	if !CORE.IsSentinelActive() {
		terminal.Warn("Redis Sentinel already stopped")
		return EC_WARN
	}

	spinner.Show("Stopping Sentinel")
	err := CORE.SentinelStop()
	spinner.Done(err == nil)

	if err != nil {
		terminal.Error("\nError while executing command: %v", err)
		return EC_ERROR
	}

	err = SC.PropagateCommand(API.COMMAND_SENTINEL_STOP, -1, "")

	if err != nil {
		fmtc.NewLine()
		terminal.Error(err)
		return EC_ERROR
	}

	logger.Info(-1, "Stopped Sentinel")

	return EC_OK
}

// SentinelResetCommand is "sentinel-reset" command handler
func SentinelResetCommand(args CommandArgs) int {
	if !CORE.IsSentinelActive() {
		terminal.Warn("Sentinel must works for executing this command")
		return EC_WARN
	}

	spinner.Show("Sending RESET to Sentinel")
	err := CORE.SentinelReset()
	spinner.Done(err == nil)

	if err != nil {
		terminal.Error("\nError while executing command: %v", err)
		return EC_ERROR
	}

	logger.Info(-1, "Sentinel state has been reset for all instances")

	return EC_OK
}

// SentinelStatusCommand is "sentinel-status" command handler
func SentinelStatusCommand(args CommandArgs) int {
	if CORE.IsSentinelActive() {
		fmtc.Printf("Redis Sentinel is {g}works{!}\n")
	} else {
		fmtc.Printf("Redis Sentinel is {y}stopped{!}\n")
	}

	return EC_OK
}

// SentinelSwitchMasterCommand is "sentinel-switch-master" command handler
func SentinelSwitchMasterCommand(args CommandArgs) int {
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

	spinner.Show("Switching master")
	err = CORE.SentinelMasterSwitch(id)
	spinner.Done(err == nil)

	if err != nil {
		terminal.Error("\nError while executing command: %v", err)
		return EC_ERROR
	}

	logger.Info(-1, "Switched master role in Sentinel to current node")

	return EC_OK
}

// SentinelCheckCommand is "sentinel-check" command handler
func SentinelCheckCommand(args CommandArgs) int {
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

	status, ok := CORE.SentinelCheck(id)

	if !ok {
		terminal.Error(status)
		return EC_ERROR
	}

	fmtc.Printf("{g}%s{!}\n", status)

	return EC_OK
}

// SentinelInfoCommand is "sentinel-info" command handler
func SentinelInfoCommand(args CommandArgs) int {
	if !CORE.IsSentinelActive() {
		terminal.Error("Sentinel must works for executing this command")
		return EC_ERROR
	}

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

	info, err := CORE.SentinelInfo(id)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	if options.GetB(OPT_PAGER) && !useRawOutput {
		if pager.Setup() == nil {
			defer pager.Complete()
		}
	}

	t := table.NewTable().SetSizes(23, 96)

	t.Separator()
	fmtc.Printf("{*} %s{!}\n", "Master")
	t.Separator()

	for _, item := range info.Master {
		t.Print(item.Name, item.Value)
	}

	replicasCount := len(info.Replicas)

	if replicasCount != 0 {
		t.Separator()
		fmtc.Printf("{*} %s{!} {s}(%d){!}\n", "Replicas", replicasCount)
		t.Separator()

		for index, replica := range info.Replicas {
			for _, item := range replica {
				t.Print(item.Name, item.Value)
			}

			if index != replicasCount-1 {
				t.Separator()
			}
		}
	}

	sentinelsCount := len(info.Sentinels)

	if sentinelsCount != 0 {
		t.Separator()
		fmtc.Printf("{*} %s{!} {s}(%d){!}\n", "Sentinels", sentinelsCount)
		t.Separator()

		for index, sentinel := range info.Sentinels {
			for _, item := range sentinel {
				t.Print(item.Name, item.Value)
			}

			if index != sentinelsCount-1 {
				t.Separator()
			}
		}
	}

	t.Separator()

	return EC_OK
}

// SentinelMasterCommand is "sentinel-master" command handler
func SentinelMasterCommand(args CommandArgs) int {
	if !CORE.IsSentinelActive() {
		terminal.Error("Sentinel must works for executing this command")
		return EC_ERROR
	}

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

	ip, err := CORE.SentinelMasterIP(id)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	fmtc.Printf("Master IP is {*}%s{!}\n", ip)

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //
