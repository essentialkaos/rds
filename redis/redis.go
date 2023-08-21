package redis

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v12/mathutil"
	"github.com/essentialkaos/ek/v12/sliceutil"
	"github.com/essentialkaos/ek/v12/system/process"

	"github.com/essentialkaos/redy/v4"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	STR_SIMPLE RespType = 1 << iota
	STR_BULK
	INT
	ARRAY
	NIL

	ERR_IO
	ERR_REDIS

	STR = STR_SIMPLE | STR_BULK
	ERR = ERR_IO | ERR_REDIS
)

// ////////////////////////////////////////////////////////////////////////////////// //

type Resp = redy.Resp

type Info = redy.Info

type Config = redy.Config

type RespType = redy.RespType

type ConfigPropDiff struct {
	PropName  string
	FileValue string
	MemValue  string
}

// ////////////////////////////////////////////////////////////////////////////////// //

type Request struct {
	Command []string
	Auth    Auth
	Port    int
	DB      int
	Timeout time.Duration
}

type Auth struct {
	User     string
	Password string
}

// ////////////////////////////////////////////////////////////////////////////////// //

var ErrNotEnoughArgs error = errors.New("Not enough command arguments")

// ////////////////////////////////////////////////////////////////////////////////// //

var filteredProps = []string{
	"masterauth",
	"masteruser",
	"rename-command",
	"requirepass",
	"user",
}

var client *redy.Client

// ////////////////////////////////////////////////////////////////////////////////// //

// ExecCommand executes some command on redis instance
func ExecCommand(req *Request) (*redy.Resp, error) {
	if len(req.Command) == 0 {
		return nil, ErrNotEnoughArgs
	}

	return execCmd(req)
}

// ReadConfig read and parse redis config file
func ReadConfig(file string) (*redy.Config, error) {
	return redy.ReadConfig(file)
}

// GetConfig read and parse in-memory config
func GetConfig(req *Request) (*redy.Config, error) {
	resp, err := execCmd(req)

	if err != nil {
		return nil, err
	}

	return redy.ParseConfig(resp)
}

// GetInfo executes INFO command and parse output to struct
func GetInfo(req *Request) (*redy.Info, error) {
	resp, err := execCmd(req)

	if err != nil {
		return nil, err
	}

	info, err := redy.ParseInfo(resp)

	if err != nil {
		return nil, err
	}

	AppendSwapInfo(info)

	return info, nil
}

// GetConfigsDiff returns difference between file and in memory configurations
func GetConfigsDiff(fileConfig, memConfig *Config) []ConfigPropDiff {
	var result []ConfigPropDiff

	if fileConfig == nil || memConfig == nil {
		return nil
	}

	props := append(memConfig.Props, fileConfig.Props...)
	sort.Strings(props)
	props = sliceutil.Deduplicate(props)

	for _, prop := range props {
		fp, mp := fileConfig.Get(prop), memConfig.Get(prop)

		if prop == "slaveof" || prop == "replicaof" {
			if prop == "replicaof" && fileConfig.Has("replicaof") && memConfig.Has("slaveof") {
				continue
			}

			if prop == "slaveof" && fileConfig.Has("slaveof") && memConfig.Has("replicaof") {
				continue
			}
		}

		if sliceutil.Contains(filteredProps, prop) {
			continue
		}

		switch prop {
		case "slaveof", "replicaof":
			fp = getConfPropAny(fileConfig, "slaveof", "replicaof")
			mp = getConfPropAny(memConfig, "slaveof", "replicaof")
		case "client-output-buffer-limit":
			fp = strings.ReplaceAll(fp, " slave ", " replica ")
			mp = strings.ReplaceAll(mp, " slave ", " replica ")
		}

		switch {
		case fp == "\"\"" && mp == "":
			continue
		case fp != "" && strings.Trim(fp, `"`) != strings.Trim(mp, `"`):
			result = append(
				result, ConfigPropDiff{
					PropName:  prop,
					MemValue:  mp,
					FileValue: fp,
				},
			)
		}
	}

	return result
}

// AppendSwapInfo appends info about swap usage to basic redis info
func AppendSwapInfo(info *redy.Info) {
	pid := info.GetI("Server", "process_id")

	if pid == -1 {
		return
	}

	memInfo, err := process.GetMemInfo(pid)

	if err != nil {
		return
	}

	if len(info.Sections["Memory"].Fields) < 3 {
		info.Sections["Memory"].Fields = append(
			info.Sections["Memory"].Fields,
			"used_memory_swap",
			"used_memory_swap_human",
		)
	} else {
		info.Sections["Memory"].Fields = append(
			info.Sections["Memory"].Fields[:2],
			append(
				[]string{"used_memory_swap", "used_memory_swap_human"},
				info.Sections["Memory"].Fields[2:]...,
			)...,
		)
	}

	info.Sections["Memory"].Values["used_memory_swap"] = strconv.FormatUint(memInfo.VmSwap, 10)
	info.Sections["Memory"].Values["used_memory_swap_human"] = getHumanSize(memInfo.VmSwap)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// IsEmpty returns true if auth data is empty
func (a Auth) IsEmpty() bool {
	return a.User == "" || a.Password == ""
}

// ////////////////////////////////////////////////////////////////////////////////// //

// execCmd executes command on instance
func execCmd(req *Request) (*redy.Resp, error) {
	rc := getClient(req.Port, req.Timeout)
	err := rc.Connect()

	if err != nil {
		return nil, err
	}

	defer rc.Close()

	var resp *redy.Resp

	if !req.Auth.IsEmpty() {
		resp = rc.Cmd("AUTH", req.Auth.User, req.Auth.Password)

		if resp.Err != nil {
			return nil, resp.Err
		}
	}

	if req.DB != 0 {
		resp = rc.Cmd("SELECT", req.DB)

		if resp.Err != nil {
			return nil, resp.Err
		}
	}

	switch len(req.Command) {
	case 1:
		resp = rc.Cmd(req.Command[0])
	default:
		resp = rc.Cmd(req.Command[0], sliceutil.StringToInterface(req.Command[1:]))
	}

	if resp.Err != nil {
		return nil, resp.Err
	}

	return resp, nil
}

// getClient returns Redy client
func getClient(port int, timeout time.Duration) *redy.Client {
	if client == nil {
		client = &redy.Client{}
	}

	client.Addr = "127.0.0.1:" + strconv.Itoa(port)

	if timeout > 0 {
		client.WriteTimeout = timeout
		client.ReadTimeout = timeout
	} else {
		client.WriteTimeout = 3 * time.Second
		client.ReadTimeout = 3 * time.Second
	}

	return client
}

// getRenamedCommand returns command with prefix
func getRenamedCommand(rc *redy.Client, rn map[string]string, command string) string {
	renamedCommand, ok := rn[strings.ToUpper(command)]

	if !ok {
		return command
	}

	// Check that renamed command is supported
	resp, err := rc.Cmd("COMMAND", "INFO", renamedCommand).Array()

	if err != nil || len(resp) == 0 || resp[0].HasType(NIL) {
		return command
	}

	return renamedCommand
}

// getHumanSize returns size in human readable format
func getHumanSize(size uint64) string {
	f := float64(size)

	switch {
	case f >= 1073741824:
		return fmt.Sprintf("%g", formatFloat(f/1073741824)) + "G"
	case f >= 1048576:
		return fmt.Sprintf("%g", formatFloat(f/1048576)) + "M"
	case f >= 1024:
		return fmt.Sprintf("%g", formatFloat(f/1024)) + "K"
	default:
		return fmt.Sprintf("%d", size) + "B"
	}
}

// formatFloat formats floating numbers
func formatFloat(f float64) float64 {
	if f < 10.0 {
		return mathutil.Round(f, 2)
	}

	return mathutil.Round(f, 1)
}

// getConfPropAny returns value for first non-empty property
func getConfPropAny(config *Config, props ...string) string {
	for _, p := range props {
		v := config.Get(p)

		if v != "" {
			return v
		}
	}

	return ""
}
