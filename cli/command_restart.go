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

	API "github.com/essentialkaos/rds/api"
	CORE "github.com/essentialkaos/rds/core"
	SC "github.com/essentialkaos/rds/sync/client"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// RestartCommand is "restart" command handler
func RestartCommand(args CommandArgs) int {
	err := args.Check(false)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	if !checkVirtualIP() {
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

	if !warnAboutUnsafeAction(id, "Do you want to restart this instance?") {
		return EC_OK
	}

	force := false

	if state.IsWorks() {
		if args.Has(1) {
			force, err = getForceArg(args.Get(1))

			if err != nil {
				terminal.Error(err)
				return EC_ERROR
			}
		}

		spinner.Show("Stopping instance {*}%d{!}", id)
		err = CORE.StopInstance(id, force)
		spinner.Done(err == nil)

		if err != nil {
			fmtc.NewLine()

			terminal.Error(err)
			err = CORE.SaveStates(CORE.GetStatesFilePath())

			if err != nil {
				terminal.Error(err)
			}

			return EC_ERROR
		}
	}

	spinner.Show("Starting instance {*}%d{!}", id)

	err = CORE.StartInstance(id, false)

	spinner.Done(err == nil)

	if err != nil {
		fmtc.NewLine()

		terminal.Error(err)
		err = CORE.SaveStates(CORE.GetStatesFilePath())

		if err != nil {
			terminal.Error(err)
		}

		return EC_ERROR
	}

	logger.Info(id, "Instance restarted (force: %t)", force)

	return EC_OK
}

// RestartPropCommand is "@restart" command handler
func RestartPropCommand(args CommandArgs) int {
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

	meta, err := CORE.GetInstanceMeta(id)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	ec := RestartCommand(args)

	if ec != EC_OK {
		return ec
	}

	err = SC.PropagateCommand(API.COMMAND_RESTART, id, meta.UUID)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	return EC_OK
}
