package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/spinner"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// KillCommand is "kill" command handler
func KillCommand(args CommandArgs) int {
	err := args.Check(false)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	if !isSystemConfigured() {
		return EC_WARN
	}

	id, _, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	state, err := CORE.GetInstanceState(id, false)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	if state.IsStopped() {
		terminal.Warn("Instance with ID %d already stopped", id)
		return EC_WARN
	}

	err = showInstanceBasicInfoCard(id, state)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	ok, err := terminal.ReadAnswer("Do you want to kill this instance?", "N")

	if !ok || err != nil {
		return EC_CANCEL
	}

	fmtc.NewLine()

	spinner.Show("Killing instance %d", id)
	err = CORE.KillInstance(id)

	spinner.Done(err == nil)

	if err != nil {
		fmtc.NewLine()
		terminal.Error(err)
		return EC_ERROR
	}

	logger.Info(id, "Instance killed")

	err = CORE.SaveStates(CORE.GetStatesFilePath())

	if err != nil {
		terminal.Error(err)
	}

	return EC_OK
}
