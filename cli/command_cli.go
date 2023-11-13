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

	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
	RC "github.com/essentialkaos/rds/redis/client"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// CliCommand is "cli" command handler
func CliCommand(args CommandArgs) int {
	var cliCfg *RC.Config

	err := args.Check(true)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	id, db, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	port := CORE.GetInstancePort(id)

	if len(args) == 1 {
		cliCfg = &RC.Config{
			ID:          id,
			Port:        port,
			DB:          db,
			HistoryFile: fmt.Sprintf("%s/.rediscli_history", CORE.User.HomeDir),
		}
	} else {
		cliCfg = &RC.Config{
			Port:      port,
			DB:        db,
			Command:   args[1:],
			RawOutput: useRawOutput,
		}
	}

	if args.Has(1) || strings.ToUpper(args.Get(1)) == "MONITOR" {
		ops, err := getCurrentInstanceTraffic(id)

		if err != nil {
			terminal.Error(err)
			return EC_ERROR
		}

		cliCfg.DisableMonitor = ops > RC.MONITOR_MAX_OPS
	}

	meta, err := CORE.GetInstanceMeta(id)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	if options.GetB(OPT_PRIVATE) {
		cliCfg.User = CORE.REDIS_USER_ADMIN
		cliCfg.Password = meta.Preferencies.AdminPassword
	} else if meta.Preferencies.ServicePassword != "" {
		cliCfg.User = CORE.REDIS_USER_SERVICE
		cliCfg.Password = meta.Preferencies.ServicePassword
	}

	if len(args) == 1 {
		err = RC.RunRedisCli(cliCfg)
	} else {
		err = RC.ExecRedisCmd(cliCfg)
	}

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	return EC_OK
}
