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

// StartCommand is "start" command handler
func StartCommand(args CommandArgs) int {
	err := args.Check(false)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	if !isSystemConfigured() {
		return EC_WARN
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

	if state.IsWorks() {
		terminal.Warn("Instance with ID %d already works", id)
		return EC_WARN
	}

	meta, err := CORE.GetInstanceMeta(id)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	spinner.Show("Starting instance %d {s}(%s){!}", id, meta.Desc)
	err = CORE.StartInstance(id, false)
	spinner.Done(err == nil)

	if err != nil {
		fmtc.NewLine()
		terminal.Error(err)
		logger.Error(id, "Instance starting error: %v", err)
		return EC_ERROR
	}

	logger.Info(id, "Instance started")

	err = CORE.SaveStates(CORE.GetStatesFilePath())

	if err != nil {
		fmtc.NewLine()
		terminal.Error(err)
	}

	return EC_OK
}

// StartPropCommand is "@start" command handler
func StartPropCommand(args CommandArgs) int {
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

	ec := StartCommand(args)

	if ec != EC_OK {
		return ec
	}

	err = SC.PropagateCommand(API.COMMAND_START, id, meta.UUID)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	return EC_OK
}
