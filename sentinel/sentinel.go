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
const NAME_PREFIX = "rds-"

// ////////////////////////////////////////////////////////////////////////////////// //

// InstanceConfig contains info about instance for Sentinel monitoring
type InstanceConfig struct {
	ID   int
	IP   string
	Port int

	Quorum                int
	DownAfterMilliseconds int
	FailoverTimeout       int
	ParallelSyncs         int

	Auth Auth
}

type SentinelConfig struct {
	Port int

	Auth Auth
}

// Auth contains info for authorization
type Auth struct {
	User     string
	Password string
}

// InfoItem is info key/value struct
type InfoItem struct {
	Name  string
	Value string
}

type InfoSlice []InfoItem

// Info contains info about all Sentinels and Redis instances
type Info struct {
	Master    InfoSlice
	Replicas  []InfoSlice
	Sentinels []InfoSlice
}

// ////////////////////////////////////////////////////////////////////////////////// //

// client is Redis client
var client *redy.Client

// ////////////////////////////////////////////////////////////////////////////////// //

// Monitor adds instance to Sentinel monitoring
func Monitor(sCfg *SentinelConfig, iCfg *InstanceConfig) error {
	rc := getClient(sCfg.Port, 3*time.Second)
	err := rc.Connect()

	if err != nil {
		return err
	}

	defer rc.Close()

	instanceName := getInstanceName(iCfg.ID)

	err = sentinelAuth(rc, sCfg)

	if err != nil {
		return err
	}

	resp := rc.Cmd("SENTINEL", []any{"MONITOR", instanceName, iCfg.IP, iCfg.Port, iCfg.Quorum})

	if resp.Err != nil {
		return fmt.Errorf("Can't add instance %d to monitoring: %v", iCfg.ID, resp.Err)
	}

	resp = rc.Cmd("SENTINEL", []any{"SET", instanceName, "auth-user", iCfg.Auth.User})

	if resp.Err != nil {
		return fmt.Errorf("Can't set auth-user for instance %d: %v", iCfg.ID, resp.Err)
	}

	resp = rc.Cmd("SENTINEL", []any{"SET", instanceName, "auth-pass", iCfg.Auth.Password})

	if resp.Err != nil {
		return fmt.Errorf("Can't set auth-pass for instance %d: %v", iCfg.ID, resp.Err)
	}

	err = configureFailover(rc, iCfg)

	if err != nil {
		return err
	}

	return nil
}

// CheckQuorum checks if the current Sentinel configuration is able to
// reach the quorum needed to failover a master, and the majority
// needed to authorize the failover
func CheckQuorum(sCfg *SentinelConfig, instanceID int) (string, bool) {
	cmd := []any{"CKQUORUM", getInstanceName(instanceID)}
	resp, err := execSentinelCommand(sCfg, cmd)

	if err != nil {
		return err.Error(), false
	}

	respStr, err := resp.Str()

	return respStr, err == nil
}

// Remove removes instance from Sentinel monitoring
func Remove(sCfg *SentinelConfig, instanceID int) error {
	cmd := []any{"REMOVE", getInstanceName(instanceID)}
	_, err := execSentinelCommand(sCfg, cmd)

	return err
}

// Reset sends RESET command to Sentinel
func Reset(sCfg *SentinelConfig) error {
	cmd := []any{"RESET", "*"}
	_, err := execSentinelCommand(sCfg, cmd)

	return err
}

// Failover sends FAILOVER command to Sentinel
func Failover(sCfg *SentinelConfig, instanceID int) error {
	cmd := []any{"FAILOVER", getInstanceName(instanceID)}
	_, err := execSentinelCommand(sCfg, cmd)

	return err
}

// GetMasterIP returns master IP
func GetMasterIP(sCfg *SentinelConfig, instanceID int) (string, error) {
	cmd := []any{"GET-MASTER-ADDR-BY-NAME", getInstanceName(instanceID)}
	resp, err := execSentinelCommand(sCfg, cmd)

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

// IsSentinelMonitors returns true if instance already monitored by Sentinel
func IsSentinelMonitors(sCfg *SentinelConfig, instanceID int) bool {
	cmd := []any{"MASTER", getInstanceName(instanceID)}
	_, err := execSentinelCommand(sCfg, cmd)

	return err == nil
}

// GetInfo returns info about master, replicas and sentinels
func GetInfo(sCfg *SentinelConfig, instanceID int) (*Info, error) {
	rc := getClient(sCfg.Port, 3*time.Second)
	err := rc.Connect()

	if err != nil {
		return nil, err
	}

	defer rc.Close()

	err = sentinelAuth(rc, sCfg)

	if err != nil {
		return nil, err
	}

	instanceName := getInstanceName(instanceID)

	info := &Info{
		Replicas:  make([]InfoSlice, 0),
		Sentinels: make([]InfoSlice, 0),
	}

	resp := rc.Cmd("SENTINEL", []any{"MASTER", instanceName})

	if resp.Err != nil {
		return nil, resp.Err
	}

	masterProps, _ := resp.List()
	info.Master = convertSliceToItemSlice(masterProps)

	resp = rc.Cmd("SENTINEL", []any{"REPLICAS", instanceName})

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

	resp = rc.Cmd("SENTINEL", []any{"SENTINELS", instanceName})

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

// IsEmpty returns true if auth data is empty
func (a Auth) IsEmpty() bool {
	return a.User == "" || a.Password == ""
}

// ////////////////////////////////////////////////////////////////////////////////// //

// getInstanceName generates instance name
func getInstanceName(instanceID int) string {
	return NAME_PREFIX + strconv.Itoa(instanceID)
}

// sentinelAuth authenticates on Sentinel instance
func sentinelAuth(rc *redy.Client, cfg *SentinelConfig) error {
	if !cfg.Auth.IsEmpty() {
		resp := rc.Cmd("AUTH", cfg.Auth.User, cfg.Auth.Password)

		if resp.Err != nil {
			return fmt.Errorf("Can't authenticate on Sentinel: %w", resp.Err)
		}
	}

	return nil
}

// configureFailover configures failover for instance
func configureFailover(rc *redy.Client, cfg *InstanceConfig) error {
	instanceName := getInstanceName(cfg.ID)

	resp := rc.Cmd(
		"SENTINEL",
		[]any{"SET", instanceName, "down-after-milliseconds", cfg.DownAfterMilliseconds},
	)

	if resp.Err != nil {
		return resp.Err
	}

	resp = rc.Cmd(
		"SENTINEL",
		[]any{"SET", instanceName, "failover-timeout", cfg.FailoverTimeout},
	)

	if resp.Err != nil {
		return resp.Err
	}

	resp = rc.Cmd(
		"SENTINEL",
		[]any{"SET", instanceName, "parallel-syncs", cfg.ParallelSyncs},
	)

	return resp.Err
}

// execSentinelCommand executes command on sentinel
func execSentinelCommand(cfg *SentinelConfig, command []any) (*redy.Resp, error) {
	rc := getClient(cfg.Port, 3*time.Second)
	err := rc.Connect()

	if err != nil {
		return nil, err
	}

	defer rc.Close()

	err = sentinelAuth(rc, cfg)

	if err != nil {
		return nil, err
	}

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
