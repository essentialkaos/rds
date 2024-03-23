package api

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"github.com/essentialkaos/ek/v12/req"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type MasterCommand string

const (
	COMMAND_CREATE         MasterCommand = "create"
	COMMAND_DESTROY        MasterCommand = "destroy"
	COMMAND_EDIT           MasterCommand = "edit"
	COMMAND_START          MasterCommand = "start"
	COMMAND_STOP           MasterCommand = "stop"
	COMMAND_RESTART        MasterCommand = "restart"
	COMMAND_START_ALL      MasterCommand = "start-all"
	COMMAND_STOP_ALL       MasterCommand = "stop-all"
	COMMAND_RESTART_ALL    MasterCommand = "restart-all"
	COMMAND_SENTINEL_START MasterCommand = "sentinel-start"
	COMMAND_SENTINEL_STOP  MasterCommand = "sentinel-stop"
)

type ClientState uint8

const (
	STATE_UNKNOWN ClientState = iota
	STATE_ONLINE
	STATE_POSSIBLE_DOWN
	STATE_DOWN
	STATE_SYNCING
	STATE_DEAD
)

type CoreCompatibility uint8

const (
	CORE_COMPAT_OK CoreCompatibility = iota
	CORE_COMPAT_PARTIAL
	CORE_COMPAT_ERROR
)

// CommandQueue is command queue
type CommandQueue struct {
	Items   []*CommandQueueItem `json:"items"`    // Items with actions
	ModTime int64               `json:"mod_time"` // Time of latest action
}

type CommandQueueItem struct {
	Command      MasterCommand `json:"command"`
	InstanceID   int           `json:"instance_id"`
	InstanceUUID string        `json:"instance_uuid"`
	Initiator    string        `json:"initiator"`
	Timestamp    int64         `json:"timestamp"`
}

type ReplicationInfo struct {
	Master       *MasterInfo   `json:"master"`
	Clients      []*ClientInfo `json:"clients"`
	SuppliantCID string        `json:"suppliant_cid,omitempty"`
}

type MasterInfo struct {
	Version  string `json:"version"`
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
}

type ClientInfo struct {
	CID            string      `json:"cid"`
	Role           string      `json:"role"`
	Version        string      `json:"version"`
	Hostname       string      `json:"hostname"`
	IP             string      `json:"ip"`
	ConnectionDate int64       `json:"connection_date"`
	LastSeenLag    float64     `json:"last_seen_lag"`
	LastSyncLag    float64     `json:"last_sync_lag"`
	State          ClientState `json:"state"`
}

type StatsInfo struct {
	Minions    int     `json:"minions"`
	Sentinels  int     `json:"sentinels"`
	MaxSeenLag float64 `json:"max_seen_lag"`
	MaxSyncLag float64 `json:"max_sync_lag"`
}

// ////////////////////////////////////////////////////////////////////////////////// //

type StatusCode uint8

const (
	STATUS_OK                        StatusCode = 0
	STATUS_WRONG_REQUEST             StatusCode = 1
	STATUS_WRONG_AUTH_TOKEN          StatusCode = 2
	STATUS_UNKNOWN_CLIENT            StatusCode = 3
	STATUS_WRONG_METHOD              StatusCode = 4
	STATUS_WRONG_ARGS                StatusCode = 5
	STATUS_INCORRECT_REQUEST         StatusCode = 6
	STATUS_UNKNOWN_INSTANCE          StatusCode = 7
	STATUS_INCOMPATIBLE_CORE_VERSION StatusCode = 8
	STATUS_UNKNOWN_ERROR             StatusCode = 99
)

type Method string

const (
	METHOD_HELLO       Method = "hello"
	METHOD_PUSH        Method = "push"
	METHOD_PULL        Method = "pull"
	METHOD_FETCH       Method = "fetch"
	METHOD_INFO        Method = "info"
	METHOD_REPLICATION Method = "replication"
	METHOD_STATS       Method = "stats"
	METHOD_BYE         Method = "bye"
)

type ResponseStatus struct {
	Desc string     `json:"desc"`
	Code StatusCode `json:"code"`
}

// ////////////////////////////////////////////////////////////////////////////////// //

type DefaultRequest struct {
	CID string `json:"cid"`
}

type DefaultResponse struct {
	Status ResponseStatus `json:"status"`
}

type HelloRequest struct {
	Version  string `json:"version"`
	Hostname string `json:"hostname"`
	Role     string `json:"role"`
}

type HelloResponse struct {
	Status        ResponseStatus      `json:"status"`
	Version       string              `json:"version"`
	CID           string              `json:"cid"`
	Auth          *CORE.SuperuserAuth `json:"auth"`
	SentinelWorks bool                `json:"sentinel_works"`
}

type InfoRequest struct {
	CID  string `json:"cid"`
	ID   int    `json:"id"`
	UUID string `json:"UUID"`
}

type InfoResponse struct {
	Status ResponseStatus     `json:"status"`
	Info   *CORE.InstanceInfo `json:"info"`
}

type FetchResponse struct {
	Instances []*CORE.InstanceInfo `json:"instances"`
	Status    ResponseStatus       `json:"status"`
}

type PullResponse struct {
	Commands []*CommandQueueItem `json:"commands"`
	Status   ResponseStatus      `json:"status"`
}

type ReplicationResponse struct {
	Status ResponseStatus   `json:"status"`
	Info   *ReplicationInfo `json:"info"`
}

type PushRequest struct {
	Command   MasterCommand `json:"command"`
	ID        int           `json:"id"`
	UUID      string        `json:"uuid"`
	Initiator string        `json:"initiator"`
}

type StatsResponse struct {
	Status ResponseStatus `json:"status"`
	Stats  *StatsInfo     `json:"stats"`
}

type ByeRequest struct {
	CID string `json:"cid"`
}

// ////////////////////////////////////////////////////////////////////////////////// //

// GetAuthHeader return API authentication header
func GetAuthHeader(token string) req.Headers {
	return req.Headers{
		"Authorization": "Bearer " + token,
	}
}

// ////////////////////////////////////////////////////////////////////////////////// //

// String returns string representation of client status
func (s ClientState) String() string {
	switch s {
	case STATE_ONLINE:
		return "online"
	case STATE_POSSIBLE_DOWN:
		return "possible-down"
	case STATE_DOWN:
		return "down"
	case STATE_SYNCING:
		return "syncing"
	case STATE_DEAD:
		return "dead"
	}

	return "unknown"
}

// String returns string representation of method
func (m Method) String() string {
	return string(m)
}

// Pattern returns mux pattern for method
func (m Method) Pattern() string {
	return "/" + string(m)
}
