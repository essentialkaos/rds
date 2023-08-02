package sentinel

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/essentialkaos/redy/v4"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// NAME_PREFIX used for instance name generation
const NAME_PREFIX = "instance"

// ////////////////////////////////////////////////////////////////////////////////// //

// Request contains data for Sentinel configuration
type Config struct {
	ID   int
	IP   string
	Port int

	Quorum                int
	DownAfterMilliseconds int
	FailoverTimeout       int
	ParallelSyncs         int

	CommandPrefix string
}

// InfoItem is info key/value struct
type InfoItem struct {
	Name  string
	Value string
}

// Info contains info about all sentinels and redis instances
type Info struct {
	Master    []InfoItem
	Replicas  [][]InfoItem
	Sentinels [][]InfoItem
}

// ////////////////////////////////////////////////////////////////////////////////// //

// client is Redis client
var client *redy.Client

// ////////////////////////////////////////////////////////////////////////////////// //

// protectedCommands is slice with commands names which is protected by prefix
var protectedCommands = []string{
	"CLIENT", "CONFIG", "SLAVEOF", "REPLICAOF",
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Monitor adds instance to Sentinel monitoring
func Monitor(port int, cfg *Config) error {
	rc := getClient(port, 3*time.Second)
	err := rc.Connect()

	if err != nil {
		return err
	}

	defer rc.Close()

	name := NAME_PREFIX + strconv.Itoa(cfg.ID)

	resp := rc.Cmd(
		"SENTINEL",
		[]any{"MONITOR", name, cfg.IP, cfg.Port, cfg.Quorum},
	)

	if resp.Err != nil {
		return resp.Err
	}

	err = configureFailover(rc, cfg)

	if err != nil {
		return err
	}

	return configureCommands(rc, cfg)
}

// CheckQuorum checks if the current Sentinel configuration is able to
// reach the quorum needed to failover a master, and the majority
// needed to authorize the failover
func CheckQuorum(port, id int) (string, bool) {
	cmd := []any{"CKQUORUM", "instance" + strconv.Itoa(id)}
	resp, err := execSentinelCommand(port, cmd)

	if err != nil {
		return err.Error(), false
	}

	respStr, err := resp.Str()

	return respStr, err == nil
}

// Remove removes instance from Sentinel monitoring
func Remove(port, id int) error {
	cmd := []any{"REMOVE", "instance" + strconv.Itoa(id)}
	_, err := execSentinelCommand(port, cmd)

	return err
}

// Reset sends RESET command to Sentinel
func Reset(port int) error {
	cmd := []any{"RESET", "*"}
	_, err := execSentinelCommand(port, cmd)

	return err
}

// Failover sends FAILOVER command to Sentinel
func Failover(port, id int) error {
	cmd := []any{"FAILOVER", "instance" + strconv.Itoa(id)}
	_, err := execSentinelCommand(port, cmd)

	return err
}

// GetMasterIP returns master IP
func GetMasterIP(port, id int) (string, error) {
	cmd := []any{"GET-MASTER-ADDR-BY-NAME", "instance" + strconv.Itoa(id)}
	resp, err := execSentinelCommand(port, cmd)

	if err != nil {
		return "", err
	}

	if resp.HasType(redy.NIL) {
		return "", errors.New("Can't find info about instance with given ID")
	}

	masterInfo, err := resp.List()

	if err != nil {
		return "", fmt.Errorf("Can't decode command response: %v", err)
	}

	if len(masterInfo) < 2 {
		return "", fmt.Errorf("Response has wrong number of values (%d ≠ 2)", len(masterInfo))
	}

	return masterInfo[0], nil
}

// IsSentinelEnabled returns true if instance already monitored by Sentinel
func IsSentinelEnabled(port, id int) bool {
	cmd := []any{"MASTER", "instance" + strconv.Itoa(id)}
	_, err := execSentinelCommand(port, cmd)

	return err == nil
}

// GetInfo returns info about master, replicas and sentinels
func GetInfo(port, id int) (*Info, error) {
	rc := getClient(port, 3*time.Second)
	err := rc.Connect()

	if err != nil {
		return nil, err
	}

	defer rc.Close()

	name := "instance" + strconv.Itoa(id)

	info := &Info{
		Replicas:  make([][]InfoItem, 0),
		Sentinels: make([][]InfoItem, 0),
	}

	resp := rc.Cmd("SENTINEL", []any{"MASTER", name})

	if resp.Err != nil {
		return nil, resp.Err
	}

	masterProps, _ := resp.List()
	info.Master = convertSliceToItemSlice(masterProps)

	resp = rc.Cmd("SENTINEL", []any{"REPLICAS", name})

	if resp.Err != nil {
		return nil, resp.Err
	}

	replicas, err := resp.Array()

	if err != nil {
		return nil, errors.New("Can't decode replicas list")
	}

	for _, replica := range replicas {
		replicaProps, _ := replica.List()
		info.Replicas = append(info.Replicas, convertSliceToItemSlice(replicaProps))
	}

	resp = rc.Cmd("SENTINEL", []any{"SENTINELS", name})

	if resp.Err != nil {
		return nil, resp.Err
	}

	sentinels, err := resp.Array()

	if err != nil {
		return nil, errors.New("Can't decode sentinels list")
	}

	for _, sentinel := range sentinels {
		sentinelProps, _ := sentinel.List()
		info.Sentinels = append(info.Sentinels, convertSliceToItemSlice(sentinelProps))
	}

	return info, nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// configureFailover configures failover for instance
func configureFailover(rc *redy.Client, cfg *Config) error {
	name := NAME_PREFIX + strconv.Itoa(cfg.ID)

	resp := rc.Cmd(
		"SENTINEL",
		[]any{"SET", name, "down-after-milliseconds", cfg.DownAfterMilliseconds},
	)

	if resp.Err != nil {
		return resp.Err
	}

	resp = rc.Cmd(
		"SENTINEL",
		[]any{"SET", name, "failover-timeout", cfg.FailoverTimeout},
	)

	if resp.Err != nil {
		return resp.Err
	}

	resp = rc.Cmd(
		"SENTINEL",
		[]any{"SET", name, "parallel-syncs", cfg.ParallelSyncs},
	)

	return resp.Err
}

// configureCommands configures renamed commands for instance
func configureCommands(rc *redy.Client, cfg *Config) error {
	if cfg.CommandPrefix == "" {
		return nil
	}

	var resp *redy.Resp

	name := NAME_PREFIX + strconv.Itoa(cfg.ID)

	for _, cmd := range protectedCommands {
		resp = rc.Cmd(
			"SENTINEL",
			[]any{"SET", name, "rename-command", cmd, cfg.CommandPrefix + "_" + cmd},
		)

		if resp.Err != nil {
			return resp.Err
		}
	}

	return nil
}

// execSentinelCommand executes command on sentinel
func execSentinelCommand(port int, command []any) (*redy.Resp, error) {
	rc := getClient(port, 3*time.Second)
	err := rc.Connect()

	if err != nil {
		return nil, err
	}

	defer rc.Close()

	resp := rc.Cmd("SENTINEL", command)

	return resp, resp.Err
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

// convertSliceToItemSlice converts slice with info to key/value slice
func convertSliceToItemSlice(s []string) []InfoItem {
	var result []InfoItem

	for i := 0; i < len(s); i += 2 {
		result = append(result, InfoItem{s[i], s[i+1]})
	}

	return result
}
