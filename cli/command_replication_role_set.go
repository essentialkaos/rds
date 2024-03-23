package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"strings"
	"time"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/spinner"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
	REDIS "github.com/essentialkaos/rds/redis"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// ReplicationCommand is "replication" command handler
func ReplicationRoleSetCommand(args CommandArgs) int {
	if len(args) == 0 {
		terminal.Error(
			"You must specify target role (%s or %s)",
			CORE.ROLE_MASTER, CORE.ROLE_MINION,
		)
		return EC_ERROR
	}

	targetRole := strings.ToLower(args.Get(0))

	switch {
	case CORE.IsSyncDaemonActive():
		terminal.Warn("You must stop RDS Sync daemon before reconfiguration")
		return EC_WARN

	case !CORE.HasInstances():
		terminal.Warn("No instances are created on this node. Reconfiguration is not required.")
		return EC_WARN

	case targetRole != CORE.ROLE_MASTER && targetRole != CORE.ROLE_MINION:
		terminal.Error(
			"Unknown target role %s (must be %q or %q)",
			targetRole, CORE.ROLE_MASTER, CORE.ROLE_MINION,
		)
		return EC_ERROR

	case CORE.Config.GetS(CORE.REPLICATION_ROLE) == "":
		terminal.Error("Node role in configuration file is empty")
		return EC_ERROR

	case targetRole == CORE.ROLE_MASTER && !CORE.IsMaster(),
		targetRole == CORE.ROLE_MINION && !CORE.IsMinion():
		terminal.Error(
			"Target role is %q but role in configuration file is set to %q",
			targetRole, CORE.Config.GetS(CORE.REPLICATION_ROLE),
		)
		return EC_ERROR
	}

	virtualIP := CORE.Config.GetS(CORE.KEEPALIVED_VIRTUAL_IP)

	if virtualIP != "" && targetRole == CORE.ROLE_MASTER {
		switch CORE.GetKeepalivedState() {
		case CORE.KEEPALIVED_STATE_UNKNOWN:
			terminal.Error("Can't check keepalived virtual IP status")

		case CORE.KEEPALIVED_STATE_BACKUP:
			terminal.Error("This server has no keepalived virtual IP (%s).", virtualIP)
			terminal.Error("You must assign a virtual IP to this machine before changing the role to master.")
			return EC_ERROR
		}
	}

	switch targetRole {
	case CORE.ROLE_MASTER:
		fmtc.Println("This command will reconfigure this node from {*}MINION{!} to {*}MASTER{!} role.")
		fmtc.NewLine()
		fmtc.Println("What will be done:")
		fmtc.Println("{s}1.{!} Synchronization with masters will be disabled for all instances;")
		fmtc.Println("{s}2.{!} Configuration files will be regenerated for all instances.")

	case CORE.ROLE_MINION:
		fmtc.Println("This command will reconfigure this node from {*}MASTER{!} to {*}MINION{!} role.")
		fmtc.NewLine()
		fmtc.Println("What will be done:")
		fmtc.Println("{s}1.{!} All instances will be stopped;")
		fmtc.Println("{s}2.{!} Configuration files will be regenerated for all instances.")
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
	logger.Info(-1, "Started node reconfigurated from master to minion role")

	if stopAllInstancesForRoleSet() == EC_ERROR {
		return EC_ERROR
	}

	fmtc.NewLine()

	if regenerateAllConfigs() == EC_ERROR {
		logger.Error(-1, "Node reconfigurated finished with error due to problem with configuration files regeneration")
		return EC_ERROR
	}

	logger.Info(-1, "Node reconfigurated from master to minion role")

	return EC_OK
}

// setRoleFromMinionToMaster sets current role from minion to master
func setRoleFromMinionToMaster() int {
	logger.Info(-1, "Started node reconfigurated from minion to master role")

	if disableSyncingForAllInstances() == EC_ERROR {
		return EC_ERROR
	}

	fmtc.NewLine()

	ec := regenerateAllConfigs()

	switch ec {
	case EC_ERROR:
		logger.Error(-1, "Node reconfigurated finished with error due to problem with configuration files regeneration")
		return EC_ERROR
	case EC_OK:
		if reloadAllConfigs() == EC_ERROR {
			logger.Error(-1, "Node reconfigurated finished with error due to problem with configuration files reloading")
			return EC_ERROR
		}
	}

	logger.Info(-1, "Node reconfigurated from minion to master role")

	return EC_OK
}

// stopAllInstancesForRoleSet stops all instances for changing roles
func stopAllInstancesForRoleSet() int {
	idList, err := CORE.GetInstanceIDListByState(CORE.INSTANCE_STATE_WORKS)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	return stopAllInstances(idList)
}

// disableSyncingForAllInstances disables syncing for all running instances
func disableSyncingForAllInstances() int {
	idList, err := CORE.GetInstanceIDListByState(CORE.INSTANCE_STATE_WORKS)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	if len(idList) == 0 {
		terminal.Warn("No instances are works")
		return EC_OK
	}

	for _, id := range idList {
		spinner.Show("Disable syncing for instance {*}%d{!}", id)

		resp, err := CORE.ExecCommand(id, &REDIS.Request{
			Command: []string{"REPLICAOF", "NO", "ONE"},
			Timeout: time.Minute,
		})

		if resp != nil && resp.Err != nil {
			err = resp.Err
		}

		spinner.Done(err == nil)

		if err != nil {
			terminal.Error(err)
			fmtc.NewLine()
			return EC_ERROR
		}
	}

	return EC_OK
}
