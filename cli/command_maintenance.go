package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"strings"

	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/terminal"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// MaintenanceCommand is "maintenance" command handler
func MaintenanceCommand(args CommandArgs) int {
	if len(args) == 0 {
		terminal.Error("You must specify maintenance state (enable/disable or yes/no)")
		return EC_ERROR
	}

	var err error

	switch strings.ToLower(args.Get(0)) {
	case "enable", "yes":
		err = createMaintenanceLock()
		if err == nil {
			fmtc.Println("Maintenance mode is {g}enabled{!}")
			logger.Info(-1, "Maintenance mode enabled")
		}

	case "disable", "no":
		err = removeMaintenanceLock()
		if err == nil {
			fmtc.Println("Maintenance mode is {y}disabled{!}")
			logger.Info(-1, "Maintenance mode disabled")
		}

	default:
		terminal.Error("Unknown value \"%s\"", args.Get(0))
		return EC_ERROR
	}

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //
