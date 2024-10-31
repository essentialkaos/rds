package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"

	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fmtutil/panel"
	"github.com/essentialkaos/ek/v13/fmtutil/table"
	"github.com/essentialkaos/ek/v13/spinner"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/terminal/input"

	CORE "github.com/essentialkaos/rds/core"
	REDIS "github.com/essentialkaos/rds/redis"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// ReloadCommand is "reload" command handler
func ReloadCommand(args CommandArgs) int {
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
		return reloadAllConfigs()
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

	state, err := CORE.GetInstanceState(id, false)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	if !state.IsWorks() {
		terminal.Warn("Instance should work for configuration reloading")
		return EC_WARN
	}

	return reloadInstanceConfig(id)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// reloadInstanceConfig reloads config for instance with given ID
func reloadInstanceConfig(id int) int {
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

	fmtc.Println("{s}Note that some configuration properties can't be updated without instance restart.{!}\n")

	ok, err := input.ReadAnswer(
		"Do you want to reload configuration for this instance?", "N",
	)

	if err != nil || !ok {
		return EC_CANCEL
	}

	diff, err := CORE.GetInstanceConfigChanges(id)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	if isDiffContainsReplicationProp(diff) {
		panel.Warn("Replication settings is changed",
			`Instance configuration contains changed {*}REPLICAOF{!}/{*}SLAVEOF{!} properties
which can't be changed through the 'reload' command and will be ignored.
Use {*}REPLICAOF{!} or {*}SLAVEOF{!} commands for changing replication settings.`)
		fmtc.NewLine()
	}

	diff = cleanDiff(diff)

	if len(diff) == 0 {
		fmtc.Println("{g}Instance configuration is up-to-date{!}")
		return EC_OK
	}

	renderConfigsDiff(diff)

	fmtc.NewLine()

	ok, err = input.ReadAnswer(
		"Do you want to apply this settings?", "N",
	)

	if err != nil || !ok {
		return EC_CANCEL
	}

	errs := CORE.ReloadInstanceConfig(id)

	if len(errs) == 0 {
		fmtc.Println("{g}Instance configuration successfully updated{!}")
		logger.Info(id, "Instance configuration reloaded")
		return EC_OK
	}

	for _, err = range errs {
		terminal.Error(err)
	}

	logger.Error(id, "Instance configuration reloaded with errors (%d)", len(errs))

	return EC_ERROR
}

// reloadAllConfigs reloads configs for all instances
func reloadAllConfigs() int {
	fmtc.Println("{s}Note that some configuration properties can't be updated without instance restart.{!}\n")

	ok, err := input.ReadAnswer("Do you want to reload configurations for all instances?", "N")

	if err != nil || !ok {
		return EC_CANCEL
	}

	instances, err := CORE.GetInstanceIDListByState(CORE.INSTANCE_STATE_WORKS)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	if len(instances) == 0 {
		terminal.Warn("No instances are works")
		return EC_WARN
	}

	var hasErrors bool

	for _, id := range instances {
		spinner.Show("Reloading configuration for instance {*}%d{!}", id)
		errs := CORE.ReloadInstanceConfig(id)
		spinner.Done(len(errs) == 0)

		if len(errs) != 0 {
			for _, err := range errs {
				terminal.Error(err)
			}
			fmtc.NewLine()
			hasErrors = true
			logger.Error(id, "Instance configuration reloaded (batch) with errors (%d)", len(errs))
		} else {
			logger.Info(id, "Instance configuration reloaded (batch)")
		}
	}

	if hasErrors {
		return EC_ERROR
	}

	return EC_OK
}

// renderConfigsDiff renders data from configs diff
func renderConfigsDiff(diff []REDIS.ConfigPropDiff) {
	t := table.NewTable().SetHeaders("PROPERTY", "VALUE CHANGE")

	for _, info := range diff {
		switch {
		case info.FileValue == "\"\"":
			t.Add(info.PropName, fmt.Sprintf("\"%s\" → \"\"", info.MemValue))
		default:
			t.Add(info.PropName, fmt.Sprintf("\"%s\" → \"%s\"", info.MemValue, info.FileValue))
		}
	}

	t.Render()
}

// isDiffContainsReplicationProp returns true if diff contains slaveof/replicaof
// properties
func isDiffContainsReplicationProp(diff []REDIS.ConfigPropDiff) bool {
	for _, d := range diff {
		switch d.PropName {
		case "slaveof", "replicaof":
			return true
		}
	}

	return false
}

// cleanDiff removes properties which can't be changed through 'CONFIG SET'
func cleanDiff(diff []REDIS.ConfigPropDiff) []REDIS.ConfigPropDiff {
	var result []REDIS.ConfigPropDiff

	for _, d := range diff {
		switch d.PropName {
		case "slaveof", "replicaof":
			continue
		default:
			result = append(result, d)
		}
	}

	return result
}
