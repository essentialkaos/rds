package sentinel

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"time"

	"github.com/essentialkaos/ek/v13/log"
	"github.com/essentialkaos/ek/v13/pluralize"
	"github.com/essentialkaos/ek/v13/req"

	API "github.com/essentialkaos/rds/api"
	CORE "github.com/essentialkaos/rds/core"
	AUXI "github.com/essentialkaos/rds/sync/auxi"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Exit codes
const (
	EC_OK    = 0
	EC_ERROR = 1
)

// ////////////////////////////////////////////////////////////////////////////////// //

// cid is client ID
var cid string

// errorFlags is flags for error messages deduplication
var errorFlags = map[API.Method]bool{
	API.METHOD_HELLO: false,
	API.METHOD_PULL:  false,
	API.METHOD_FETCH: false,
	API.METHOD_INFO:  false,
}

// daemonVersion is current daemon version
var daemonVersion string

// sentinelWorks is true if Sentinel is works
var sentinelWorks bool

// ////////////////////////////////////////////////////////////////////////////////// //

// Start starts sync daemon in Sentinel mode
func Start(app, ver, rev string) int {
	daemonVersion = ver

	if rev == "" {
		log.Aux("%s %s started in SENTINEL mode", app, ver)
	} else {
		log.Aux("%s %s (git:%s) started in SENTINEL mode", app, ver, rev)
	}

	if !sendHelloCommand() {
		return EC_ERROR
	}

	// Fetch info about all instances only if Sentinel works
	if sentinelWorks {
		sendFetchCommand()
	}

	runSyncLoop()

	return EC_OK
}

// Stop stops sync daemon
func Stop() {
	if sentinelWorks {
		syncSentinelState(false)
	}

	sendByeCommand()
}

// ////////////////////////////////////////////////////////////////////////////////// //

// runSyncLoop starts sync loop
func runSyncLoop() {
	for range time.NewTicker(time.Second).C {
		sendPullCommand()
	}
}

// sendHelloCommand sends hello command to master
func sendHelloCommand() bool {
	log.Info("Sending hello to master on %s…", CORE.Config.GetS(CORE.REPLICATION_MASTER_IP))

	hostname, _ := os.Hostname()

	helloRequest := &API.HelloRequest{
		Version:  daemonVersion + "/" + CORE.VERSION,
		Hostname: hostname,
		Role:     CORE.ROLE_SENTINEL,
	}

	helloResponse := &API.HelloResponse{}
	err := sendRequest(API.METHOD_HELLO, helloRequest, helloResponse)

	if err != nil {
		if !errorFlags[API.METHOD_HELLO] {
			errorFlags[API.METHOD_HELLO] = true
			log.Error(err.Error())
		}

		return false
	}

	errorFlags[API.METHOD_HELLO] = false

	if helloResponse.Status.Code != API.STATUS_OK {
		log.Crit("Master hello response contains error: %s", helloResponse.Status.Desc)
		return false
	}

	switch AUXI.GetCoreCompatibility(helloResponse.Version) {
	case API.CORE_COMPAT_PARTIAL:
		log.Warn("This client might be incompatible with master node")
	case API.CORE_COMPAT_ERROR:
		log.Crit("This client is not compatible with master node")
		return false
	}

	cid = helloResponse.CID

	log.Info("Master (%s) return CID %s for this client", helloResponse.Version, cid)

	sentinelWorks = helloResponse.SentinelWorks

	// Start or stop Sentinel monitoring
	syncSentinelState(sentinelWorks)

	return true
}

// sendFetchCommand sends fetch command to master
func sendFetchCommand() {
	log.Info("Fetching info about all instances on master…")

	fetchRequest := &API.DefaultRequest{CID: cid}
	fetchResponse := &API.FetchResponse{}

	err := sendRequest(API.METHOD_FETCH, fetchRequest, fetchResponse)

	if err != nil {
		if !errorFlags[API.METHOD_FETCH] {
			errorFlags[API.METHOD_FETCH] = true
			log.Error(err.Error())
		}

		return
	}

	errorFlags[API.METHOD_FETCH] = false

	if fetchResponse.Status.Code != API.STATUS_OK {
		log.Error("Master response contains error: %s", fetchResponse.Status.Desc)
		return
	}

	if len(fetchResponse.Instances) == 0 {
		log.Info("No instances are created on master")
	} else {
		log.Info(
			pluralize.P(
				"Master return info about %d %s",
				len(fetchResponse.Instances), "instance", "instances",
			),
		)
	}

	processFetchedData(fetchResponse.Instances)

	log.Info("Fetched info processing successfully completed")
}

// sendPullCommand sends pull command to master
func sendPullCommand() {
	log.Debug("Pulling commands on master…")

	pullRequest := &API.DefaultRequest{CID: cid}
	pullResponse := &API.PullResponse{}

	err := sendRequest(API.METHOD_PULL, pullRequest, pullResponse)

	if err != nil {
		if !errorFlags[API.METHOD_PULL] {
			errorFlags[API.METHOD_PULL] = true
			log.Error(err.Error())
		}

		return
	}

	errorFlags[API.METHOD_PULL] = false

	if pullResponse.Status.Code != API.STATUS_OK {
		log.Error("Master response for pull command contains error: %s", pullResponse.Status.Desc)

		if pullResponse.Status.Code == API.STATUS_UNKNOWN_CLIENT {
			if sendHelloCommand() {
				sendFetchCommand()
			}
		}

		return
	}

	if len(pullResponse.Commands) == 0 {
		return
	}

	log.Info(
		pluralize.P(
			"Master return %d %s from queue",
			len(pullResponse.Commands), "command", "commands",
		),
	)

	processCommands(pullResponse.Commands)
}

// sendInfoCommand sends info command to master
func sendInfoCommand(id int, uuid string) (*CORE.InstanceInfo, bool) {
	log.Debug("Fetching info for instance with ID %d (%s)", id, uuid)

	infoRequest := &API.InfoRequest{CID: cid, ID: id, UUID: uuid}
	infoResponse := &API.InfoResponse{}

	err := sendRequest(API.METHOD_INFO, infoRequest, infoResponse)

	if err != nil {
		if !errorFlags[API.METHOD_INFO] {
			errorFlags[API.METHOD_INFO] = true
			log.Error(err.Error())
		}

		return nil, false
	}

	errorFlags[API.METHOD_INFO] = false

	if infoResponse.Status.Code != API.STATUS_OK {
		log.Error("Master response for info command contains error: %s", infoResponse.Status.Desc)

		if infoResponse.Status.Code == API.STATUS_UNKNOWN_CLIENT {
			if sendHelloCommand() {
				sendFetchCommand()
			}
		}

		return nil, false
	}

	return infoResponse.Info, true
}

// sendByeCommand sends bye command to the master node
func sendByeCommand() {
	byeRequest := &API.ByeRequest{CID: cid}
	byeResponse := &API.DefaultResponse{}

	err := sendRequest(API.METHOD_BYE, byeRequest, byeResponse)

	if err != nil {
		log.Error(err.Error())
		return
	}

	if byeResponse.Status.Code != API.STATUS_OK {
		log.Error(
			"Master response for bye command contains error: %s",
			byeResponse.Status.Desc,
		)
	}

	log.Info("This client successfully unregistered on the master")
}

// ////////////////////////////////////////////////////////////////////////////////// //

// processCommands processes command queue item and route to handler
func processCommands(items []*API.CommandQueueItem) {
	items = removeConflictActions(items)

	for _, item := range items {
		log.Debug("Processing command \"%v\"", item.Command)

		switch item.Command {
		case API.COMMAND_CREATE:
			createCommandHandler(item)
		case API.COMMAND_DESTROY:
			destroyCommandHandler(item)
		case API.COMMAND_EDIT:
			// ignore
		case API.COMMAND_START:
			startCommandHandler(item)
		case API.COMMAND_STOP:
			stopCommandHandler(item)
		case API.COMMAND_RESTART:
			// ignore
		case API.COMMAND_START_ALL:
			startAllCommandHandler(item)
		case API.COMMAND_STOP_ALL:
			stopAllCommandHandler(item)
		case API.COMMAND_RESTART_ALL:
			// ignore
		case API.COMMAND_SENTINEL_START:
			sentinelStartCommandHandler(item)
		case API.COMMAND_SENTINEL_STOP:
			sentinelStopCommandHandler(item)
		default:
			log.Error("Received unknown command %s", item.Command)
		}
	}
}

// createCommandHandler is handler for "create" command
func createCommandHandler(item *API.CommandQueueItem) {
	log.Info("(%3d|%s) Instance creation command", item.InstanceID, item.Initiator)

	if !sentinelWorks {
		log.Warn("Command %s ignored - Sentinel not working", item.Command)
		return
	}

	if CORE.IsInstanceExist(item.InstanceID) {
		log.Error("(%3d) Can't execute command %s - instance already exist", item.InstanceID, item.Command)
		return
	}

	info, ok := sendInfoCommand(item.InstanceID, item.InstanceUUID)

	if !ok {
		return
	}

	createInstance(info.Meta, info.State)
}

// destroyCommandHandler is handler for "destroy" command
func destroyCommandHandler(item *API.CommandQueueItem) {
	log.Info("(%3d|%s) Instance destroying command", item.InstanceID, item.Initiator)

	if !sentinelWorks {
		log.Warn("Command %s ignored - Sentinel not working", item.Command)
		return
	}

	if !isValidCommandItem(item) {
		return
	}

	destroyInstance(item.InstanceID)
}

// startCommandHandler is handler for "start" command
func startCommandHandler(item *API.CommandQueueItem) {
	log.Info("(%3d|%s) Instance starting command", item.InstanceID, item.Initiator)

	if !sentinelWorks {
		log.Warn("Command %s ignored - Sentinel not working", item.Command)
		return
	}

	if !isValidCommandItem(item) {
		return
	}

	enableMonitoring(item.InstanceID)
}

// stopCommandHandler is handler for "stop" command
func stopCommandHandler(item *API.CommandQueueItem) {
	log.Info("(%3d|%s) Instance stopping command", item.InstanceID, item.Initiator)

	if !sentinelWorks {
		log.Warn("Command %s ignored - Sentinel not working", item.Command)
		return
	}

	if !isValidCommandItem(item) {
		return
	}

	disableMonitoring(item.InstanceID)
}

// startAllCommandHandler is handler for "start-all" command
func startAllCommandHandler(item *API.CommandQueueItem) {
	log.Info("(%3d|%s) Starting all instances command", item.InstanceID, item.Initiator)

	if !CORE.HasInstances() {
		log.Warn("Command %s ignored - no instances are created", item.Command)
		return
	}

	for _, id := range CORE.GetInstanceIDList() {
		if CORE.IsSentinelMonitors(id) {
			continue
		}

		err := CORE.SentinelStartMonitoring(id)

		if err != nil {
			log.Error("(%3d) Can't start Sentinel monitoring: %v", id, err)
		}
	}

	log.Info("Sentinel monitoring enabled for all instances")
}

// stopAllCommandHandler is handler for "stop-all" command
func stopAllCommandHandler(item *API.CommandQueueItem) {
	if !CORE.HasInstances() {
		log.Warn("Command %s ignored - no instances are created", item.Command)
		return
	}

	log.Info("(%3d|%s) Stopping all instances command", item.InstanceID, item.Initiator)

	for _, id := range CORE.GetInstanceIDList() {
		if !CORE.IsSentinelMonitors(id) {
			continue
		}

		err := CORE.SentinelStopMonitoring(id)

		if err != nil {
			log.Error("(%3d) Can't stop Sentinel monitoring: %v", id, err)
		}
	}

	log.Info("Sentinel monitoring disabled for all instances")
}

// sentinelStartCommandHandler is handler for "sentinel-start" command
func sentinelStartCommandHandler(item *API.CommandQueueItem) {
	log.Info("(%3d|%s) Sentinel starting command", item.InstanceID, item.Initiator)

	if CORE.IsSentinelActive() {
		log.Warn("Command %s ignored - Sentinel already works", item.Command)
		return
	}

	errs := CORE.SentinelStart()

	if len(errs) != 0 {
		for _, err := range errs {
			log.Error("Error while starting Sentinel: %v", err)
		}

		return
	}

	sentinelWorks = true

	log.Info("Sentinel started")

	// Fetch info about all instances and enable monitoring if required
	sendFetchCommand()
}

// sentinelStopCommandHandler is handler for "sentinel-stop" command
func sentinelStopCommandHandler(item *API.CommandQueueItem) {
	log.Info("(%3d|%s) Sentinel stopping command", item.InstanceID, item.Initiator)

	if !CORE.IsSentinelActive() {
		log.Warn("Command %s ignored - Sentinel already stopped", item.Command)
		return
	}

	err := CORE.SentinelStop()

	if err != nil {
		log.Error("Error while stopping Sentinel: %v", err)
		return
	}

	sentinelWorks = false

	log.Info("Sentinel stopped")
}

// ////////////////////////////////////////////////////////////////////////////////// //

// createInstance creates new instance
func createInstance(meta *CORE.InstanceMeta, state CORE.State) {
	// We don't need these passwords on sentinel node, so we wipe them
	meta.Preferencies.AdminPassword = ""
	meta.Preferencies.SyncPassword = ""
	meta.Preferencies.ServicePassword = ""

	err := CORE.CreateInstance(meta)

	if err != nil {
		log.Error("(%3d) Error while instance creation: %v", meta.ID, err)
	} else {
		log.Info("(%3d) Added info about created instance", meta.ID)
	}

	if meta.Preferencies.ReplicationType.IsReplica() && state.IsWorks() {
		enableMonitoring(meta.ID)
	}
}

// destroyInstance destroys instance
func destroyInstance(id int) {
	err := CORE.DestroyInstance(id)

	if err != nil {
		log.Error("(%3d) Error while instance destroying: %v", id, err)
	} else {
		log.Info("(%3d) Sentinel monitoring disabled for destroyed instance", id)
	}
}

// enableMonitoring enables Sentinel monitoring for instance with given ID
func enableMonitoring(id int) {
	if CORE.IsSentinelMonitors(id) {
		return
	}

	err := CORE.SentinelStartMonitoring(id)

	if err != nil {
		log.Error("(%3d) Can't start Sentinel monitoring: %v", id, err)
	} else {
		log.Info("(%3d) Sentinel monitoring enabled for instance", id)
	}
}

// disableMonitoring disables Sentinel monitoring for instance with given ID
func disableMonitoring(id int) {
	if !CORE.IsSentinelMonitors(id) {
		return
	}

	err := CORE.SentinelStopMonitoring(id)

	if err != nil {
		log.Error("(%3d) Can't stop Sentinel monitoring: %v", id, err)
		return
	}

	log.Info("(%3d) Sentinel monitoring disabled for instance", id)
}

// isValidCommandItem validates command item from queue
func isValidCommandItem(item *API.CommandQueueItem) bool {
	if !CORE.IsInstanceExist(item.InstanceID) {
		log.Warn(
			"(%3d) Can't execute command %s - instance does not exist",
			item.InstanceID, item.Command,
		)
		return false
	}

	meta, err := CORE.GetInstanceMeta(item.InstanceID)

	if err != nil {
		log.Error(
			"(%3d) Can't execute command %s - can't read instance meta: %v",
			item.InstanceID, item.Command, err,
		)
		return false
	}

	if item.InstanceUUID != meta.UUID {
		log.Error(
			"(%3d) Command %s ignored - gotten instance UUID is differ from current instance UUID",
			item.InstanceID, item.Command,
		)
		return false
	}

	return true
}

// processFetchedData processes fetched data
func processFetchedData(instances []*CORE.InstanceInfo) {
	err := CORE.SentinelReset()

	if err != nil {
		log.Error("Can't reset Sentinel state: %v", err)
	}

	idList := CORE.GetInstanceIDList()

	if len(idList) != 0 {
		for _, id := range idList {
			if !isInstanceSliceContainsInstance(instances, id) {
				err = CORE.DestroyInstance(id)

				if err != nil {
					log.Error("(%3d) Can't remove info about destroyed instance: %v", id, err)
				}
			}
		}
	}

	if len(instances) != 0 {
		for _, info := range instances {
			processInstanceData(info)
		}
	}
}

// processInstanceData processes info from master node about instance
func processInstanceData(info *CORE.InstanceInfo) {
	id := info.Meta.ID

	if CORE.IsInstanceExist(id) {
		meta, err := CORE.GetInstanceMeta(id)

		if err != nil {
			log.Error("(%3d) Can't read local instance meta. Skipping instance…", id)
			return
		}

		if info.Meta.UUID != meta.UUID {
			log.Warn("(%3d) Instance exists on master, but have different UUID. Instance will be destroyed.", id)
			destroyInstance(id)
			return
		}

		if info.State.IsWorks() {
			enableMonitoring(id)
			return
		}
	}

	createInstance(info.Meta, info.State)
}

// syncSentinelState syncs state of Sentinel with master
func syncSentinelState(works bool) {
	var err error

	if works {
		if !CORE.IsSentinelActive() {
			errs := CORE.SentinelStart()

			if len(errs) != 0 {
				log.Error("Can't start Sentinel daemon: %v", err)

				// Print all errors into log
				for _, err := range errs {
					log.Error("Error while starting Sentinel: %v", err)
				}
			} else {
				log.Info("Sentinel daemon started")
			}
		}
	} else {
		if CORE.IsSentinelActive() {
			err = CORE.SentinelStop()

			if err != nil {
				log.Error("Error while stopping Sentinel: %v", err)
			} else {
				log.Info("Sentinel daemon stopped")
			}
		}
	}
}

// removeConflictActions filters create+destroy commands for same instance
func removeConflictActions(items []*API.CommandQueueItem) []*API.CommandQueueItem {
	if len(items) == 0 {
		return items
	}

	var item *API.CommandQueueItem
	var result []*API.CommandQueueItem

	initList := make(map[string]uint8)

	for _, item = range items {
		if item.Command == API.COMMAND_CREATE {
			initList[item.InstanceUUID] = 1
		} else if item.Command == API.COMMAND_DESTROY {
			if initList[item.InstanceUUID] == 1 {
				initList[item.InstanceUUID] = 2
				log.Warn("(%3d) Instance was created but later was destroyed. All actions for instance will be skipped.", item.InstanceID)
			}
		}
	}

	for _, item = range items {
		if initList[item.InstanceUUID] == 2 {
			continue
		}

		result = append(result, item)
	}

	return result
}

// isInstanceSliceContainsInstance returns true if instance slice contains instance
// with given ID
func isInstanceSliceContainsInstance(instances []*CORE.InstanceInfo, id int) bool {
	for _, info := range instances {
		if info.Meta.ID == id {
			return true
		}
	}

	return false
}

// ////////////////////////////////////////////////////////////////////////////////// //

// getURL returns method URL
func getURL(method API.Method) string {
	host := CORE.Config.GetS(CORE.REPLICATION_MASTER_IP)
	port := CORE.Config.GetS(CORE.REPLICATION_MASTER_PORT)

	return "http://" + host + ":" + port + "/" + string(method)
}

// sendRequest sends request to the master node
func sendRequest(method API.Method, reqData, respData any) error {
	resp, err := req.Request{
		URL:         getURL(method),
		Headers:     API.GetAuthHeader(CORE.Config.GetS(CORE.REPLICATION_AUTH_TOKEN)),
		ContentType: req.CONTENT_TYPE_JSON,
		Body:        reqData,
		AutoDiscard: true,
	}.Post()

	if err != nil {
		return fmt.Errorf("Can't send %s request to master: %v", string(method), err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Master return HTTP status code %d", resp.StatusCode)
	}

	err = resp.JSON(&respData)

	if err != nil {
		return fmt.Errorf("Can't decode response: %v", err)
	}

	return nil
}
