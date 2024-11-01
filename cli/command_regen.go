package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/spinner"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/terminal/input"

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
		terminal.Error(err)
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
		terminal.Error(err)
		return EC_ERROR
	}

	err = showInstanceBasicInfoCard(id, state)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	ok, err := input.ReadAnswer(
		"Do you want to regenerate configuration file for this instance?", "N",
	)

	if err != nil || !ok {
		return EC_CANCEL
	}

	meta, err := CORE.GetInstanceMeta(id)

	if err == nil {
		spinner.Show("Regenerating configuration file for instance {*}%d{!} {s}(%s){!}", id, meta.Desc)
	} else {
		spinner.Show("Regenerating configuration file for instance {*}%d{!}", id)
	}

	err = CORE.RegenerateInstanceConfig(id)

	spinner.Done(err == nil)

	if err != nil {
		fmtc.NewLine()
		terminal.Error(err)
		logger.Error(id, "Configuration file regeneration error: %v", err)
		return EC_ERROR
	}

	logger.Info(id, "Configuration file regenerated")

	return EC_OK
}

// regenerateAllConfigs regenerates configuration files for all instances
func regenerateAllConfigs() int {
	ok, err := input.ReadAnswer("Do you want to regenerate configuration files for all instances?", "N")

	if err != nil || !ok {
		return EC_CANCEL
	}

	for _, id := range CORE.GetInstanceIDList() {
		meta, err := CORE.GetInstanceMeta(id)

		if err == nil {
			spinner.Show("Regenerating configuration file for instance {*}%d{!} {s}(%s){!}", id, meta.Desc)
		} else {
			spinner.Show("Regenerating configuration file for instance {*}%d{!}", id)
		}

		err = CORE.RegenerateInstanceConfig(id)

		spinner.Done(err == nil)

		if err != nil {
			fmtc.NewLine()
			terminal.Error(err)
			logger.Error(id, "Configuration file regeneration (batch) error: %v", err)
			return EC_ERROR
		} else {
			logger.Info(id, "Configuration file regenerated (batch)")
		}
	}

	return EC_OK
}
