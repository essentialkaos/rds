package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/log"
	"github.com/essentialkaos/ek/v12/spinner"
	"github.com/essentialkaos/ek/v12/terminal"

	API "github.com/essentialkaos/rds/api"
	CORE "github.com/essentialkaos/rds/core"
	SC "github.com/essentialkaos/rds/sync/client"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// StopCommand is "stop" command handler
func StopCommand(args CommandArgs) int {
	err := args.Check(false)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	if !isSystemConfigured() {
		return EC_WARN
	}

	id, _, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	state, err := CORE.GetInstanceState(id, true)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	force := false

	if args.Has(1) {
		force, err = getForceArg(args.Get(1))

		if err != nil {
			terminal.Error(err.Error())
			return EC_ERROR
		}
	}

	switch {
	case state.IsStopped():
		terminal.Warn("Instance with ID %d already stopped", id)
		return EC_WARN
	case state.IsDead():
		terminal.Error("PID file for instance %d exist but service doesn't work", id)
		return EC_ERROR
	}

	if !warnAboutUnsafeAction(id, "Do you want to stop this instance?") {
		return EC_ERROR
	}

	spinner.Show("Stopping instance %d", id)
	err = CORE.StopInstance(id, force)
	spinner.Done(err == nil)

	if err != nil {
		fmtc.NewLine()
		terminal.Error(err.Error())
		return EC_ERROR
	}

	log.Info("(%s) Stopped instance with ID %d (force: %t)", CORE.User.RealName, id, force)

	err = CORE.SaveStates(CORE.GetStatesFilePath())

	if err != nil {
		fmtc.NewLine()
		terminal.Error(err.Error())
	}

	return EC_OK
}

// StopPropCommand is "@stop" command handler
func StopPropCommand(args CommandArgs) int {
	err := args.Check(false)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	id, _, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	meta, err := CORE.GetInstanceMeta(id)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	ec := StopCommand(args)

	if ec != EC_OK {
		return ec
	}

	err = SC.PropagateCommand(API.COMMAND_STOP, id, meta.UUID)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	return EC_OK
}
