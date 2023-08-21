package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bytes"
	"strconv"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/fmtutil/table"
	"github.com/essentialkaos/ek/v12/strutil"
	"github.com/essentialkaos/ek/v12/terminal"
	"github.com/essentialkaos/ek/v12/timeutil"

	CORE "github.com/essentialkaos/rds/core"
	REDIS "github.com/essentialkaos/rds/redis"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// ClientsCommand is "clients" command handler
func ClientsCommand(args CommandArgs) int {
	err := args.Check(true)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	id, _, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	meta, err := CORE.GetInstanceMeta(id)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	req := &REDIS.Request{
		Command: []string{"CLIENT", "LIST", "TYPE", "NORMAL"},
		Port:    CORE.GetInstancePort(id),
		Auth:    REDIS.Auth{CORE.REDIS_USER_ADMIN, meta.Preferencies.AdminPassword},
		Timeout: time.Second,
	}

	resp, err := REDIS.ExecCommand(req)

	if err != nil {
		terminal.Error("Error while executing request: %v", err)
		return EC_ERROR
	}

	clientsData, _ := resp.Str()
	printClientsInfo(clientsData, args.Get(1))

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// printClientsInfo prints info about connected clients
func printClientsInfo(clientsData, filter string) {
	buf := bytes.NewBufferString(clientsData)
	t := table.NewTable().SetHeaders(
		"ID", "NAME", "USER", "ADDR", "FD", "DB", "FLAGS",
		"SUB", "PSUB", "EVENTS", "AGE", "IDLE", "CMD",
	)

	for {
		line, err := buf.ReadString('\n')

		if err != nil {
			break
		}

		info := parseFieldsLine(line, " ")
		sub, _ := strconv.Atoi(info["sub"])
		psub, _ := strconv.Atoi(info["psub"])
		age, _ := strconv.Atoi(info["age"])
		idle, _ := strconv.Atoi(info["idle"])
		cmd := strings.ToUpper(strings.ReplaceAll(info["cmd"], "|", " "))

		if filter != "" {
			switch {
			case info["name"] == filter, info["user"] == filter,
				strings.HasPrefix(info["addr"], filter+":"):
				// passthru
			default:
				continue
			}
		}

		t.Add(
			info["id"],
			strutil.Q(info["name"], "{s-}—{!}"),
			strutil.Q(info["user"], "{s-}—{!}"),
			info["addr"],
			info["fd"],
			info["db"],
			info["flags"],
			fmtutil.PrettyNum(sub),
			fmtutil.PrettyNum(psub),
			info["events"],
			timeutil.PrettyDurationSimple(age),
			timeutil.PrettyDurationSimple(idle),
			cmd,
		)
	}

	if !t.HasData() {
		terminal.Warn("There is no clients data")
		return
	}

	t.Render()
}
