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

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// RegenCommand is "regen" command handler
func RegenCommand(args CommandArgs) int {
	if !CORE.HasInstances() {
		terminal.Warn("No instances are created")
		return EC_WARN
	}

	if len(args) == 0 {
		terminal.Error("You must define instance ID or \"all\"")
		return EC_ERROR
	}

	instanceID := args.Get(0)

	if instanceID == "*" || instanceID == "all" {
		return regenerateAllConfigs()
	}

	id, _, err := CORE.ParseIDDBPair(instanceID)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	if !CORE.IsInstanceExist(id) {
		terminal.Error("Instance with ID %d does not exist", id)
		return EC_ERROR
	}

	return regenerateInstanceConfig(id)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// regenerateInstanceConfig regenerates configuration file for instance with given ID
func regenerateInstanceConfig(id int) int {
	state, err := CORE.GetInstanceState(id, true)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	err = showInstanceBasicInfoCard(id, state)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	ok, err := terminal.ReadAnswer(
		"Do you want to regenerate configuration file for this instance?", "N",
	)

	if !ok || err != nil {
		return EC_ERROR
	}

	fmtc.NewLine()

	spinner.Show("Regenerating configuration file for instance %d", id)
	err = CORE.RegenerateInstanceConfig(id)
	spinner.Done(err == nil)

	if err != nil {
		fmtc.NewLine()
		terminal.Error(err.Error())
		return EC_ERROR
	}

	log.Info("(%s) Regenerated configuration file for instance %d", CORE.User.RealName, id)

	return EC_OK
}

func regenerateAllConfigs() int {
	ok, err := terminal.ReadAnswer("Do you want to regenerate configuration files for all instances?", "N")

	if !ok || err != nil {
		return EC_ERROR
	}

	fmtc.NewLine()

	for _, id := range CORE.GetInstanceIDList() {
		spinner.Show("Regenerating configuration file for instance %d", id)
		err := CORE.RegenerateInstanceConfig(id)
		spinner.Done(err == nil)

		if err != nil {
			fmtc.NewLine()
			terminal.Error(err.Error())
			return EC_ERROR
		}
	}

	log.Info("(%s) Regenerated configuration files for all instances", CORE.User.RealName)

	return EC_OK
}
