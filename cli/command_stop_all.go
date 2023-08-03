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

// StopAllCommand is "stop-all" command handler
func StopAllCommand(args CommandArgs) int {
	var err error

	if !CORE.HasInstances() {
		terminal.Warn("No instances are created")
		return EC_WARN
	}

	if !isSystemConfigured() {
		return EC_WARN
	}

	idList, err := CORE.GetInstanceIDListByState(CORE.INSTANCE_STATE_WORKS | CORE.INSTANCE_STATE_DEAD)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	if len(idList) == 0 {
		terminal.Warn("No instances are works")
		return EC_WARN
	}

	if !isAllRedisCompatible(idList) {
		terminal.Warn("Some instances not checked for configuration compatibility with the newly installed version of Redis.")
		terminal.Warn("We STRONGLY recommend stop instances one by one in this case.")

		fmtc.NewLine()

		ok, err := terminal.ReadAnswer("Do you want to stop all instances?", "N")

		if !ok || err != nil {
			return EC_ERROR
		}

		fmtc.NewLine()
	}

	if isSomeConfigUpdated(idList) {
		terminal.Warn("Some configuration files were changed and can be incompatible.")
		terminal.Warn("We STRONGLY recommend stop instances one by one in this case.")

		fmtc.NewLine()

		ok, err := terminal.ReadAnswer("Do you want to stop all instances?", "N")

		if !ok || err != nil {
			return EC_ERROR
		}

		fmtc.NewLine()
	}

	return stopAllInstances(idList)
}

// StopAllPropCommand is "@stop-all" command handler
func StopAllPropCommand(args CommandArgs) int {
	var err error

	ec := StopAllCommand(args)

	if ec != EC_OK {
		return ec
	}

	err = SC.PropagateCommand(API.COMMAND_STOP_ALL, -1, "")

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// stopAllInstances stops all instances from given slice
func stopAllInstances(idList []int) int {
	var hasErrors bool

	if len(idList) == 0 {
		terminal.Warn("No instances are works")
		return EC_OK
	}

	log.Info("(%s) Initiated the stopping of all working instances", CORE.User.RealName)

	for _, id := range idList {
		meta, err := CORE.GetInstanceMeta(id)

		if err == nil {
			spinner.Show("Stopping instance %d {s}(%s){!}", id, meta.Desc)
		} else {
			spinner.Show("Stopping instance %d", id)
		}

		err = CORE.StopInstance(id, false)

		if err != nil {
			spinner.Done(false)
			hasErrors = true
			continue
		}

		spinner.Done(true)
	}

	log.Info("(%s) Stopped all working instances", CORE.User.RealName)

	err := CORE.SaveStates(CORE.GetStatesFilePath())

	if err != nil {
		fmtc.NewLine()
		terminal.Error(err.Error())
		return EC_ERROR
	}

	if hasErrors {
		return EC_ERROR
	}

	return EC_OK
}
