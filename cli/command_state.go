package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"path"
	"time"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/spinner"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// SaveStateCommand is "state-save" command handler
func SaveStateCommand(args CommandArgs) int {
	if !CORE.HasInstances() {
		terminal.Warn("No instances are created")
		return EC_WARN
	}

	if len(args) == 0 {
		terminal.Error("You must define target file")
		return EC_ERROR
	}

	statesFile := args.Get(0)

	if fsutil.IsExist(statesFile) {
		terminal.Error("File %s already exist. You can't save state to this file.", statesFile)
		return EC_ERROR
	}

	spinner.Show("Saving instances states")
	err := CORE.SaveStates(statesFile)

	if err != nil {
		spinner.Done(false)
		fmtc.NewLine()
		terminal.Error(err.Error())
		return EC_ERROR
	}

	spinner.Done(true)

	logger.Info(-1, "States saved to file %s", statesFile)

	return EC_OK
}

// RestoreStateCommand is "state-restore" command handler
func RestoreStateCommand(args CommandArgs) int {
	ec := checkForRestoreState(args)

	if ec != EC_OK {
		return ec
	}

	var err error
	var statesFile string

	switch len(args) {
	case 0:
		statesFile = CORE.GetStatesFilePath()
	default:
		statesFile = args.Get(0)
	}

	if !fsutil.CheckPerms("FRS", statesFile) {
		terminal.Error("States file %s does not exist or empty", statesFile)
		return EC_ERROR
	}

	ok, err := terminal.ReadAnswer(
		fmtc.Sprintf("Do you want to restore states from file %s?", statesFile), "N",
	)

	if !ok || err != nil {
		return EC_CANCEL
	}

	fmtc.NewLine()

	statesInfo, err := CORE.ReadStates(statesFile)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	logger.Info(-1, "Started states restoring from file %s", path.Clean(statesFile))

	restored := false

	for _, stateInfo := range statesInfo.States {
		id := stateInfo.ID

		if !CORE.IsInstanceExist(id) {
			continue
		}

		spinner.Show("Restoring instance %d state", id)
		state, err := CORE.GetInstanceState(id, false)

		if err != nil {
			spinner.Done(false)
			continue
		}

		if state == stateInfo.State || (state.IsDead() && stateInfo.State.IsStopped()) {
			spinner.Skip()
			continue
		}

		switch {
		case stateInfo.State.IsWorks():
			err = CORE.StartInstance(id, true)
		case stateInfo.State.IsStopped():
			err = CORE.StopInstance(id, false)
		}

		spinner.Done(err == nil)

		if err == nil {
			restored = true
			logger.Info(id, "Instance state restored")
		} else {
			logger.Info(id, "Instance state restored with error: %v", err)
		}
	}

	if !restored {
		logger.Info(-1, "No actions were made while states restoring")
		return EC_OK
	}

	err = CORE.SaveStates(CORE.GetStatesFilePath())

	if err != nil {
		fmtc.NewLine()
		terminal.Error(err.Error())
		return EC_ERROR
	}

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// checkForRestoreState checks system for executing 'state-restore' command
func checkForRestoreState(args CommandArgs) int {
	if !isSystemConfigured() {
		return EC_WARN
	}

	isRebooted, err := isSystemWasRebooted()

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	if !isRebooted || !CORE.IsMaster() {
		return EC_OK
	}

	terminal.Warn("Warning! It looks like you are trying to start instances after the system reboot.\n")
	terminal.Warn("If you have disabled data saving for instances on the master node and instances dump data")
	terminal.Warn("only on minions, it can cause FULL DATA LOSS. Please check it before proceeding.\n")
	terminal.Warn("We will continue in 10 seconds…\n")

	time.Sleep(10 * time.Second)

	ok, err := terminal.ReadAnswer("Do you want to restore state for all instances?", "N")

	if !ok || err != nil {
		return EC_CANCEL
	}

	fmtc.NewLine()

	return EC_OK
}
