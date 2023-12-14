package sync

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/httputil"
	"github.com/essentialkaos/ek/v12/knf"
	"github.com/essentialkaos/ek/v12/log"
	"github.com/essentialkaos/ek/v12/mathutil"
	"github.com/essentialkaos/ek/v12/netutil"
	"github.com/essentialkaos/ek/v12/sortutil"
	"github.com/essentialkaos/ek/v12/timeutil"

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

type ClientInfo struct {
	CID            string
	Role           string
	Version        string
	Hostname       string
	IP             string
	LastSeen       int64
	LastSync       int64
	ConnectionDate int64
	State          API.ClientState
	Syncing        bool
}

const (
	DELAY_POSSIBLE_DOWN int64 = 15      // 15 sec
	DELAY_DOWN                = 60      // 1 min
	DELAY_DEAD                = 15 * 60 // 15 min
)

// ////////////////////////////////////////////////////////////////////////////////// //

type ClientsList []*API.ClientInfo

func (s ClientsList) Len() int      { return len(s) }
func (s ClientsList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s ClientsList) Less(i, j int) bool {
	if s[i].Hostname != "" && s[j].Hostname != "" {
		return sortutil.NaturalLess(s[i].Hostname, s[j].Hostname)
	}

	return s[i].ConnectionDate < s[j].ConnectionDate
}

// ////////////////////////////////////////////////////////////////////////////////// //

// map cid -> client info
var clients map[string]*ClientInfo

// command queue
var queue *API.CommandQueue

var (
	statusOK                = API.ResponseStatus{"OK", 0}
	statusArgError          = API.ResponseStatus{"Not enough arguments", API.STATUS_WRONG_ARGS}
	statusClientError       = API.ResponseStatus{"Request come from IP not associated with this client", API.STATUS_INCORRECT_REQUEST}
	statusIncompError       = API.ResponseStatus{"Client is not compatible with this master", API.STATUS_INCOMPATIBLE_CORE_VERSION}
	statusTokenError        = API.ResponseStatus{"Token is invalid", API.STATUS_WRONG_AUTH_TOKEN}
	statusWrongRequestError = API.ResponseStatus{"Wrong request", API.STATUS_WRONG_REQUEST}
)

// server server is HTTP server
var server *http.Server

// daemonVersion is current daemon version
var daemonVersion string

// statsInfo contains current stats
var statsInfo *API.StatsInfo

// ////////////////////////////////////////////////////////////////////////////////// //

// Start start sync daemon in master mode
func Start(app, ver, rev string) int {
	daemonVersion = ver

	var err error

	clients = make(map[string]*ClientInfo)
	queue = &API.CommandQueue{make([]*API.CommandQueueItem, 0), -1}

	err = restoreInstancesState()

	if err != nil {
		log.Crit("Can't restore instances state: %v", err)
		return EC_ERROR
	}

	collectInstancesData()

	addr := CORE.Config.GetS(CORE.REPLICATION_MASTER_IP) +
		":" + CORE.Config.GetS(CORE.REPLICATION_MASTER_PORT)

	startAPIServer(addr)

	go checkLoop()

	if rev == "" {
		log.Aux("%s %s started in MASTER mode (%s)", app, ver, addr)
	} else {
		log.Aux("%s %s (git:%s) started in MASTER mode (%s)", app, ver, rev, addr)
	}

	err = server.ListenAndServe()

	if err != nil && err != http.ErrServerClosed {
		log.Crit("HTTP Server error: %v", err)
		return EC_ERROR
	}

	return EC_OK
}

// Stop gracefully stops sync daemon HTTP server
func Stop() {
	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}
}

// ////////////////////////////////////////////////////////////////////////////////// //

// startAPIServer starts API HTTP server
func startAPIServer(addr string) {
	server = &http.Server{
		Addr:           addr,
		Handler:        http.NewServeMux(),
		ReadTimeout:    3 * time.Second,
		WriteTimeout:   3 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	registerAPIHandlers(server.Handler.(*http.ServeMux))
}

// registerAPIHandlers register all handlers
func registerAPIHandlers(mux *http.ServeMux) {
	mux.HandleFunc(API.METHOD_HELLO.Pattern(), helloHandler)
	mux.HandleFunc(API.METHOD_FETCH.Pattern(), fetchHandler)
	mux.HandleFunc(API.METHOD_INFO.Pattern(), infoHandler)
	mux.HandleFunc(API.METHOD_PUSH.Pattern(), pushHandler)
	mux.HandleFunc(API.METHOD_PULL.Pattern(), pullHandler)
	mux.HandleFunc(API.METHOD_STATS.Pattern(), statsHandler)
	mux.HandleFunc(API.METHOD_REPLICATION.Pattern(), replicationHandler)
	mux.HandleFunc(API.METHOD_BYE.Pattern(), byeHandler)
	mux.HandleFunc("/", anyHandler)
}

// anyHandler handler for any unsupported command
func anyHandler(w http.ResponseWriter, r *http.Request) {
	appendHeader(w)
	encodeAndWrite(w, &API.DefaultResponse{Status: statusWrongRequestError})
}

// helloHandler is "hello" command handler
func helloHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	appendHeader(w)

	if !checkAuthHeader(w, r, API.METHOD_HELLO) {
		return
	}

	if !checkRequestMethod(w, r, "POST", API.METHOD_HELLO) {
		return
	}

	helloRequest := &API.HelloRequest{}
	err = readAndDecode(r, helloRequest)

	if err != nil {
		encodeAndWrite(w, &API.DefaultResponse{Status: statusArgError})
		return
	}

	coreCompat := AUXI.GetCoreCompatibility(helloRequest.Version)

	if coreCompat == API.CORE_COMPAT_ERROR {
		encodeAndWrite(w, &API.DefaultResponse{Status: statusIncompError})
		return
	}

	auth, err := CORE.ReadSUAuth()

	if err != nil {
		log.Error("Can't read superuser auth data: %v", err)
	}

	helloResponse := &API.HelloResponse{
		Status:        statusOK,
		Version:       daemonVersion + "/" + CORE.VERSION,
		CID:           genCID(),
		SentinelWorks: CORE.IsSentinelActive(),
		Auth:          auth,
	}

	if coreCompat == API.CORE_COMPAT_PARTIAL {
		log.Warn("Client %s can be not fully compatible with this master", helloResponse.CID)
	}

	registerClient(httputil.GetRemoteHost(r), helloRequest, helloResponse.CID)

	err = encodeAndWrite(w, helloResponse)

	if err != nil {
		log.Error("Can't encode response: %v", err)
	}
}

// infoHandler is "info" command handler
func infoHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	appendHeader(w)

	if !checkAuthHeader(w, r, API.METHOD_INFO) {
		return
	}

	if !checkRequestMethod(w, r, "POST", API.METHOD_INFO) {
		return
	}

	infoRequest := &API.InfoRequest{}
	err = readAndDecode(r, infoRequest)

	if err != nil {
		encodeAndWrite(w, &API.DefaultResponse{Status: statusArgError})
		return
	}

	if !checkClient(w, r, infoRequest.CID, API.METHOD_INFO) {
		return
	}

	client := clients[infoRequest.CID]

	if !checkRequestHost(w, r, client.IP, API.METHOD_INFO) {
		return
	}

	client.LastSeen = time.Now().UnixNano()

	if !CORE.IsInstanceExist(infoRequest.ID) {
		encodeAndWrite(w, &API.DefaultResponse{
			Status: API.ResponseStatus{
				Code: API.STATUS_UNKNOWN_INSTANCE,
				Desc: fmt.Sprintf("Instance with ID %d does not exist", infoRequest.ID),
			},
		})

		return
	}

	meta, err := CORE.GetInstanceMeta(infoRequest.ID)

	if err != nil {
		encodeAndWrite(w, &API.DefaultResponse{
			Status: API.ResponseStatus{
				Code: API.STATUS_UNKNOWN_ERROR,
				Desc: fmt.Sprintf("Error getting instance %d meta: %v", infoRequest.ID, err),
			},
		})

		return
	}

	if infoRequest.UUID != meta.UUID {
		encodeAndWrite(w, &API.DefaultResponse{
			Status: API.ResponseStatus{
				Code: API.STATUS_UNKNOWN_INSTANCE,
				Desc: fmt.Sprintf("Instance with ID %d have UUID %s (%s in request data)", infoRequest.ID, meta.UUID, infoRequest.UUID),
			},
		})

		return
	}

	state, err := CORE.GetInstanceState(infoRequest.ID, false)

	if err != nil {
		encodeAndWrite(w, &API.DefaultResponse{
			Status: API.ResponseStatus{
				Code: API.STATUS_UNKNOWN_ERROR,
				Desc: fmt.Sprintf("Error getting instance %d state: %v", infoRequest.ID, err),
			},
		})

		return
	}

	if infoRequest.UUID != meta.UUID {
		encodeAndWrite(w, &API.DefaultResponse{
			Status: API.ResponseStatus{
				Code: API.STATUS_UNKNOWN_INSTANCE,
				Desc: fmt.Sprintf("Instance with ID %d have UUID %s (%s in request data)", infoRequest.ID, meta.UUID, infoRequest.UUID),
			},
		})

		return
	}

	infoResponse := &API.InfoResponse{
		Status: statusOK,
		Info:   &CORE.InstanceInfo{State: state, Meta: meta},
	}

	err = encodeAndWrite(w, infoResponse)

	if err != nil {
		log.Error("Can't encode response: %v", err)
	}
}

// pushHandler is "push" command handler
func pushHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	appendHeader(w)

	if !checkAuthHeader(w, r, API.METHOD_PUSH) {
		return
	}

	if !checkRequestMethod(w, r, "POST", API.METHOD_PUSH) {
		return
	}

	req := &API.PushRequest{}
	err = readAndDecode(r, req)

	if err != nil {
		encodeAndWrite(w, &API.DefaultResponse{Status: statusArgError})
		return
	}

	cIP, rIP := netutil.GetIP(), httputil.GetRemoteHost(r)

	if rIP != cIP && rIP != CORE.Config.GetS(CORE.REPLICATION_MASTER_IP, "127.0.0.1") {
		encodeAndWrite(w, &API.DefaultResponse{Status: statusClientError})
		return
	}

	pushResponse := &API.DefaultResponse{
		Status: statusOK,
	}

	if req.ID == -1 {
		log.Info(
			"Received push command (command: %s | initiator: %s)",
			req.Command, req.Initiator,
		)
	} else {
		log.Info(
			"Received push command (command: %s | initiator: %s | ID: %d | UUID: %s)",
			req.Command, req.Initiator, req.ID, req.UUID,
		)
	}

	processPushCommand(req.Command, req.Initiator, req.ID, req.UUID)

	err = encodeAndWrite(w, pushResponse)

	if err != nil {
		log.Error("Can't encode %s response: %v", API.METHOD_PUSH, err)
	}
}

// pullHandler is "pull" command handler
func pullHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	appendHeader(w)

	if !checkAuthHeader(w, r, API.METHOD_PULL) {
		return
	}

	if !checkRequestMethod(w, r, "POST", API.METHOD_PULL) {
		return
	}

	pullRequest := &API.DefaultRequest{}
	err = readAndDecode(r, pullRequest)

	if err != nil {
		encodeAndWrite(w, &API.DefaultResponse{Status: statusArgError})
		return
	}

	if !checkClient(w, r, pullRequest.CID, API.METHOD_PULL) {
		return
	}

	client := clients[pullRequest.CID]

	if !checkRequestHost(w, r, client.IP, API.METHOD_PULL) {
		return
	}

	client.LastSeen = time.Now().UnixNano()

	pullResponse := &API.PullResponse{
		Status:   statusOK,
		Commands: getItemsFromQueue(client.LastSync),
	}

	err = encodeAndWrite(w, pullResponse)

	if err != nil {
		log.Error("Can't encode %s response: %v", API.METHOD_PULL, err)
		return
	}

	if client.Syncing {
		log.Info("Client with ID %s finished initial synchronization", client.CID)
	}

	client.LastSync = time.Now().UnixNano()
	client.Syncing = false
}

// fetchHandler is "fetch" command handler
func fetchHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	appendHeader(w)

	if !checkAuthHeader(w, r, API.METHOD_FETCH) {
		return
	}

	if !checkRequestMethod(w, r, "POST", API.METHOD_FETCH) {
		return
	}

	fetchRequest := &API.DefaultRequest{}
	err = readAndDecode(r, fetchRequest)

	if err != nil {
		encodeAndWrite(w, &API.DefaultResponse{Status: statusArgError})
		return
	}

	if !checkClient(w, r, fetchRequest.CID, API.METHOD_FETCH) {
		return
	}

	client := clients[fetchRequest.CID]

	if !checkRequestHost(w, r, client.IP, API.METHOD_FETCH) {
		return
	}

	client.LastSeen = time.Now().UnixNano()
	client.Syncing = true

	fetchResponse := &API.FetchResponse{
		Status:    statusOK,
		Instances: collectInstancesData(),
	}

	err = encodeAndWrite(w, fetchResponse)

	if err != nil {
		log.Error("Can't encode %s response: %v", API.METHOD_FETCH, err)
		client.Syncing = false
		return
	}

	maxSyncWait := CORE.Config.GetD(CORE.REPLICATION_MAX_SYNC_WAIT, knf.Second)
	maxInitTimeDur := maxSyncWait * time.Duration(len(fetchResponse.Instances))
	deadline := time.Now().Add(maxInitTimeDur)

	log.Info(
		"Client with ID %s started initial synchronization process (deadline: %s)",
		client.CID, timeutil.Format(deadline, "%Y/%m/%d %H:%M:%S"),
	)

	client.LastSync = time.Now().UnixNano()
}

// replicationHandler is "replication" command handler
func replicationHandler(w http.ResponseWriter, r *http.Request) {
	appendHeader(w)

	if !checkAuthHeader(w, r, API.METHOD_REPLICATION) {
		return
	}

	if !checkRequestMethod(w, r, "GET", API.METHOD_REPLICATION) {
		return
	}

	ip := httputil.GetRemoteHost(r)

	replicationResponse := &API.ReplicationResponse{
		Status: statusOK,
		Info: &API.ReplicationInfo{
			Master:  getMasterInfo(),
			Clients: getClientsInfo(),
		},
	}

	replicationResponse.Info.SuppliantCID = getSuppliantCID(ip, replicationResponse.Info.Clients)

	err := encodeAndWrite(w, replicationResponse)

	if err != nil {
		log.Error("Can't encode %s response: %v", API.METHOD_REPLICATION, err)
		return
	}
}

// statsHandler is "stats" command handler
func statsHandler(w http.ResponseWriter, r *http.Request) {
	appendHeader(w)

	if !checkRequestMethod(w, r, "GET", API.METHOD_STATS) {
		return
	}

	if statsInfo == nil {
		statsInfo = &API.StatsInfo{}
	}

	// Reset stats
	statsInfo.Minions = 0
	statsInfo.Sentinels = 0
	statsInfo.MaxSeenLag = 0
	statsInfo.MaxSyncLag = 0

	now := time.Now().UnixNano()

	for _, client := range clients {
		switch client.Role {
		case CORE.ROLE_MINION:
			statsInfo.Minions++
		case CORE.ROLE_SENTINEL:
			statsInfo.Sentinels++
		}

		seenLag := float64(now-client.LastSeen) / 1_000_000_000.0
		syncLag := float64(now-client.LastSync) / 1_000_000_000.0

		seenLag = mathutil.Round(seenLag, 3)
		syncLag = mathutil.Round(syncLag, 3)

		statsInfo.MaxSeenLag = math.Max(statsInfo.MaxSeenLag, seenLag)
		statsInfo.MaxSyncLag = math.Max(statsInfo.MaxSyncLag, syncLag)
	}

	err := encodeAndWrite(w, &API.StatsResponse{statusOK, statsInfo})

	if err != nil {
		log.Error("Can't encode %s response: %v", API.METHOD_STATS, err)
		return
	}
}

// byeHandler is "bye" command handler
func byeHandler(w http.ResponseWriter, r *http.Request) {
	appendHeader(w)

	if !checkAuthHeader(w, r, API.METHOD_BYE) {
		return
	}

	if !checkRequestMethod(w, r, "POST", API.METHOD_BYE) {
		return
	}

	byeRequest := &API.ByeRequest{}
	err := readAndDecode(r, byeRequest)

	if err != nil {
		encodeAndWrite(w, &API.DefaultResponse{Status: statusArgError})
		return
	}

	client, ok := clients[byeRequest.CID]
	ip := httputil.GetRemoteHost(r)

	if ip != client.IP {
		log.Error(
			"Got bye request from client with CID %s: Client with given CID registered from different IP (%s ≠ %s)",
			byeRequest.CID, ip, client.IP,
		)

		encodeAndWrite(w, &API.DefaultResponse{
			Status: API.ResponseStatus{
				Code: API.STATUS_UNKNOWN_CLIENT,
				Desc: fmt.Sprintf("Client with ID %s registered from different IP", byeRequest.CID),
			},
		})

		return
	}

	if !ok {
		log.Warn(
			"Got bye request from client with CID %s: There is no client with given CID",
			byeRequest.CID,
		)

		encodeAndWrite(w, &API.DefaultResponse{
			Status: API.ResponseStatus{
				Code: API.STATUS_UNKNOWN_CLIENT,
				Desc: fmt.Sprintf("Client with ID %s is not found", byeRequest.CID),
			},
		})

		return
	}

	log.Info(
		"Got bye request from client with CID %s: Client successfully unregistered",
		byeRequest.CID,
	)

	delete(clients, byeRequest.CID)

	encodeAndWrite(w, &API.DefaultResponse{Status: statusOK})
}

// ////////////////////////////////////////////////////////////////////////////////// //

// checkRequestMethod checks http method and writes error to writer if method is not
// supported
func checkRequestMethod(w http.ResponseWriter, r *http.Request, method string, apiMethod API.Method) bool {
	if r.Method == method {
		return true
	}

	ip := httputil.GetRemoteHost(r)

	log.Error(
		"{%s:%s:%s} Got request with unsupported HTTP method (%s ≠ %s)",
		r.Method, ip, apiMethod, r.Method, method,
	)

	encodeAndWrite(w, &API.DefaultResponse{
		Status: API.ResponseStatus{
			Code: API.STATUS_WRONG_METHOD,
			Desc: fmt.Sprintf("Method %s is not supported", r.Method),
		},
	})

	return false
}

// checkRequestHost checks request host and writes error to writer if request come from
// unknown IP
func checkRequestHost(w http.ResponseWriter, r *http.Request, clientIP string, apiMethod API.Method) bool {
	rIP := httputil.GetRemoteHost(r)

	if rIP == clientIP {
		return true
	}

	ip := httputil.GetRemoteHost(r)

	log.Error(
		"{%s:%s:%s} Got request from unknown IP (%s ≠ %s)",
		r.Method, ip, apiMethod, ip, clientIP,
	)

	encodeAndWrite(w, &API.DefaultResponse{Status: statusClientError})

	return false
}

// checkClient checks client ID and writes error to writer if request come from unknown client
func checkClient(w http.ResponseWriter, r *http.Request, cid string, apiMethod API.Method) bool {
	if clients[cid] != nil {
		return true
	}

	ip := httputil.GetRemoteHost(r)

	log.Error(
		"{%s:%s:%s} Got request from unknown client (%s)",
		r.Method, ip, apiMethod, cid,
	)

	encodeAndWrite(w, &API.DefaultResponse{
		Status: API.ResponseStatus{
			Code: API.STATUS_UNKNOWN_CLIENT,
			Desc: fmt.Sprintf("Client with ID %s is not found", cid),
		},
	})

	return false
}

// checkAuthHeader checks request headers for token and writes error to writer if
// token is invalid
func checkAuthHeader(w http.ResponseWriter, r *http.Request, apiMethod API.Method) bool {
	token := CORE.Config.GetS(CORE.REPLICATION_AUTH_TOKEN)

	for headerName, header := range r.Header {
		if headerName != "Authorization" {
			continue
		}

		if strings.Join(header, " ") == "Bearer "+token {
			return true
		}
	}

	ip := httputil.GetRemoteHost(r)

	log.Error("{%s:%s:%s} Got request with unknown auth token", r.Method, ip, apiMethod)

	encodeAndWrite(w, &API.DefaultResponse{Status: statusTokenError})

	return false
}

// restoreInstancesState restores state of every instance
func restoreInstancesState() error {
	statesFile := CORE.GetStatesFilePath()

	if !fsutil.IsExist(statesFile) {
		return nil
	}

	statesInfo, err := CORE.ReadStates(statesFile)

	if err != nil {
		return fmt.Errorf("Can't read states file: %v", err)
	}

	if isStateInconsistent(statesInfo) {
		if !statesInfo.Sentinel {
			return fmt.Errorf("Instances state is inconsistent. You must restore state using 'state-restore' command.")
		}
	} else {
		return nil
	}

	log.Info("Restoring states…")

	for _, stateInfo := range statesInfo.States {
		if !CORE.IsInstanceExist(stateInfo.ID) {
			continue
		}

		state, err := CORE.GetInstanceState(stateInfo.ID, false)

		if err != nil {
			return fmt.Errorf("Can't check state of instance %d: %v", stateInfo.ID, err)
		}

		if state == stateInfo.State || state.IsDead() == stateInfo.State.IsStopped() {
			continue
		}

		switch {
		case stateInfo.State.IsWorks():
			err = CORE.StartInstance(stateInfo.ID, true)
		case stateInfo.State.IsStopped():
			err = CORE.StopInstance(stateInfo.ID, false)
		}

		if err != nil {
			return fmt.Errorf("Can't restore state of instance %d: %v", stateInfo.ID, err)
		}
	}

	log.Info("State successfully restored")

	return nil
}

// isStateInconsistent returns true is system state is inconsistent
func isStateInconsistent(statesInfo *CORE.StatesInfo) bool {
	for _, stateInfo := range statesInfo.States {
		if !CORE.IsInstanceExist(stateInfo.ID) {
			continue
		}

		state, err := CORE.GetInstanceState(stateInfo.ID, false)

		if err != nil {
			return true
		}

		if state == stateInfo.State || (state.IsDead() && stateInfo.State.IsStopped()) {
			continue
		}

		return true
	}

	return false
}

// collectInstancesData collect info about all instances in first time
func collectInstancesData() []*CORE.InstanceInfo {
	idList := CORE.GetInstanceIDList()

	var result []*CORE.InstanceInfo

	if len(idList) == 0 {
		return result
	}

	for _, id := range idList {
		state, err := CORE.GetInstanceState(id, false)

		if err != nil {
			continue
		}

		meta, err := CORE.GetInstanceMeta(id)

		if err != nil {
			continue
		}

		result = append(result, &CORE.InstanceInfo{State: state, Meta: meta})
	}

	return result
}

// appendHeader append header to response
func appendHeader(w http.ResponseWriter) {
	w.Header().Set("Server", "RDS-Sync/"+daemonVersion)
	w.Header().Set("Content-Type", "application/json")
}

// encodeAndWrite encode struct to json and write as response
func encodeAndWrite(w http.ResponseWriter, data any) error {
	jd, err := json.MarshalIndent(data, "", "  ")

	if err != nil {
		return err
	}

	w.WriteHeader(200)
	w.Write(jd)

	return nil
}

// readAndDecode read json data from request and decode
func readAndDecode(r *http.Request, v any) error {
	decoder := json.NewDecoder(r.Body)

	return decoder.Decode(v)
}

// registerClient register client in index
func registerClient(ip string, request *API.HelloRequest, cid string) {
	now := time.Now()

	client := &ClientInfo{
		CID:            cid,
		Version:        request.Version,
		Hostname:       request.Hostname,
		Role:           request.Role,
		IP:             ip,
		State:          API.STATE_ONLINE,
		LastSeen:       now.UnixNano(),
		LastSync:       now.UnixNano(),
		ConnectionDate: now.Unix(),
	}

	hasClientFromIP, clientCID := hasClient(client.IP)

	if hasClientFromIP {
		log.Info(
			"Client with CID %s unregistered: New hello request received from IP (%s) associated with this client",
			clientCID, client.IP,
		)

		delete(clients, clientCID)
	}

	clients[cid] = client

	log.Info("Registered client %d:%s (%s)", len(clients), cid, renderClientInfo(client))
}

// getMasterInfo return info about master
func getMasterInfo() *API.MasterInfo {
	ip := CORE.Config.GetS(CORE.REPLICATION_MASTER_IP)

	if ip == "" {
		ip = netutil.GetIP()
	}

	hostname, _ := os.Hostname()

	return &API.MasterInfo{
		Version:  daemonVersion + "/" + CORE.VERSION,
		IP:       ip,
		Hostname: hostname,
	}
}

// getClientsInfo return slice with info about clients
func getClientsInfo() []*API.ClientInfo {
	var result []*API.ClientInfo

	if len(clients) == 0 {
		return result
	}

	now := time.Now().UnixNano()

	for _, client := range clients {
		seenLag := mathutil.Round(float64(now-client.LastSeen)/1_000_000_000.0, 3)
		syncLag := mathutil.Round(float64(now-client.LastSync)/1_000_000_000.0, 3)

		result = append(result, &API.ClientInfo{
			CID:            client.CID,
			Role:           client.Role,
			Version:        client.Version,
			Hostname:       client.Hostname,
			IP:             client.IP,
			LastSeenLag:    seenLag,
			LastSyncLag:    syncLag,
			ConnectionDate: client.ConnectionDate,
			State:          getClientState(now, client),
		})
	}

	sort.Sort(ClientsList(result))

	return result
}

// getSuppliantCID try to find CID of suppliant
func getSuppliantCID(ip string, clients []*API.ClientInfo) string {
	for _, client := range clients {
		if client.IP == ip {
			return client.CID
		}
	}

	return ""
}

// getClientState calculate current client state
func getClientState(now int64, client *ClientInfo) API.ClientState {
	timeDiff := now - client.LastSeen

	if client.Syncing {
		return API.STATE_SYNCING
	}

	switch {
	case timeDiff <= DELAY_POSSIBLE_DOWN*1_000_000_000:
		return API.STATE_ONLINE

	case timeDiff <= DELAY_DOWN*1_000_000_000:
		return API.STATE_POSSIBLE_DOWN

	case timeDiff <= DELAY_DEAD*1_000_000_000:
		return API.STATE_DOWN

	default:
		return API.STATE_DEAD
	}
}

// getItemsFromQueue return items from queue
func getItemsFromQueue(lastSync int64) []*API.CommandQueueItem {
	var items = make([]*API.CommandQueueItem, 0)

	if lastSync > queue.ModTime || len(queue.Items) == 0 {
		return items
	}

	for _, item := range queue.Items {
		if item.Timestamp > lastSync {
			items = append(items, item)
		}
	}

	return items
}

// processPushCommand process push command
func processPushCommand(command API.MasterCommand, initiator string, id int, uuid string) {
	ts := time.Now().UnixNano()

	item := &API.CommandQueueItem{
		Command:      command,
		InstanceID:   id,
		InstanceUUID: uuid,
		Timestamp:    ts,
		Initiator:    initiator,
	}

	queue.Items = append(queue.Items, item)
	queue.ModTime = ts
}

// checkLoop cleans command queue and checks clients status
func checkLoop() {
	for range time.Tick(time.Minute) {
		cleanupQueue()
		checkClientsStatus()
	}
}

// cleanupQueue remove old items from queue
func cleanupQueue() {
	if len(queue.Items) == 0 {
		return
	}

	items := queue.Items

	now := time.Now().UnixNano()
	mts := now - (DELAY_DEAD * 1_000_000_000)

	for {
		if len(items) == 0 {
			break
		}

		item := items[0]

		if item.Timestamp < mts {
			items = items[1:]
		} else {
			break
		}
	}
}

// checkClientsStatus check status for each client
func checkClientsStatus() {
	now := time.Now().UnixNano()

	for _, client := range clients {
		state := getClientState(now, client)

		if state == API.STATE_ONLINE {
			if state != client.State {
				log.Info(
					"Client with CID %s (%s) is back to online",
					client.CID, renderClientInfo(client),
				)

				client.State = state
			}

			continue
		}

		switch state {
		case API.STATE_POSSIBLE_DOWN:
			if state != client.State {
				log.Warn(
					"Client with CID %s (%s) is possibly down",
					client.CID, renderClientInfo(client),
				)

				client.State = state
			}

		case API.STATE_DOWN:
			if state != client.State {
				log.Warn(
					"Client with CID %s (%s) is down",
					client.CID, renderClientInfo(client),
				)

				client.State = state
			}

		case API.STATE_DEAD:
			log.Warn(
				"Client with CID %s (%s) unregistered: client inactive more than %s",
				client.CID, renderClientInfo(client), timeutil.PrettyDuration(DELAY_DEAD),
			)

			delete(clients, client.CID)
		}
	}
}

// hasClient returns true and CID if master has registered client from given IP
func hasClient(ip string) (bool, string) {
	for cid, client := range clients {
		if client.IP == ip {
			return true, cid
		}
	}

	return false, ""
}

// genCID return new client id
func genCID() string {
	hash := crc32.NewIEEE()
	timestamp := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)

	hash.Write([]byte(timestamp))

	return fmt.Sprintf("%08x", hash.Sum32())
}

// renderClientInfo return client info as string
func renderClientInfo(client *ClientInfo) string {
	if client.Hostname == "" {
		return fmt.Sprintf(
			"Role: %s | Version: %s | IP: %s",
			client.Role, client.Version, client.IP,
		)
	}

	return fmt.Sprintf(
		"Role: %s | Version: %s | Hostname: %s | IP: %s",
		client.Role, client.Version, client.Hostname, client.IP,
	)
}
