package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"time"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/spinner"
	"github.com/essentialkaos/ek/v12/terminal"

	API "github.com/essentialkaos/rds/api"
	CORE "github.com/essentialkaos/rds/core"
	SC "github.com/essentialkaos/rds/sync/client"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// StartAllCommand is "start-all" command handler
func StartAllCommand(args CommandArgs) int {
	ec := checkForStartAll()

	if ec != EC_OK {
		return ec
	}

	idList, err := CORE.GetInstanceIDListByState(CORE.INSTANCE_STATE_STOPPED | CORE.INSTANCE_STATE_DEAD)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	if len(idList) == 0 {
		terminal.Warn("No stopped or dead instances")
		return EC_WARN
	}

	hasErrors := false

	for _, id := range idList {
		meta, err := CORE.GetInstanceMeta(id)

		if err == nil {
			spinner.Show("Starting instance %d {s}(%s){!}", id, meta.Desc)
		} else {
			spinner.Show("Starting instance %d", id)
		}

		state, err := CORE.GetInstanceState(id, false)

		if err != nil {
			spinner.Done(false)
			hasErrors = true
			continue
		}

		if state.IsStopped() || state.IsDead() {
			err = CORE.StartInstance(id, true)

			if err != nil {
				spinner.Done(false)
				hasErrors = true
				logger.Info(id, "Instance starting (batch) error: %v", err)
				continue
			}
		}

		logger.Info(id, "Instance started (batch)")

		spinner.Done(true)
	}

	err = CORE.SaveStates(CORE.GetStatesFilePath())

	if err != nil {
		fmtc.NewLine()
		terminal.Error(err)
		return EC_ERROR
	}

	if hasErrors {
		return EC_ERROR
	}

	return EC_OK
}

// StartAllPropCommand is "@start-all" command handler
func StartAllPropCommand(args CommandArgs) int {
	var err error

	ec := StartAllCommand(args)

	if ec != EC_OK {
		return ec
	}

	err = SC.PropagateCommand(API.COMMAND_START_ALL, -1, "")

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// checkForStartAll checks system for executing 'start-all' command
func checkForStartAll() int {
	if !CORE.HasInstances() {
		terminal.Warn("No instances are created")
		return EC_WARN
	}

	if !isSystemConfigured() {
		return EC_WARN
	}

	if !checkVirtualIP() {
		return EC_WARN
	}

	isRebooted, err := isSystemWasRebooted()

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	if !isRebooted || !CORE.IsMaster() {
		return EC_OK
	}

	terminal.Warn("Warning! It looks like you are trying to start instances after the system reboot.\n")
	terminal.Warn("This command would start ALL instances even if they didn't work before")
	terminal.Warn("the system reboot. Maybe is better to use 'restore-state' command?\n")

	ok, err := terminal.ReadAnswer("Do you really want to start ALL instances?", "N")

	if !ok || err != nil {
		return EC_CANCEL
	}

	fmtc.NewLine()

	terminal.Warn("If you have disabled data saving for instances on the master node and instances dump data")
	terminal.Warn("only on minions, it can cause FULL DATA LOSS. Please check it before proceeding.\n")
	terminal.Warn("We will continue in 10 seconds…\n")

	time.Sleep(10 * time.Second)

	ok, err = terminal.ReadAnswer("Do you want to start all instances?", "N")

	if !ok || err != nil {
		return EC_CANCEL
	}

	fmtc.NewLine()

	return EC_OK
}
