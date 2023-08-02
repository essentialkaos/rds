package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"time"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/log"
	"github.com/essentialkaos/ek/v12/spinner"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
	REDIS "github.com/essentialkaos/rds/redis"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// ReplicationCommand is "replication" command handler
func ReplicationRoleSetCommand(args CommandArgs) int {
	if CORE.IsSyncDaemonActive() {
		terminal.Warn("You must stop RDS Sync daemon before reconfiguration")
		return EC_WARN
	}

	if !CORE.HasInstances() {
		terminal.Warn("No instances are created on this node. Reconfiguration is not required.")
		return EC_WARN
	}

	if !CORE.IsMinion() && !CORE.IsMaster() {
		terminal.Warn("Node must have \"master\" or \"minion\" role for reconfiguration")
		return EC_WARN
	}

	switch CORE.Config.GetS(CORE.REPLICATION_ROLE) {
	case CORE.ROLE_MASTER:
		fmtc.Println("This command will reconfigure this node from {*}MINION{!} to {*}MASTER{!} role.")
		fmtc.NewLine()
		fmtc.Println("What will be done:")
		fmtc.Println("{s}1.{!} For all instances will be disabled syncing with masters;")
		fmtc.Println("{s}2.{!} For all instances will be regenerated configuration files.")

	case CORE.ROLE_MINION:
		fmtc.Println("This command will reconfigure this node from {*}MASTER{!} to {*}MINION{!} role.")
		fmtc.NewLine()
		fmtc.Println("What will be done:")
		fmtc.Println("{s}1.{!} All instances will be stopped;")
		fmtc.Println("{s}2.{!} For all instances will be regenerated configuration files.")
	}

	fmtc.NewLine()

	ok, err := terminal.ReadAnswer("Do you OK with that?", "N")

	if err != nil || !ok {
		return EC_ERROR
	}

	fmtc.NewLine()

	var ec int

	if CORE.IsMaster() {
		ec = setRoleFromMinionToMaster()
	} else {
		ec = setRoleFromMasterToMinion()
	}

	if ec == EC_ERROR {
		return ec
	}

	fmtc.NewLine()
	fmtc.Println("{g}Node role successfully set!{!}")
	fmtc.NewLine()
	fmtc.Println("Now you can start RDS Sync daemon.")

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// setRoleFromMasterToMinion sets current role from master to minion
func setRoleFromMasterToMinion() int {
	if stopAllInstancesForRoleSet() != EC_OK {
		return EC_ERROR
	}

	fmtc.NewLine()

	if regenerateAllConfigs() != EC_OK {
		return EC_ERROR
	}

	log.Info("(%s) Reconfigurated node to minion role", CORE.User.RealName)

	return EC_OK
}

// setRoleFromMinionToMaster sets current role from minion to master
func setRoleFromMinionToMaster() int {
	if disableSyncingForAllInstances() != EC_OK {
		return EC_ERROR
	}

	fmtc.NewLine()

	if regenerateAllConfigs() != EC_OK {
		return EC_ERROR
	}

	log.Info("(%s) Reconfigurated node to master role", CORE.User.RealName)

	return EC_OK
}

// stopAllInstancesForRoleSet stops all instances for changing roles
func stopAllInstancesForRoleSet() int {
	ok, err := terminal.ReadAnswer("Do you want to stop all instances?", "N")

	if !ok || err != nil {
		return EC_ERROR
	}

	fmtc.NewLine()

	idList, err := CORE.GetInstanceIDListByState(CORE.INSTANCE_STATE_WORKS)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	return stopAllInstances(idList)
}

// disableSyncingForAllInstances disables syncing for all running instances
func disableSyncingForAllInstances() int {
	ok, err := terminal.ReadAnswer("Do you want to disable syncing for all instances?", "N")

	if !ok || err != nil {
		return EC_ERROR
	}

	fmtc.NewLine()

	idList, err := CORE.GetInstanceIDListByState(CORE.INSTANCE_STATE_WORKS)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	if len(idList) == 0 {
		terminal.Warn("No instances are works")
		return EC_OK
	}

	for _, id := range idList {
		spinner.Show("Disable saving for instance %d", id)

		resp, err := CORE.ExecCommand(id, &REDIS.Request{
			Command: []string{"REPLICAOF", "NO", "ONE"},
			Timeout: time.Minute,
		})

		if resp != nil && resp.Err != nil {
			err = resp.Err
		}

		spinner.Done(err == nil)

		if err != nil {
			terminal.Error(err.Error())
			fmtc.NewLine()
			return EC_ERROR
		}
	}

	return EC_OK
}
