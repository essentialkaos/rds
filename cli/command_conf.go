package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v12/fmtutil/table"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/pager"
	"github.com/essentialkaos/ek/v12/sliceutil"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
	REDIS "github.com/essentialkaos/rds/redis"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// FilteredProps is a slice with props that must not be shown in the command output
var FilteredProps = []string{
	"masterauth",
	"masteruser",
	"rename-command",
	"requirepass",
	"user",
}

// ////////////////////////////////////////////////////////////////////////////////// //

// ConfCommand is "conf" command handler
func ConfCommand(args CommandArgs) int {
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

	state, err := CORE.GetInstanceState(id, false)

	if err != nil {
		terminal.Error(err)
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

	if (options.GetB(OPT_PAGER) || prefs.AutoPaging) && !useRawOutput {
		if pager.Setup() == nil {
			defer pager.Complete()
		}
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

	t := table.NewTable("NAME", "VALUE")

	for _, prop := range fileConfig.Props {
		if !options.GetB(OPT_PRIVATE) && sliceutil.Contains(FilteredProps, prop) {
			continue
		}

		propFmt, found := filterConfProp(prop, filter)

		if !found {
			continue
		}

		hasData = true

		if len(diff) == 0 || prop == "user" {
			printConfProps(t, propFmt, fileConfig.Data[prop], "", true)
			continue
		}

		diffInfo := findDiffProp(diff, prop)

		switch {
		case diffInfo.PropName != "":
			printConfProps(t, propFmt, fileConfig.Data[prop], diffInfo.MemValue, false)
		case prop == "replicaof" && findDiffProp(diff, "slaveof").PropName != "":
			printConfProps(t, propFmt, fileConfig.Data[prop], findDiffProp(diff, "slaveof").MemValue, false)
		case prop == "slaveof" && findDiffProp(diff, "replicaof").PropName != "":
			printConfProps(t, propFmt, fileConfig.Data[prop], findDiffProp(diff, "replicaof").MemValue, false)
		default:
			printConfProps(t, propFmt, fileConfig.Data[prop], "", true)
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

// filterConfProp filters configuration properties
func filterConfProp(prop string, filter []string) (string, bool) {
	if len(filter) == 0 {
		return prop, true
	}

	for _, f := range filter {
		if strings.Contains(prop, f) {
			return strings.ReplaceAll(prop, f, "{_}"+f+"{!}"), true
		}
	}

	return prop, false
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
