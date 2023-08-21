package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"strconv"
	"strings"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// CheckCommand is "check" command handler
func CheckCommand(args CommandArgs) int {
	if !CORE.HasInstances() {
		terminal.Warn("No instances are created")
		return EC_WARN
	}

	if !isSystemConfigured() {
		return EC_WARN
	}

	var hasErrors bool
	var deadInstances []string

	for _, id := range CORE.GetInstanceIDList() {
		state, err := CORE.GetInstanceState(id, false)

		if err != nil || state.IsDead() {
			deadInstances = append(deadInstances, strconv.Itoa(id))
			hasErrors = true
		}
	}

	if hasErrors {
		terminal.Error("Instances with ID's %s are dead", strings.Join(deadInstances, ","))
		return EC_ERROR
	}

	fmtc.Println("{g}All instances work fine{!}")

	return EC_OK
}
