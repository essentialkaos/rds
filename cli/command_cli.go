package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"

	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
	RC "github.com/essentialkaos/rds/redis/client"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// CliCommand is "cli" command handler
func CliCommand(args CommandArgs) int {
	err := args.Check(true)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	id, db, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	port := CORE.GetInstancePort(id)
	meta, err := CORE.GetInstanceMeta(id)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	var disableMonitor bool

	renamings, err := CORE.GetInstanceRenamedCommands(id)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	if args.Has(1) || args.Get(1) == renamings["MONITOR"] {
		ops, err := getCurrentInstanceTraffic(id)

		if err != nil {
			terminal.Error(err.Error())
			return EC_ERROR
		}

		disableMonitor = ops > RC.MONITOR_MAX_OPS
	}

	if len(args) == 1 {
		err = RC.RunRedisCli(&RC.CLIProps{
			ID:             id,
			Port:           port,
			DB:             db,
			Password:       meta.Preferencies.Password,
			HistoryFile:    fmt.Sprintf("%s/.rediscli_history", CORE.User.HomeDir),
			Renamings:      renamings,
			DisableMonitor: disableMonitor,
			Secure:         options.GetB(OPT_PRIVATE),
		})
	} else {
		err = RC.ExecRedisCmd(&RC.CLIProps{
			Port:           port,
			DB:             db,
			Password:       meta.Preferencies.Password,
			Command:        args[1:],
			Renamings:      renamings,
			DisableMonitor: disableMonitor,
			Secure:         options.GetB(OPT_PRIVATE),
			RawOutput:      useRawOutput,
		})
	}

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	return EC_OK
}
