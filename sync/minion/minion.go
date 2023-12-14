package minion

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/knf"
	"github.com/essentialkaos/ek/v12/log"
	"github.com/essentialkaos/ek/v12/mathutil"
	"github.com/essentialkaos/ek/v12/pluralize"
	"github.com/essentialkaos/ek/v12/req"
	"github.com/essentialkaos/ek/v12/timeutil"
	"github.com/essentialkaos/ek/v12/version"

	API "github.com/essentialkaos/rds/api"
	CORE "github.com/essentialkaos/rds/core"
	REDIS "github.com/essentialkaos/rds/redis"
	AUXI "github.com/essentialkaos/rds/sync/auxi"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Exit codes
const (
	EC_OK    = 0
	EC_ERROR = 1
)

// ////////////////////////////////////////////////////////////////////////////////// //

// InstanceSyncState contains info about current instance state
type InstanceSyncState struct {
	SyncLeftBytes  int
	IsLoading      bool
	IsSyncing      bool
	IsWaiting      bool
	IsConnected    bool
	IsDisklessSync bool
}

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

// connectedToMaster is true if minion currently connected to the master node
var connectedToMaster bool

// ////////////////////////////////////////////////////////////////////////////////// //

// Start starts sync daemon in minion mode
func Start(app, ver, rev string) int {
	daemonVersion = ver

	if rev == "" {
		log.Aux("%s %s started in MINION mode", app, ver)
	} else {
		log.Aux("%s %s (git:%s) started in MINION mode", app, ver, rev)
	}

	if !sendHelloCommand() {
		return EC_ERROR
	}

	sendFetchCommand()
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
	connectedToMaster = false

	log.Info("Sending hello to master on %s…", CORE.Config.GetS(CORE.REPLICATION_MASTER_IP))

	hostname, _ := os.Hostname()

	helloRequest := &API.HelloRequest{
		Version:  daemonVersion + "/" + CORE.VERSION,
		Hostname: hostname,
		Role:     CORE.ROLE_MINION,
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

	connectedToMaster = true
	cid = helloResponse.CID

	log.Info("Master (%s) return CID %s for this client", helloResponse.Version, cid)

	sentinelWorks = helloResponse.SentinelWorks

	// Start or stop Sentinel monitoring
	syncSentinelState(sentinelWorks)

	if helloResponse.Auth == nil {
		log.Warn("Looks like master is not initialized (superuser data not generated) - hello response contains empty superuser auth data")
		return true
	}

	return CORE.SaveSUAuth(helloResponse.Auth, true) == nil
}

// sendFetchCommand sends fetch command to the master node
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

// sendPullCommand sends pull command to the master node
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

// sendInfoCommand sends info command to the master node
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

// processCommands processes command queue items and routes them to handlers
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
			editCommandHandler(item)
		case API.COMMAND_START:
			startCommandHandler(item)
		case API.COMMAND_STOP:
			stopCommandHandler(item)
		case API.COMMAND_RESTART:
			restartCommandHandler(item)
		case API.COMMAND_START_ALL:
			startAllCommandHandler(item)
		case API.COMMAND_STOP_ALL:
			stopAllCommandHandler(item)
		case API.COMMAND_RESTART_ALL:
			restartAllCommandHandler(item)
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
	if CORE.IsInstanceExist(item.InstanceID) {
		log.Error("(%3d) Can't execute command %s - instance already exist", item.InstanceID, item.Command)
		return
	}

	log.Info("(%3d) Creating instance…", item.InstanceID)

	info, ok := sendInfoCommand(item.InstanceID, item.InstanceUUID)

	if !ok {
		return
	}

	createInstance(info.Meta, info.State)
}

// destroyCommandHandler is handler for "destroy" command
func destroyCommandHandler(item *API.CommandQueueItem) {
	if !isValidCommandItem(item) {
		return
	}

	log.Info("(%3d) Destroying instance…", item.InstanceID)

	destroyInstance(item.InstanceID)
}

// editCommandHandler is handler for "edit" command
func editCommandHandler(item *API.CommandQueueItem) {
	if !isValidCommandItem(item) {
		return
	}

	log.Info("(%3d) Updating instance meta…", item.InstanceID)

	info, ok := sendInfoCommand(item.InstanceID, item.InstanceUUID)

	if !ok {
		return
	}

	editInstance(info.Meta)
}

// startCommandHandler is handler for "start" command
func startCommandHandler(item *API.CommandQueueItem) {
	if !isValidCommandItem(item) {
		return
	}

	log.Info("(%3d) Starting instance…", item.InstanceID)

	startInstance(item.InstanceID)
}

// stopCommandHandler is handler for "stop" command
func stopCommandHandler(item *API.CommandQueueItem) {
	if !isValidCommandItem(item) {
		return
	}

	log.Info("(%3d) Stopping instance…", item.InstanceID)

	stopInstance(item.InstanceID)
}

// restartCommandHandler is handler for "restart" command
func restartCommandHandler(item *API.CommandQueueItem) {
	if !isValidCommandItem(item) {
		return
	}

	log.Info("(%3d) Restarting instance…", item.InstanceID)

	restartInstance(item.InstanceID)
}

// startAllCommandHandler is handler for "start-all" command
func startAllCommandHandler(item *API.CommandQueueItem) {
	if !CORE.HasInstances() {
		log.Warn("Command %s ignored - no instances are created", item.Command)
		return
	}

	log.Info("Starting all instances…")

	startAllInstances()
}

// stopAllCommandHandler is handler for "stop-all" command
func stopAllCommandHandler(item *API.CommandQueueItem) {
	if !CORE.HasInstances() {
		log.Warn("Command %s ignored - no instances are created", item.Command)
		return
	}

	log.Info("Stopping all instances…")

	stopAllInstances()
}

// restartAllCommandHandler is handler for "restart-all" command
func restartAllCommandHandler(item *API.CommandQueueItem) {
	if !CORE.HasInstances() {
		log.Warn("Command %s ignored - no instances are created", item.Command)
		return
	}

	log.Info("Restarting all instances…")

	restartAllInstances()
}

// sentinelStartCommandHandler is handler for "sentinel-start" command
func sentinelStartCommandHandler(item *API.CommandQueueItem) {
	if CORE.IsSentinelActive() {
		log.Warn("Command %s ignored - Sentinel already works", item.Command)
		return
	}

	log.Info("Starting sentinel…")

	startSentinel()
}

// sentinelStopCommandHandler is handler for "sentinel-stop" command
func sentinelStopCommandHandler(item *API.CommandQueueItem) {
	if !CORE.IsSentinelActive() {
		log.Warn("Command %s ignored - Sentinel already stopped", item.Command)
		return
	}

	log.Info("Stopping sentinel…")

	stopSentinel()
}

// processFetchedData processes fetched data
func processFetchedData(instances []*CORE.InstanceInfo) {
	idList := CORE.GetInstanceIDList()

	if len(idList) != 0 {
		for _, id := range idList {
			if !isInstanceSliceContainsInstance(instances, id) {
				log.Info("(%3d) Instance doesn't exist on master. Instance will be destroyed.", id)
				destroyInstance(id)
			}
		}
	}

	if len(instances) != 0 {
		for _, info := range instances {
			processInstanceData(info)
		}
	}
}

// processInstanceData processes info about instance
func processInstanceData(info *CORE.InstanceInfo) {
	id := info.Meta.ID

	// If there is no instance, create it
	if !CORE.IsInstanceExist(id) {
		createInstance(info.Meta, info.State)
		return
	}

	meta, err := CORE.GetInstanceMeta(id)

	if err != nil {
		log.Error("(%3d) Can't read local instance meta. Skipping instance…", id)
		return
	}

	// If instance exist, but there is no data, recreate instance
	if !CORE.HasInstanceData(id) || info.Meta.UUID != meta.UUID {
		switch {
		case info.Meta.UUID != meta.UUID:
			log.Warn("(%3d) Instance exists on master, but have different UUID. Instance will be recreated.", id)
		default:
			log.Warn("(%3d) Instance data not present on disk (possible sentinel → minion migration). Instance will be recreated.", id)
		}

		if !destroyInstance(id) {
			return
		}

		createInstance(info.Meta, info.State)
		return
	}

	// If meta is not equal, tychange it
	if !isMetaEqual(info.Meta, meta) {
		editInstance(info.Meta)
	}
	state, err := CORE.GetInstanceState(id, false)

	if err != nil {
		log.Error("(%3d) Can't check instance state: %v", id, err)
		return
	}

	// Sync instance state
	if info.State.IsWorks() && !state.IsWorks() {
		startInstance(id)
		checkRedisVersionCompatibility(info.Meta)
		return
	} else if info.State.IsStopped() && !state.IsStopped() {
		stopInstance(id)
		return
	}

	// Check instance for problems
	checkReplicaMode(id)
	checkRedisVersionCompatibility(info.Meta)

	log.Info("(%3d) Instance is up-to-date with master", id)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// createInstance creates instance
func createInstance(meta *CORE.InstanceMeta, state CORE.State) bool {
	id := meta.ID
	err := CORE.CreateInstance(meta)

	if err != nil {
		log.Error("(%3d) Error while instance creation: %v", id, err)
		return false
	}

	log.Info("(%3d) Instance successfully created", id)

	checkReplicaMode(id)
	checkRedisVersionCompatibility(meta)

	if state.IsWorks() {
		err = CORE.StartInstance(id, false)

		if err != nil {
			log.Error("(%3d) Starting instance failed: %v", id, err)
		} else {
			log.Info("(%3d) Instance started", id)
			syncBlocker(id)
		}
	}

	return true
}

// destroyInstance destroys instance
func destroyInstance(id int) bool {
	err := CORE.DestroyInstance(id)

	if err != nil {
		log.Error("(%3d) Instance destroying failed: %v", id, err)
		return false
	}

	log.Info("(%3d) Instance destroyed", id)

	return true
}

// editInstance modify instance
func editInstance(meta *CORE.InstanceMeta) bool {
	id := meta.ID
	oldMeta, err := CORE.GetInstanceMeta(id)

	if err != nil {
		log.Error("(%3d) Can't read instance meta: %v", id, err)
	}

	err = CORE.UpdateInstance(meta)

	if err != nil {
		log.Error("(%3d) Error while metadata update: %v", id, err)
		return false
	}

	log.Info("(%3d) Metadata updated", id)

	if oldMeta.Preferencies.ReplicationType != meta.Preferencies.ReplicationType {
		err = changeInstanceReplicationType(id, meta.Preferencies.ReplicationType)

		if err != nil {
			log.Error("(%3d) Error while changing replication type: %v", id, err)
			return false
		}

		log.Info(
			"(%3d) Replication type changed (%s → %s)",
			id, oldMeta.Preferencies.ReplicationType, meta.Preferencies.ReplicationType,
		)
	}

	return true
}

// startInstance starts instance
func startInstance(id int) bool {
	state, err := CORE.GetInstanceState(id, false)

	if err != nil {
		log.Error("(%3d) Can't get instance state: %v", id, err)
		return false
	}

	if state.IsWorks() {
		log.Warn("(%3d) Can't start instance - instance already works", id)
		return false
	}

	checkReplicaMode(id)

	err = CORE.StartInstance(id, false)

	if err != nil {
		log.Error("(%3d) Instance start failed", id)
		return false
	}

	log.Info("(%3d) Instance started", id)

	syncBlocker(id)

	return true
}

// stopInstance stops instance
func stopInstance(id int) bool {
	state, err := CORE.GetInstanceState(id, false)

	if err != nil {
		log.Error("(%3d) Can't get instance state: %v", id, err)
		return false
	}

	if state.IsStopped() {
		log.Warn("(%3d) Can't stop instance - instance already stopped", id)
		return false
	}

	err = CORE.StopInstance(id, true)

	if err != nil {
		log.Error("(%3d) Instance stop failed", id)
		return false
	}

	log.Info("(%3d) Instance stopped", id)

	return true
}

// restartInstance restarts instance
func restartInstance(id int) bool {
	state, err := CORE.GetInstanceState(id, false)

	if err != nil {
		log.Error("(%3d) Can't get instance state: %v", id, err)
		return false
	}

	if state.IsWorks() {
		err = CORE.StopInstance(id, false)

		if err != nil {
			log.Error("(%3d) Instance stopping failed: %v", id, err)
			return false
		}
	}

	err = CORE.StartInstance(id, false)

	if err != nil {
		log.Error("(%3d) Instance restart failed: %v", id, err)
		return false
	}

	log.Info("(%3d) Instance restarted", id)

	syncBlocker(id)

	return true
}

// startAllInstances starts all instances
func startAllInstances() bool {
	idList := CORE.GetInstanceIDList()

	if len(idList) == 0 {
		return false
	}

	for _, id := range idList {
		state, err := CORE.GetInstanceState(id, false)

		if err != nil {
			log.Error("(%3d) Can't get instance state: %v", id, err)
			continue
		}

		if state.IsStopped() {
			err = CORE.StartInstance(id, false)

			if err != nil {
				log.Error("(%3d) Instance start failed: %v", id, err)
			} else {
				syncBlocker(id)
			}
		}
	}

	log.Info("All instances started")

	return true
}

// stopAllInstances stops all instances
func stopAllInstances() bool {
	idList := CORE.GetInstanceIDList()

	if len(idList) == 0 {
		return false
	}

	for _, id := range idList {
		state, err := CORE.GetInstanceState(id, false)

		if err != nil {
			log.Error("(%3d) Can't get instance state: %v", id, err)
			continue
		}

		if state.IsWorks() {
			err = CORE.StopInstance(id, false)
			if err != nil {
				log.Error("(%3d) Instance stop failed: %v", id, err)
			}
		}
	}

	log.Info("All instances stopped")

	return true
}

// restartAllInstances restarts all instances
func restartAllInstances() bool {
	idList := CORE.GetInstanceIDList()

	if len(idList) == 0 {
		return false
	}

	for _, id := range idList {
		state, err := CORE.GetInstanceState(id, false)

		if err != nil {
			log.Error("(%3d) Can't get instance state: %v", id, err)
			continue
		}

		if state.IsWorks() {
			err = CORE.StopInstance(id, false)

			if err != nil {
				log.Error("(%3d) Instance stop failed: %v", id, err)
			}

			err = CORE.StartInstance(id, false)

			if err != nil {
				log.Error("(%3d) Instance start failed: %v", id, err)
			} else {
				syncBlocker(id)
			}
		} else {
			err = CORE.StartInstance(id, false)

			if err != nil {
				log.Error("(%3d) Instance start failed: %v", id, err)
			} else {
				syncBlocker(id)
			}
		}
	}

	log.Info("All instances restarted")

	return true
}

// startSentinel starts Sentinel
func startSentinel() bool {
	errs := CORE.SentinelStart()

	if len(errs) != 0 {
		for _, err := range errs {
			log.Error("Error while starting Sentinel: %v", err)
		}

		return false
	}

	sentinelWorks = true

	log.Info("Sentinel started")

	return true
}

// stopSentinel stops Sentinel
func stopSentinel() bool {
	err := CORE.SentinelStop()

	if err != nil {
		log.Error("Error while stopping Sentinel: %v", err)
		return false
	}

	sentinelWorks = false

	log.Info("Sentinel stopped")

	return true
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
		return fmt.Errorf("Can't send %s request to master", string(method))
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

// syncSentinelState syncs state of Sentinel with master
func syncSentinelState(sentinelWorks bool) {
	var err error

	if sentinelWorks {
		if !CORE.IsSentinelActive() {
			errs := CORE.SentinelStart()

			if len(errs) != 0 {
				log.Error("Can't start Sentinel daemon: %v", err)

				// Print all errors to log
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
				log.Warn("(%3d) The instance was created but later was destroyed. All actions with the instance will be skipped.", item.InstanceID)
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

// syncBlocker used for blocking RDS sync process when Redis replica syncing
// with master instance or loading data from disk
func syncBlocker(id int) {
	config, err := CORE.GetInstanceConfig(id, time.Second)

	if err != nil {
		log.Error("(%3d) Can't read instance config: %v", id, err)
		// We wait 1 min to reduce the load on a minion if there is a lot of instances
		time.Sleep(time.Minute)
		return
	}

	hasReplica := config.Get("replicaof") != "" || config.Get("slaveof") != ""

	// Instance is not a replica (standby), go to next…
	if !hasReplica {
		return
	}

	log.Info("(%3d) Starting sync with master instance…", id)

	time.Sleep(CORE.Config.GetD(CORE.REPLICATION_INIT_SYNC_DELAY, knf.Second, 3*time.Second))

	syncingWaitLoop(id)
}

// syncingWaitLoop blocks main sync process till syncing will be completed
func syncingWaitLoop(id int) {
	start := time.Now().Unix()
	maxWait := CORE.Config.GetD(CORE.REPLICATION_MAX_SYNC_WAIT, knf.Second)
	deadline := time.Now().Add(maxWait)

	log.Info(
		"(%3d) Instance is syncing with master (deadline: %s)…",
		id, timeutil.Format(deadline, "%Y/%m/%d %H:%M:%S"),
	)

	var syncingFlag, loadingFlag, disklessFlag bool
	var syncLeftBytesPrev int

	for now := range time.Tick(time.Second) {
		if now.After(deadline) {
			log.Warn("(%3d) Max wait time is reached (%g sec) but instance is still syncing. Continue anyway…", id, maxWait.Seconds())
			break
		}

		state := getInstanceSyncState(id)

		if !disklessFlag && state.IsDisklessSync {
			log.Info("(%3d) Diskless sync is used. It means that we can't know how much data will be transferred to the replica.", id)
			disklessFlag = true
		}

		if state.IsLoading && !loadingFlag {
			log.Info("(%3d) Instance is loading data in memory…", id)
			loadingFlag = true
		}

		if !state.IsConnected && state.IsWaiting && syncLeftBytesPrev > 0 {
			log.Error("(%3d) It looks like instance can't load received data (possible version mismatch). Continue node syncing…", id)
			break
		}

		if state.IsSyncing && !state.IsLoading {
			loadingFlag = false

			syncingTime := now.Unix() - start

			if !syncingFlag || syncingTime%15 == 0 {
				if disklessFlag {
					syncSpeed := float64(mathutil.Abs(state.SyncLeftBytes)) - float64(syncLeftBytesPrev)
					log.Info(
						"(%3d) Receiving data from master (%s/s), %s was received…", id,
						fmtutil.PrettySize(math.Max(0, syncSpeed)), fmtutil.PrettySize(mathutil.Abs(state.SyncLeftBytes)),
					)
				} else {
					syncSpeed := float64(syncLeftBytesPrev) - float64(state.SyncLeftBytes)
					log.Info(
						"(%3d) Receiving data from master (%s/s), %s is left…", id,
						fmtutil.PrettySize(math.Max(0, syncSpeed)), fmtutil.PrettySize(state.SyncLeftBytes),
					)
				}
			}

			syncLeftBytesPrev = mathutil.Abs(state.SyncLeftBytes)

			loadingFlag = false
			syncingFlag = true
		}

		if !state.IsSyncing && state.IsConnected {
			log.Info("(%3d) Instance completed sync with master", id)
			break
		}
	}
}

// changeInstanceReplicationType changes replication type for given instance
func changeInstanceReplicationType(id int, replType CORE.ReplicationType) error {
	err := CORE.RegenerateInstanceConfig(id)

	if err != nil {
		return fmt.Errorf("Can't regenerate instance configuration file: %v", err)
	}

	state, err := CORE.GetInstanceState(id, false)

	if err != nil {
		return fmt.Errorf("Can't get instance state: %v", err)
	}

	if !state.IsWorks() {
		return nil
	}

	switch replType {
	case CORE.REPL_TYPE_REPLICA:
		err = changeInstanceToReplica(id)
	case CORE.REPL_TYPE_STANDBY:
		err = changeInstanceToStadby(id)
	}

	return err
}

// chengeInstanceToReplica changes instance replication type to "replica"
func changeInstanceToReplica(id int) error {
	masterHost := CORE.Config.GetS(CORE.REPLICATION_MASTER_IP)
	masterPort := strconv.Itoa(CORE.GetInstancePort(id))

	resp, err := CORE.ExecCommand(id, &REDIS.Request{
		Command: []string{"REPLICAOF", masterHost, masterPort},
		Timeout: time.Minute,
	})

	syncBlocker(id)

	if resp.Err != nil {
		err = resp.Err
	}

	return err
}

// chengeInstanceToStadby changes instance replication type to "standby"
func changeInstanceToStadby(id int) error {
	resp, err := CORE.ExecCommand(id, &REDIS.Request{
		Command: []string{"REPLICAOF", "NO", "ONE"},
		Timeout: time.Minute,
	})

	if resp.Err != nil {
		err = resp.Err
	}

	if err != nil {
		return err
	}

	resp, err = CORE.ExecCommand(id, &REDIS.Request{
		Command: []string{"FLUSHALL", "ASYNC"},
		Timeout: time.Minute,
	})

	if resp.Err != nil {
		err = resp.Err
	}

	return err
}

// getInstanceSyncState returns current instance syncing state
func getInstanceSyncState(id int) InstanceSyncState {
	info, err := CORE.GetInstanceInfo(id, time.Second, false)

	// This is timeout error means that instance is initially
	// loading data into memory
	if err != nil {
		return InstanceSyncState{IsLoading: true}
	}

	syncingInProgress := info.Get("replication", "master_sync_in_progress") == "1"
	leftBytes := info.GetI("replication", "master_sync_left_bytes")

	return InstanceSyncState{
		IsSyncing:      syncingInProgress,
		IsLoading:      info.Get("persistence", "loading") == "1",
		IsConnected:    info.Get("replication", "master_link_status") == "up",
		IsDisklessSync: syncingInProgress && leftBytes < -1,
		IsWaiting:      leftBytes == -1,
		SyncLeftBytes:  leftBytes,
	}
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

// checkRedisVersionCompatibility checks compatibility with master instance
func checkRedisVersionCompatibility(meta *CORE.InstanceMeta) {
	if meta.Compatible == "" {
		return
	}

	masterVersion, err := version.Parse(meta.Compatible)

	if err != nil {
		return
	}

	minionVersion := CORE.GetInstanceVersion(meta.ID)

	if minionVersion.String() == "" {
		return
	}

	if minionVersion.Major() < masterVersion.Major() {
		log.Warn(
			"(%3d) This Redis instance is older (%s) than master Redis instance (%s)",
			meta.ID, minionVersion.String(), masterVersion.String(),
		)
	}
}

// checkReplicaMode checks if replica has enabled read-only mode
func checkReplicaMode(id int) {
	if !CORE.Config.GetB(CORE.REPLICATION_CHECK_READONLY_MODE, true) ||
		!CORE.IsFailoverMethod(CORE.FAILOVER_METHOD_STANDBY) {
		return
	}

	instanceConfig, err := CORE.ReadInstanceConfig(id)

	if err != nil {
		log.Error("(%3d) Can't check instance config for read-only mode: %v", id, err)
		return
	}

	if instanceConfig.Get("slave-read-only") == "yes" ||
		instanceConfig.Get("replica-read-only") == "yes" {
		log.Warn("(%3d) Read-only mode is enabled for this instance. Failover can fail if this node becomes a master.", id)
	}
}

// isMetaEqual compares 2 meta strcuts
func isMetaEqual(m1, m2 *CORE.InstanceMeta) bool {
	switch {
	case m1.Desc != m2.Desc,
		m1.Preferencies.ReplicationType != m2.Preferencies.ReplicationType,
		m1.Auth.User != m2.Auth.User,
		m1.Auth.Pepper != m2.Auth.Pepper,
		m1.Auth.Hash != m2.Auth.Hash,
		isMapsEqual(m1.Storage, m2.Storage) == false,
		strings.Join(m1.Tags, " ") != strings.Join(m2.Tags, " "):
		return false
	}

	return true
}

// isMapsEqual compares 2 maps
func isMapsEqual(m1, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}

	for k, v := range m1 {
		if m2[k] != v {
			return false
		}
	}

	return true
}
