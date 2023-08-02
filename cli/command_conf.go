package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"time"

	"github.com/essentialkaos/ek/v12/fmtutil/table"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/sliceutil"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
	REDIS "github.com/essentialkaos/rds/redis"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// ConfCommand is "conf" command handler
func ConfCommand(args CommandArgs) int {
	err := args.Check(false)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	id, _, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	state, err := CORE.GetInstanceState(id, false)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	var fileConfig, memConfig *REDIS.Config
	var diff []REDIS.ConfigPropDiff

	fileConfig, err = CORE.ReadInstanceConfig(id)

	if err != nil {
		terminal.Error("Can't read configuration file data: %v", err)
		return EC_ERROR
	}

	if state.IsWorks() {
		memConfig, err = CORE.GetInstanceConfig(id, 3*time.Second)

		if err != nil {
			terminal.Error("Can't read in-memory configuration: %v", err)
			return EC_ERROR
		}

		diff = REDIS.GetConfigsDiff(fileConfig, memConfig)
	}

	if len(args) > 1 {
		printConfInfo(fileConfig, diff, args[1:])
	} else {
		printConfInfo(fileConfig, diff, nil)
	}

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// printConfInfo shows difference between file and in-memory config
func printConfInfo(fileConfig *REDIS.Config, diff []REDIS.ConfigPropDiff, filter []string) {
	hasData := false
	hasFilter := len(filter) != 0
	t := table.NewTable("NAME", "VALUE")

	for _, prop := range fileConfig.Props {
		if prop == "always-show-logo" {
			continue
		}

		if !options.GetB(OPT_PRIVATE) {
			switch prop {
			case "rename-command", "requirepass", "masterauth":
				continue
			}
		}

		if hasFilter && !sliceutil.Contains(filter, prop) {
			continue
		}

		hasData = true

		if len(diff) == 0 {
			printConfProps(t, prop, fileConfig.Data[prop], "", true)
			continue
		}

		diffInfo := findDiffProp(diff, prop)

		switch {
		case diffInfo.PropName != "":
			printConfProps(t, prop, fileConfig.Data[prop], diffInfo.MemValue, false)
		case prop == "replicaof" && findDiffProp(diff, "slaveof").PropName != "":
			printConfProps(t, prop, fileConfig.Data[prop], findDiffProp(diff, "slaveof").MemValue, false)
		case prop == "slaveof" && findDiffProp(diff, "replicaof").PropName != "":
			printConfProps(t, prop, fileConfig.Data[prop], findDiffProp(diff, "replicaof").MemValue, false)
		default:
			printConfProps(t, prop, fileConfig.Data[prop], "", true)
		}
	}

	if !hasData {
		terminal.Warn("Can't find any properties with given name")
		return
	}

	t.Render()
}

// printConfProps prints config property and value
func printConfProps(t *table.Table, prop string, values []string, curValue string, isEmpty bool) {
	if !isEmpty && curValue == "" {
		curValue = "\"\""
	}

	for i, value := range values {
		switch {
		case i == 0 && curValue != "":
			t.Add(prop, fmt.Sprintf("{s-}%s{!} {y}(%s){!}", value, curValue))
		case i != 0 && curValue != "":
			t.Add(prop, fmt.Sprintf("{s-}%s{!}", value))
		default:
			t.Add(prop, value)
		}
	}
}

// findDiffProp tries to find changed properties
func findDiffProp(diff []REDIS.ConfigPropDiff, prop string) REDIS.ConfigPropDiff {
	for _, d := range diff {
		if d.PropName == prop {
			return d
		}
	}

	return REDIS.ConfigPropDiff{}
}
