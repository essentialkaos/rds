package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/terminal/input"

	API "github.com/essentialkaos/rds/api"
	CORE "github.com/essentialkaos/rds/core"
	SC "github.com/essentialkaos/rds/sync/client"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// DestroyCommand is "destroy" command handler
func DestroyCommand(args CommandArgs) int {
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

	meta, err := CORE.GetInstanceMeta(id)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	state, err := CORE.GetInstanceState(id, true)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	err = showInstanceBasicInfoCard(id, state)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	terminal.Warn("Warning! This action will delete ALL data (configuration file, data, logs).\n")

	ok, err := input.ReadAnswer("Do you want to destroy this instance?", "N")

	if !ok || err != nil {
		return EC_CANCEL
	}

	fmtc.NewLine()

	err = CORE.DestroyInstance(id)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	logger.Info(id, "Instance destroyed")
	fmtc.Printf("{*}Done. Instance with ID %d successfully destroyed.{!}\n", id)

	err = SC.PropagateCommand(API.COMMAND_DESTROY, id, meta.UUID)

	if err != nil {
		terminal.Error(err)
	}

	err = CORE.SaveStates(CORE.GetStatesFilePath())

	if err != nil {
		terminal.Error(err)
	}

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //
