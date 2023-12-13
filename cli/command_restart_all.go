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

// RestartAllCommand is "restart-all" command handler
func RestartAllCommand(args CommandArgs) int {
	var err error

	if !CORE.HasInstances() {
		terminal.Warn("No instances are created")
		return EC_WARN
	}

	if !checkVirtualIP() {
		return EC_WARN
	}

	idList, err := CORE.GetInstanceIDListByState(CORE.INSTANCE_STATE_WORKS | CORE.INSTANCE_STATE_DEAD)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	if len(idList) == 0 {
		terminal.Warn("No instances are works")
		return EC_WARN
	}

	if !isAllRedisCompatible(idList) {
		terminal.Warn("Some instances not checked for configuration compatibility with the newly installed version of Redis.")
		terminal.Warn("We STRONGLY recommend restart instances one by one in this case.")

		fmtc.NewLine()

		ok, err := terminal.ReadAnswer("Do you want to restart all instances?", "N")

		if !ok || err != nil {
			return EC_ERROR
		}

		fmtc.NewLine()
	}

	if isSomeConfigUpdated(idList) {
		terminal.Warn("Some config files were changed and can be incompatible.")
		terminal.Warn("We STRONGLY recommend restart instances one by one in this case.")

		fmtc.NewLine()

		ok, err := terminal.ReadAnswer("Do you want to restart all instances?", "N")

		if !ok || err != nil {
			return EC_ERROR
		}

		fmtc.NewLine()
	}

	log.Info("(%s) Initiated the restarting of all working instances", CORE.User.RealName)

	hasErrors := false

	for _, id := range idList {
		meta, err := CORE.GetInstanceMeta(id)

		if err == nil {
			spinner.Show("Restarting instance %d {s}(%s){!}", id, meta.Desc)
		} else {
			spinner.Show("Restarting instance %d", id)
		}

		state, err := CORE.GetInstanceState(id, false)

		if err != nil {
			spinner.Done(false)
			terminal.Error(err)
			fmtc.NewLine()
			hasErrors = true
			continue
		}

		if !state.IsDead() {
			if state.IsWorks() {
				err = CORE.StopInstance(id, false)

				if err != nil {
					spinner.Done(false)
					terminal.Error(err)
					fmtc.NewLine()
					hasErrors = true
					continue
				}
			} else {
				continue
			}
		}

		err = CORE.StartInstance(id, true)

		if err != nil {
			spinner.Done(false)
			terminal.Error(err)
			fmtc.NewLine()
			hasErrors = true
			continue
		}

		logger.Info(id, "Instance restarted (batch)")

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

// RestartAllPropCommand is "@restart-all" command handler
func RestartAllPropCommand(args CommandArgs) int {
	var err error

	ec := RestartAllCommand(args)

	if ec != EC_OK {
		return ec
	}

	err = SC.PropagateCommand(API.COMMAND_RESTART_ALL, -1, "")

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	return EC_OK
}
