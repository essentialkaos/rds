package client

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"

	"github.com/essentialkaos/ek/v13/req"

	API "github.com/essentialkaos/rds/api"
	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// PropagateCommand sends command to RDS master
func PropagateCommand(command API.MasterCommand, id int, uuid string) error {
	var err error

	if !CORE.IsSyncDaemonActive() {
		return nil
	}

	resp, err := req.Request{
		URL:         getURL(API.METHOD_PUSH),
		Headers:     API.GetAuthHeader(CORE.Config.GetS(CORE.REPLICATION_AUTH_TOKEN)),
		ContentType: req.CONTENT_TYPE_JSON,
		AutoDiscard: true,
		Body: &API.PushRequest{
			Command:   command,
			ID:        id,
			UUID:      uuid,
			Initiator: CORE.User.RealName,
		},
	}.Post()

	if err != nil {
		return fmt.Errorf("Error while sending command to RDS master: %v", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Master returned HTTP status code %d", resp.StatusCode)
	}

	defResponse := &API.DefaultResponse{}
	err = resp.JSON(defResponse)

	if err != nil {
		return fmt.Errorf("Error while decoding RDS master response: %v", err)
	}

	if defResponse.Status.Code != 0 {
		return fmt.Errorf("Sync master error: %s", defResponse.Status.Desc)
	}

	return nil
}

// GetReplicationInfo returns list of RDS clients
func GetReplicationInfo() (*API.ReplicationInfo, error) {
	var err error

	resp, err := req.Request{
		Headers:     API.GetAuthHeader(CORE.Config.GetS(CORE.REPLICATION_AUTH_TOKEN)),
		URL:         getURL(API.METHOD_REPLICATION),
		AutoDiscard: true,
	}.Get()

	if err != nil {
		return nil, fmt.Errorf("Error while sending command to RDS master: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Master returned HTTP status code %d", resp.StatusCode)
	}

	replicationResponse := &API.ReplicationResponse{}
	err = resp.JSON(replicationResponse)

	if err != nil {
		return nil, fmt.Errorf("Error while decoding RDS master response: %v", err)
	}

	if replicationResponse.Status.Code != 0 {
		return nil, fmt.Errorf("Master return error in response: %s", replicationResponse.Status.Desc)
	}

	return replicationResponse.Info, nil
}

// GetReplicationInfo returns stats info
func GetStatsInfo() (*API.StatsInfo, error) {
	var err error

	resp, err := req.Request{
		Headers:     API.GetAuthHeader(CORE.Config.GetS(CORE.REPLICATION_AUTH_TOKEN)),
		URL:         getURL(API.METHOD_STATS),
		AutoDiscard: true,
	}.Get()

	if err != nil {
		return nil, fmt.Errorf("Error while sending command to RDS master: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Master returned HTTP status code %d", resp.StatusCode)
	}

	statsResponse := &API.StatsResponse{}
	err = resp.JSON(statsResponse)

	if err != nil {
		return nil, fmt.Errorf("Error while decoding RDS master response: %v", err)
	}

	if statsResponse.Status.Code != 0 {
		return nil, fmt.Errorf("Master return error in response: %s", statsResponse.Status.Desc)
	}

	return statsResponse.Stats, nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// getURL returns URL of master sync service
func getURL(method API.Method) string {
	host := CORE.Config.GetS(CORE.REPLICATION_MASTER_IP, "127.0.0.1")
	port := CORE.Config.GetS(CORE.REPLICATION_MASTER_PORT)

	return "http://" + host + ":" + port + "/" + string(method)
}
