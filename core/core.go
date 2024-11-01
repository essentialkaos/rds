package core

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/essentialkaos/ek/v13/env"
	"github.com/essentialkaos/ek/v13/errors"
	"github.com/essentialkaos/ek/v13/fsutil"
	"github.com/essentialkaos/ek/v13/initsystem"
	"github.com/essentialkaos/ek/v13/jsonutil"
	"github.com/essentialkaos/ek/v13/knf"
	"github.com/essentialkaos/ek/v13/log"
	"github.com/essentialkaos/ek/v13/mathutil"
	"github.com/essentialkaos/ek/v13/netutil"
	"github.com/essentialkaos/ek/v13/passwd"
	"github.com/essentialkaos/ek/v13/path"
	"github.com/essentialkaos/ek/v13/pid"
	"github.com/essentialkaos/ek/v13/strutil"
	"github.com/essentialkaos/ek/v13/system"
	"github.com/essentialkaos/ek/v13/system/process"
	"github.com/essentialkaos/ek/v13/uuid"
	"github.com/essentialkaos/ek/v13/version"

	knfv "github.com/essentialkaos/ek/v13/knf/validators"
	knff "github.com/essentialkaos/ek/v13/knf/validators/fs"
	knfn "github.com/essentialkaos/ek/v13/knf/validators/network"
	knfs "github.com/essentialkaos/ek/v13/knf/validators/system"

	REDIS "github.com/essentialkaos/rds/redis"
	SENTINEL "github.com/essentialkaos/rds/sentinel"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// VERSION is current core version
const VERSION = "A2"

// META_VERSION is current meta version
const META_VERSION = 1

// Limits
const (
	MIN_INSTANCES        = 16
	MAX_INSTANCES        = 1024
	MIN_PASS_LENGTH      = 6
	MAX_PASS_LENGTH      = 64
	MIN_DESC_LENGTH      = 6
	MAX_DESC_LENGTH      = 64
	MIN_PORT             = 1025
	MAX_PORT             = 65535
	MIN_PROCS            = 10240
	MIN_SOMAXCONN        = 2047
	MIN_STOP_DELAY       = 1      // 1 Sec
	MAX_STOP_DELAY       = 5 * 60 // 5 Min
	MIN_START_DELAY      = 1      // 1 Sec
	MAX_START_DELAY      = 5 * 60 // 5 Min
	MIN_THREADS          = 1
	MAX_THREADS          = 32
	MIN_CYCLE_TIME       = 15 * 60      // 15 Min
	MAX_CYCLE_TIME       = 24 * 60 * 60 // 1 Day
	MIN_CHANGES_NUM      = 1
	MAX_TAGS             = 3
	MIN_SYNC_WAIT        = 60          // 1 Min
	MAX_SYNC_WAIT        = 3 * 60 * 60 // 3 Hours
	MAX_FULL_START_DELAY = 30 * 60     // 30 Min
	MAX_SWITCH_WAIT      = 15 * 60     // 15 Min
	TOKEN_LENGTH         = 64
	MIN_SENTINEL_VERSION = 5
	MIN_NICE             = -20
	MAX_NICE             = 19
	MIN_IONICE_CLASS     = 0
	MAX_IONICE_CLASS     = 3
	MIN_IONICE_CLASSDATA = 0
	MAX_IONICE_CLASSDATA = 7
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	BIN_RUNUSER = "/sbin/runuser"
)

const (
	ROLE_MASTER   = "master"
	ROLE_MINION   = "minion"
	ROLE_SENTINEL = "sentinel"
)

const (
	SU_DATA_FILE            = "su.dat"
	REDIS_VERSION_DATA_FILE = "redis.dat"
	STATES_DATA_FILE        = "states.dat"
	IDS_DATA_FILE           = "ids.dat"
)

const (
	LOG_FILE_CLI  = "rds.log"
	LOG_FILE_SYNC = "rds-sync.log"
)

const (
	PID_SENTINEL = "sentinel.pid"
)

const (
	MAIN_MAX_INSTANCES               = "main:max-instances"
	MAIN_ALLOW_ID_REUSE              = "main:allow-id-reuse"
	MAIN_DISABLE_CONFIGURATION_CHECK = "main:disable-configuration-check"
	MAIN_DISABLE_FILESYSTEM_CHECK    = "main:disable-filesystem-check"
	MAIN_DISABLE_IP_CHECK            = "main:disable-ip-check"
	MAIN_DISABLE_TIPS                = "main:disable-tips"
	MAIN_WARN_USED_MEMORY            = "main:warn-used-memory"
	MAIN_DIR                         = "main:dir"
	MAIN_MIN_PASS_LENGTH             = "main:min-pass-length"
	MAIN_STRICT_SECURE               = "main:strict-secure"
	MAIN_HOSTNAME                    = "main:hostname"

	LOG_LEVEL = "log:level"

	REDIS_BINARY           = "redis:binary"
	REDIS_USER             = "redis:user"
	REDIS_START_PORT       = "redis:start-port"
	REDIS_SAVE_ON_STOP     = "redis:save-on-stop"
	REDIS_NICE             = "redis:nice"
	REDIS_IONICE_CLASS     = "redis:ionice-class"
	REDIS_IONICE_CLASSDATA = "redis:ionice-classdata"

	SENTINEL_BINARY           = "sentinel:binary"
	SENTINEL_PORT             = "sentinel:port"
	SENTINEL_QUORUM           = "sentinel:quorum"
	SENTINEL_DOWN_AFTER       = "sentinel:down-after-milliseconds"
	SENTINEL_PARALLEL_SYNCS   = "sentinel:parallel-syncs"
	SENTINEL_FAILOVER_TIMEOUT = "sentinel:failover-timeout"

	KEEPALIVED_VIRTUAL_IP = "keepalived:virtual-ip"

	TEMPLATES_REDIS    = "templates:redis"
	TEMPLATES_SENTINEL = "templates:sentinel"

	PATH_META_DIR   = "path:meta-dir"
	PATH_CONFIG_DIR = "path:config-dir"
	PATH_DATA_DIR   = "path:data-dir"
	PATH_PID_DIR    = "path:pid-dir"
	PATH_LOG_DIR    = "path:log-dir"

	REPLICATION_ROLE                = "replication:role"
	REPLICATION_MASTER_IP           = "replication:master-ip"
	REPLICATION_MASTER_PORT         = "replication:master-port"
	REPLICATION_AUTH_TOKEN          = "replication:auth-token"
	REPLICATION_FAILOVER_METHOD     = "replication:failover-method"
	REPLICATION_DEFAULT_ROLE        = "replication:default-role"
	REPLICATION_CHECK_READONLY_MODE = "replication:check-readonly-mode"
	REPLICATION_ALLOW_REPLICAS      = "replication:allow-replicas"
	REPLICATION_ALLOW_COMMANDS      = "replication:allow-commands"
	REPLICATION_ALWAYS_PROPAGATE    = "replication:always-propagate"
	REPLICATION_MAX_SYNC_WAIT       = "replication:max-sync-wait"
	REPLICATION_INIT_SYNC_DELAY     = "replication:init-sync-delay"

	DELAY_START = "delay:start"
	DELAY_STOP  = "delay:stop"
)

const (
	REDIS_USER_ADMIN    = "admin"
	REDIS_USER_SYNC     = "sync"
	REDIS_USER_SERVICE  = "service"
	REDIS_USER_SENTINEL = "sentinel"
)

// DEFAULT_FILE_PERMS is default permissions for files created by core
const DEFAULT_FILE_PERMS = 0600

// ////////////////////////////////////////////////////////////////////////////////// //

type FailoverMethod string

const (
	FAILOVER_METHOD_STANDBY  FailoverMethod = "standby"
	FAILOVER_METHOD_SENTINEL FailoverMethod = "sentinel"
)

type KeepalivedState uint8

const (
	KEEPALIVED_STATE_UNKNOWN KeepalivedState = 0
	KEEPALIVED_STATE_MASTER  KeepalivedState = 1
	KEEPALIVED_STATE_BACKUP  KeepalivedState = 2
)

type State uint16

const (
	INSTANCE_STATE_UNKNOWN State = 0
	INSTANCE_STATE_STOPPED State = 1 << iota
	INSTANCE_STATE_WORKS
	INSTANCE_STATE_DEAD
	INSTANCE_STATE_IDLE         // Extended
	INSTANCE_STATE_SYNCING      // Extended
	INSTANCE_STATE_LOADING      // Extended
	INSTANCE_STATE_SAVING       // Extended
	INSTANCE_STATE_HANG         // Extended
	INSTANCE_STATE_ABANDONED    // Extended
	INSTANCE_STATE_MASTER_UP    // Extended
	INSTANCE_STATE_MASTER_DOWN  // Extended
	INSTANCE_STATE_NO_REPLICA   // Extended
	INSTANCE_STATE_WITH_REPLICA // Extended
	INSTANCE_STATE_WITH_ERRORS  // Extended
)

type ReplicationType string

const (
	REPL_TYPE_REPLICA ReplicationType = "replica"
	REPL_TYPE_STANDBY ReplicationType = "standby"
)

type TemplateSource string

const (
	TEMPLATE_SOURCE_REDIS    TemplateSource = "redis.conf"
	TEMPLATE_SOURCE_SENTINEL TemplateSource = "sentinel.conf"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type SystemStatus struct {
	HasProblems     bool
	HasTHPIssues    bool
	HasKernelIssues bool
	HasLimitsIssues bool
	HasFSIssues     bool
}

type SuperuserAuth struct {
	Pepper string `json:"pepper"`
	Hash   string `json:"hash"`
}

type InstanceAuth struct {
	User   string `json:"user"`
	Pepper string `json:"pepper"`
	Hash   string `json:"hash"`
}

type InstanceMeta struct {
	Tags         []string              `json:"tags,omitempty"`       // List of tags
	Desc         string                `json:"desc"`                 // Description
	UUID         string                `json:"uuid"`                 // UUID
	Compatible   string                `json:"compatible,omitempty"` // Compatible redis version
	MetaVersion  int                   `json:"meta_version"`         // Meta information version
	ID           int                   `json:"id"`                   // Instance ID
	Created      int64                 `json:"created"`              // Date of creation (unix timestamp)
	Preferencies *InstancePreferencies `json:"preferencies"`         // Config data
	Config       *InstanceConfigInfo   `json:"config"`               // Config info (hash + creation date)
	Auth         *InstanceAuth         `json:"auth"`                 // Instance auth info
	Storage      Storage               `json:"storage,omitempty"`    // Core version agnostic data storage
}

type InstanceConfigInfo struct {
	Hash string `json:"hash"`
	Date int64  `json:"date"`
}

type InstancePreferencies struct {
	AdminPassword    string          `json:"admin_password,omitempty"`   // Admin user password
	SyncPassword     string          `json:"sync_password,omitempty"`    // Sync user password
	ServicePassword  string          `json:"service_password,omitempty"` // Service user password
	SentinelPassword string          `json:"sentinel_password"`          // Sentinel user password
	ReplicationType  ReplicationType `json:"replication_type"`           // Replication type
	IsSaveDisabled   bool            `json:"is_save_disabled"`           // Disabled saves flag
}

type InstanceInfo struct {
	Meta  *InstanceMeta `json:"meta"`
	State State         `json:"state"`
}

type RedisVersionInfo struct {
	CDate   int64  `json:"cdate"`
	Version string `json:"version"`
}

type StatesInfo struct {
	States   []StateInfo `json:"states"`
	Sentinel bool        `json:"sentinel"`
}

type StateInfo struct {
	ID    int   `json:"id"`
	State State `json:"state"`
}

type IDSInfo struct {
	LastID int `json:"last_id"`
}

type StatsInstances struct {
	Total         uint64 `json:"total_instances"`          // Total number of instances
	Active        uint64 `json:"active_instances"`         // Working instances
	Dead          uint64 `json:"dead_instances"`           // Dead instances
	BgSave        uint64 `json:"bgsave_instances"`         // Instances which save data in the background
	Syncing       uint64 `json:"syncing_instances"`        // Instances which currently sync data with master or replica
	AOFRewrite    uint64 `json:"aof_rewrite_instances"`    // Instances which currently rewrite aof data
	SaveFailed    uint64 `json:"save_failed_instances"`    // Instances with failed save
	ActiveMaster  uint64 `json:"active_master_instances"`  // Instances with active sync as a master
	ActiveReplica uint64 `json:"active_replica_instances"` // Instances with active sync as a replica
	Outdated      uint64 `json:"outdated_instances"`       // Outdated instances (newer version is installed but not used)
}

type StatsClients struct {
	Connected uint64 `json:"connected_clients"`
	Blocked   uint64 `json:"blocked_clients"`
}

type StatsMemory struct {
	TotalSystemMemory uint64 `json:"total_system_memory"`
	SystemMemory      uint64 `json:"system_memory"`
	TotalSystemSwap   uint64 `json:"total_system_swap"`
	SystemSwap        uint64 `json:"system_swap"`
	UsedMemory        uint64 `json:"used_memory"`
	UsedMemoryRSS     uint64 `json:"used_memory_rss"`
	UsedMemoryLua     uint64 `json:"used_memory_lua"`
	UsedSwap          uint64 `json:"used_swap"`
	IsSwapEnabled     bool   `json:"is_swap_enabled"`
}

type StatsOverall struct {
	TotalConnectionsReceived uint64 `json:"total_connections_received"`
	TotalCommandsProcessed   uint64 `json:"total_commands_processed"`
	InstantaneousOpsPerSec   uint64 `json:"instantaneous_ops_per_sec"`
	InstantaneousInputKbps   uint64 `json:"instantaneous_input_kbps"`
	InstantaneousOutputKbps  uint64 `json:"instantaneous_output_kbps"`
	RejectedConnections      uint64 `json:"rejected_connections"`
	ExpiredKeys              uint64 `json:"expired_keys"`
	EvictedKeys              uint64 `json:"evicted_keys"`
	KeyspaceHits             uint64 `json:"keyspace_hits"`
	KeyspaceMisses           uint64 `json:"keyspace_misses"`
	PubsubChannels           uint64 `json:"pubsub_channels"`
	PubsubPatterns           uint64 `json:"pubsub_patterns"`
}

type StatsKeys struct {
	Total   uint64 `json:"total_keys"`
	Expires uint64 `json:"expires_keys"`
}

type Stats struct {
	Instances *StatsInstances `json:"instances"`
	Clients   *StatsClients   `json:"clients"`
	Memory    *StatsMemory    `json:"memory"`
	Overall   *StatsOverall   `json:"overall"`
	Keys      *StatsKeys      `json:"keys"`
}

// ////////////////////////////////////////////////////////////////////////////////// //

// aligo:ignore
type instanceConfigData struct {
	Redis            version.Version
	RDS              *instanceConfigRDSData
	ID               int
	AdminPassword    string
	SyncPassword     string
	SentinelPassword string
	ServicePassword  string
	IsSecure         bool
	IsSaveDisabled   bool
	IsReplica        bool

	tags    []string
	storage Storage
}

type instanceConfigRDSData struct {
	HasReplication bool

	IsMaster   bool
	IsMinion   bool
	IsSentinel bool

	IsFailoverStandby  bool
	IsFailoverSentinel bool
}

type sentinelConfigData struct {
	Port    string
	PidFile string
	LogFile string
}

// ////////////////////////////////////////////////////////////////////////////////// //

var (
	ErrUnprivileged              = errors.New("RDS requires root privileges")
	ErrSUAuthAlreadyExist        = errors.New("Superuser credentials already generated")
	ErrSUAuthIsEmpty             = errors.New("Superuser auth data can't be empty")
	ErrSUAuthNoData              = errors.New("Superuser auth data doesn't exist")
	ErrCantDiffConfigs           = errors.New("Instance should works for configs comparison")
	ErrEmptyIDDBPair             = errors.New("ID/DB value is empty")
	ErrCantReadPID               = errors.New("Can't read PID from PID file")
	ErrStateFileNotDefined       = errors.New("You must define path to states file")
	ErrInstanceStillWorks        = errors.New("Instance still works")
	ErrSentinelRoleSetNotAllowed = errors.New("This node must have master role for switching instance role")
	ErrSentinelWrongInstanceRole = errors.New("Instance must have a replica role")
	ErrIncompatibleFailover      = errors.New("Action can't be done due to incompatibility with failover method defined in the configuration file")
	ErrSentinelWrongVersion      = errors.New("Sentinel monitoring requires Redis 5 or greater")
	ErrSentinelCantStart         = errors.New("Can't start Sentinel process")
	ErrSentinelCantStop          = errors.New("Can't stop Sentinel process")
	ErrSentinelIsStopped         = errors.New("Sentinel is stopped")
	ErrSentinelCantSetRole       = errors.New("Can't set instance role - instance still have slave (replica) role")
	ErrMetaIsNil                 = errors.New("Meta struct is nil")
	ErrMetaNoID                  = errors.New("Meta must have valid ID")
	ErrMetaNoDesc                = errors.New("Meta must have valid description")
	ErrMetaNoAuth                = errors.New("Meta doesn't have auth info")
	ErrMetaNoPrefs               = errors.New("Meta doesn't have instance preferencies")
	ErrMetaNoConfigInfo          = errors.New("Meta doesn't have info about Redis configuration file")
	ErrMetaInvalidVersion        = errors.New("Meta must have valid version")
	ErrInvalidRedisVersionCache  = errors.New("Cache is invalid")
	ErrCantParseRedisVersion     = errors.New("Can't parse version of redis-server")
	ErrCantReadRedisCreationDate = errors.New("Can't read creation date of redis-server file")
	ErrCantReadDaemonizeOption   = errors.New("Can't read 'daemonize' option value from instance configuration file")
	ErrCantDaemonizeInstance     = errors.New("Impossible to run instance - 'daemonize' property set to 'no' in configuration file")
	ErrUnknownReplicationType    = errors.New("Unsupported replication type")
	ErrUnknownTemplateSource     = errors.New("Unknown template source")
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Config is application config data
var Config *knf.Config

// User is current user info
var User *system.User

// ////////////////////////////////////////////////////////////////////////////////// //

// metaCache is meta information cache
var metaCache *MetaCache

// globalConfig is path to configuration file
var globalConfig string

// redisVersion contains current Redis version
var redisVersion version.Version

// supportedVersions is map with supported Redis versions
var supportedVersions = map[string]bool{
	"6.2": true,
	"7.0": true,
	"7.2": true,
	"7.4": true,
}

// tagRegex is regex pattern for tag validation
var tagRegex = regexp.MustCompile(`^[A-Za-z0-9_\-+]+$`)

// ////////////////////////////////////////////////////////////////////////////////// //

// IsWorks returns true if instance works
func (s State) IsWorks() bool {
	return s&INSTANCE_STATE_WORKS == INSTANCE_STATE_WORKS
}

// IsStopped returns true if instance stopped
func (s State) IsStopped() bool {
	return s&INSTANCE_STATE_STOPPED == INSTANCE_STATE_STOPPED
}

// IsDead returns true if instance dead (pid exist, but process not exist)
func (s State) IsDead() bool {
	return s&INSTANCE_STATE_DEAD == INSTANCE_STATE_DEAD
}

// IsIdle returns true if instance is idle
func (s State) IsIdle() bool {
	return s&INSTANCE_STATE_IDLE == INSTANCE_STATE_IDLE
}

// IsSyncing returns true if instance syncing with master/slave
func (s State) IsSyncing() bool {
	return s&INSTANCE_STATE_SYNCING == INSTANCE_STATE_SYNCING
}

// IsLoading returns true if instance loading rds into memory
func (s State) IsLoading() bool {
	return s&INSTANCE_STATE_LOADING == INSTANCE_STATE_LOADING
}

// IsSaving returns true if instance saving data on disk
func (s State) IsSaving() bool {
	return s&INSTANCE_STATE_SAVING == INSTANCE_STATE_SAVING
}

// IsHang returns true if instance blocked by some command
func (s State) IsHang() bool {
	return s&INSTANCE_STATE_HANG == INSTANCE_STATE_HANG
}

// IsAbandoned returns true if instance abandoned (no traffic for long time)
func (s State) IsAbandoned() bool {
	return s&INSTANCE_STATE_ABANDONED == INSTANCE_STATE_ABANDONED
}

// IsMasterUp returns true if instance master is up
func (s State) IsMasterUp() bool {
	return s&INSTANCE_STATE_MASTER_UP == INSTANCE_STATE_MASTER_UP
}

// IsMasterDown returns true if instance master is down
func (s State) IsMasterDown() bool {
	return s&INSTANCE_STATE_MASTER_DOWN == INSTANCE_STATE_MASTER_DOWN
}

func (s State) NoReplica() bool {
	return s&INSTANCE_STATE_NO_REPLICA == INSTANCE_STATE_NO_REPLICA
}

func (s State) WithReplica() bool {
	return s&INSTANCE_STATE_WITH_REPLICA == INSTANCE_STATE_WITH_REPLICA
}

func (s State) WithErrors() bool {
	return s&INSTANCE_STATE_WITH_ERRORS == INSTANCE_STATE_WITH_ERRORS
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Init starts initialization routine
func Init(conf string) []error {
	var err error

	User, err = system.CurrentUser()

	if err != nil {
		return []error{fmt.Errorf("Can't get current user info: %w", err)}
	}

	if User.UID != 0 {
		return []error{ErrUnprivileged}
	}

	globalConfig = conf
	Config, err = knf.Read(globalConfig)

	if err != nil {
		return []error{err}
	}

	errs := validateConfig(Config)

	if len(errs) != 0 {
		return errs
	}

	errs = validateDependencies()

	if len(errs) != 0 {
		return errs
	}

	metaCache = NewMetaCache(5 * time.Second)

	pid.Dir = Config.GetS(PATH_PID_DIR)

	return nil
}

// ReloadConfig reloads RDS configuration
func ReloadConfig() []error {
	newConfig, err := knf.Read(globalConfig)

	if err != nil {
		return []error{err}
	}

	errs := validateConfig(newConfig)

	if len(errs) != 0 {
		return errs
	}

	Config = newConfig

	return nil
}

// SetLogOutput setup log output
func SetLogOutput(file, minLevel string, bufIO bool) error {
	err := log.Set(path.Join(Config.GetS(PATH_LOG_DIR), file), 0644)

	if err != nil {
		return err
	}

	if bufIO {
		log.EnableBufIO(250 * time.Millisecond)
	}

	return log.MinLevel(minLevel)
}

// ReopenLog reopen log (close+open) file for log rotation purposes
func ReopenLog() error {
	return log.Reopen()
}

// GenerateToken generates unique token for sync daemon
func GenerateToken() string {
	return passwd.GenPassword(TOKEN_LENGTH, passwd.STRENGTH_MEDIUM)
}

// NewSUAuth generates new superuser auth data
func NewSUAuth() (string, *SuperuserAuth, error) {
	password := GenPassword()
	pepper := passwd.GenPassword(32, passwd.STRENGTH_MEDIUM)
	hash, err := passwd.Hash(password, pepper)

	if err != nil {
		return "", nil, fmt.Errorf("Can't generate encrypted password hash: %w", err)
	}

	return password, &SuperuserAuth{Hash: hash, Pepper: pepper}, nil
}

// HasSUAuth returns true if superuser auth data exists
func HasSUAuth() bool {
	return fsutil.CheckPerms("FRS", path.Join(Config.GetS(MAIN_DIR), SU_DATA_FILE))
}

// SaveSUAuth save superuser auth data
func SaveSUAuth(auth *SuperuserAuth, rewrite bool) error {
	if HasSUAuth() && !rewrite {
		return ErrSUAuthAlreadyExist
	}

	if auth == nil || auth.Hash == "" || auth.Pepper == "" {
		return ErrSUAuthIsEmpty
	}

	authFile := path.Join(Config.GetS(MAIN_DIR), SU_DATA_FILE)

	if fsutil.IsExist(authFile) {
		err := os.Remove(authFile)

		if err != nil {
			return err
		}
	}

	return jsonutil.Write(authFile, auth, DEFAULT_FILE_PERMS)
}

// ReadSUAuth read superuser auth data
func ReadSUAuth() (*SuperuserAuth, error) {
	if !HasSUAuth() {
		return nil, ErrSUAuthNoData
	}

	auth := &SuperuserAuth{}
	err := jsonutil.Read(path.Join(Config.GetS(MAIN_DIR), SU_DATA_FILE), auth)

	return auth, err
}

// ValidateTemplates validates templates for Redis and Sentinel
func ValidateTemplates() []error {
	var errs errors.Bundle

	meta, err := NewInstanceMeta("test", "test")

	if err != nil {
		errs.Add(fmt.Errorf("Can't generate instance meta for validation: %w", err))
	} else {
		_, err = generateConfigFromTemplate(
			TEMPLATE_SOURCE_REDIS,
			createConfigFromMeta(meta),
		)

		errs.Add(err)
	}

	_, err = generateConfigFromTemplate(
		TEMPLATE_SOURCE_SENTINEL,
		&sentinelConfigData{},
	)

	errs.Add(err)

	return errs.All()
}

// HasInstances returns true if that at least one instance exists
func HasInstances() bool {
	return !fsutil.IsEmptyDir(Config.GetS(PATH_META_DIR))
}

// GetInstanceMetaFilePath returns path to meta file for instance with given ID
func GetInstanceMetaFilePath(id int) string {
	return path.Join(Config.GetS(PATH_META_DIR), strconv.Itoa(id))
}

// GetInstanceConfigFilePath returns path to config file for instance with given ID
func GetInstanceConfigFilePath(id int) string {
	return path.Join(Config.GetS(PATH_CONFIG_DIR), strconv.Itoa(id)+".conf")
}

// GetInstanceDataDirPath returns path to data directory for instance with given ID
func GetInstanceDataDirPath(id int) string {
	return path.Join(Config.GetS(PATH_DATA_DIR), strconv.Itoa(id))
}

// GetInstanceLogDirPath returns path to logs directory for instance with given ID
func GetInstanceLogDirPath(id int) string {
	return path.Join(Config.GetS(PATH_LOG_DIR), strconv.Itoa(id))
}

// GetInstanceLogFilePath returns path to log file for instance with given ID
func GetInstanceLogFilePath(id int) string {
	return path.Join(Config.GetS(PATH_LOG_DIR), strconv.Itoa(id), "redis.log")
}

// GetInstancePIDFilePath returns path to PID file for instance with given ID
func GetInstancePIDFilePath(id int) string {
	return path.Join(Config.GetS(PATH_PID_DIR), strconv.Itoa(id)+".pid")
}

// GetStatesFilePath returns path to global states file
func GetStatesFilePath() string {
	return path.Join(Config.GetS(MAIN_DIR), STATES_DATA_FILE)
}

// GetInstancePort returns port used by redis for given instance
func GetInstancePort(id int) int {
	return Config.GetI(REDIS_START_PORT) + id
}

// IsMaster returns true if role of current RDS node has role "master"
func IsMaster() bool {
	return Config.GetS(REPLICATION_ROLE) == ROLE_MASTER
}

// IsMinion returns true if role of current RDS node has role "minion"
func IsMinion() bool {
	return Config.GetS(REPLICATION_ROLE) == ROLE_MINION
}

// IsSentinel returns true if role of current RDS node has role "sentinel"
func IsSentinel() bool {
	return Config.GetS(REPLICATION_ROLE) == ROLE_SENTINEL
}

// IsInstanceExist returns true if instance exists
func IsInstanceExist(id int) bool {
	if id < 1 {
		return false
	}

	return fsutil.IsExist(GetInstanceMetaFilePath(id))
}

// HasInstanceData returns true if instance data is present on FS
func HasInstanceData(id int) bool {
	if !IsInstanceExist(id) {
		return false
	}

	if !fsutil.IsExist(GetInstanceDataDirPath(id)) {
		return false
	}

	if !fsutil.IsExist(GetInstanceConfigFilePath(id)) {
		return false
	}

	return true
}

// GetInstanceMeta returns meta info struct for given instance
func GetInstanceMeta(id int) (*InstanceMeta, error) {
	var meta *InstanceMeta

	if !IsInstanceExist(id) {
		return nil, fmt.Errorf("Instance with ID %d doesn't exist", id)
	}

	if metaCache == nil {
		metaCache = &MetaCache{}
	}

	meta, ok := metaCache.Get(id)

	if ok {
		return meta, nil
	}

	meta = &InstanceMeta{}
	err := jsonutil.Read(GetInstanceMetaFilePath(id), meta)

	if err != nil {
		return nil, fmt.Errorf("Error while instance meta data decoding: %v", err)
	}

	if meta.MetaVersion != META_VERSION {
		return nil, fmt.Errorf(
			"Unsupported meta version used (%d). Please migrate meta to latest supported version (%d).",
			meta.MetaVersion, META_VERSION,
		)
	}

	metaCache.Set(id, meta)

	meta, _ = metaCache.Get(id)

	return meta, err
}

// GetAvailableInstanceID returns available instance ID
func GetAvailableInstanceID() int {
	if Config.GetB(MAIN_ALLOW_ID_REUSE) {
		return getFreeInstanceID()
	}

	return getUnusedInstanceID()
}

// GetInstanceState returns state of instance
func GetInstanceState(id int, extended bool) (State, error) {
	if !IsInstanceExist(id) {
		return INSTANCE_STATE_UNKNOWN, fmt.Errorf("Instance with ID %d doesn't exist", id)
	}

	pidFile := GetInstancePIDFilePath(id)

	if fsutil.IsExist(pidFile) {
		pid := GetInstancePID(id)

		if pid == -1 {
			return INSTANCE_STATE_UNKNOWN, fmt.Errorf("Can't read PID from file %s", pidFile)
		}

		pr, err := os.FindProcess(pid)

		if err != nil {
			return INSTANCE_STATE_DEAD, nil
		}

		err = pr.Signal(syscall.Signal(0))

		if err != nil {
			return INSTANCE_STATE_DEAD, nil
		}

		if extended {
			return INSTANCE_STATE_WORKS | getInstanceExtendedState(id), nil
		}

		return INSTANCE_STATE_WORKS, nil
	}

	return INSTANCE_STATE_STOPPED, nil
}

// GetInstanceVersion return instance Redis version
func GetInstanceVersion(id int) version.Version {
	if !IsInstanceExist(id) {
		return version.Version{}
	}

	state, err := GetInstanceState(id, false)

	if err != nil {
		return version.Version{}
	}

	if !state.IsWorks() {
		currentRedisVer, _ := GetRedisVersion()
		return currentRedisVer
	}

	info, err := GetInstanceInfo(id, time.Second, false)

	if err != nil {
		return version.Version{}
	}

	instanceVer, _ := version.Parse(info.Get("server", "redis_version"))

	return instanceVer
}

// GetInstanceStartDate returns timestamp when instance was started
func GetInstanceStartDate(id int) int64 {
	if !IsInstanceExist(id) {
		return -1
	}

	pidFile := GetInstancePIDFilePath(id)

	if !fsutil.IsExist(pidFile) {
		return -1
	}

	ctime, err := fsutil.GetCTime(pidFile)

	if err != nil {
		return -1
	}

	return ctime.Unix()
}

// GetInstanceConfigHash returns Redis config file SHA-256 hash
func GetInstanceConfigHash(id int) (string, error) {
	if !IsInstanceExist(id) {
		return "", fmt.Errorf("Instance with ID %d doesn't exist", id)
	}

	confFile := GetInstanceConfigFilePath(id)

	if !fsutil.IsExist(confFile) {
		return "", fmt.Errorf("Configuration file for instance %d doesn't exist", id)
	}

	cf, err := os.Open(confFile)

	if err != nil {
		return "", fmt.Errorf("Can't read configuration file for instance %d (%v)", id, err)
	}

	defer cf.Close()

	hasher := sha256.New()
	_, err = io.Copy(hasher, cf)

	if err != nil {
		return "", fmt.Errorf("Can't calculate configuration file hash for instance %d (%v)", id, err)
	}

	return fmt.Sprintf("%064x", hasher.Sum(nil)), nil
}

// NewAuthInfo generates new instance auth struct
func NewInstanceAuth(password string) (*InstanceAuth, error) {
	pepper := passwd.GenPassword(32, passwd.STRENGTH_MEDIUM)
	hash, err := passwd.Hash(password, pepper)

	if err != nil {
		return nil, fmt.Errorf("Can't generate encrypted password hash: %w", err)
	}

	return &InstanceAuth{
		User:   User.RealName,
		Hash:   hash,
		Pepper: pepper,
	}, nil
}

// NewInstanceMeta generates meta struct for new instance
func NewInstanceMeta(instancePassword, servicePassword string) (*InstanceMeta, error) {
	id := GetAvailableInstanceID()

	if id == -1 {
		return nil, errors.New("No available ID for usage")
	}

	auth, err := NewInstanceAuth(instancePassword)

	if err != nil {
		return nil, err
	}

	preferencies := &InstancePreferencies{
		ServicePassword:  servicePassword,
		AdminPassword:    GenPassword(),
		SyncPassword:     GenPassword(),
		SentinelPassword: GenPassword(),
		ReplicationType:  ReplicationType(knf.GetS(REPLICATION_DEFAULT_ROLE, string(REPL_TYPE_REPLICA))),
	}

	return &InstanceMeta{
		ID:           id,
		MetaVersion:  META_VERSION,
		UUID:         uuid.UUID4().String(),
		Preferencies: preferencies,
		Auth:         auth,
		Config:       &InstanceConfigInfo{},
		Storage:      make(map[string]string),
	}, nil
}

// CreateInstance create instance by instance meta
func CreateInstance(meta *InstanceMeta) error {
	err := meta.Validate()

	if err != nil {
		return fmt.Errorf("Meta is not valid: %v", err)
	}

	if IsInstanceExist(meta.ID) {
		return fmt.Errorf("Instance with ID %d already exist", meta.ID)
	}

	if len(meta.Desc) > MAX_DESC_LENGTH {
		meta.Desc = meta.Desc[0:MAX_DESC_LENGTH]
	}

	meta.Created = time.Now().Unix()

	if !IsSentinel() {
		err = createInstanceData(meta)

		if err != nil {
			return err
		}
	}

	err = saveInstanceMeta(meta)

	if err != nil {
		return err
	}

	metaCache.Set(meta.ID, meta)

	if !Config.GetB(MAIN_ALLOW_ID_REUSE) {
		err = updateIDSInfo(meta.ID)

		if err != nil {
			return err
		}
	}

	return nil
}

// RegenerateInstanceConfig regenerate redis config file for given instance
func RegenerateInstanceConfig(id int) error {
	var err error

	if !IsInstanceExist(id) {
		return fmt.Errorf("Instance with ID %d doesn't exist", id)
	}

	meta, err := GetInstanceMeta(id)

	if err != nil {
		return err
	}

	redisUser, err := system.LookupUser(Config.GetS(REDIS_USER))

	if err != nil {
		return err
	}

	err = createInstanceConfig(meta)

	if err != nil {
		return err
	}

	errs := errors.NewBundle().Add(
		os.Chown(GetInstanceLogDirPath(id), redisUser.UID, redisUser.GID),
		os.Chown(GetInstanceDataDirPath(id), redisUser.UID, redisUser.GID),
		os.Chown(GetInstanceConfigFilePath(id), redisUser.UID, redisUser.GID),
	)

	if fsutil.IsExist(GetInstanceLogFilePath(id)) {
		errs.Add(os.Chown(GetInstanceLogFilePath(id), redisUser.UID, redisUser.GID))
	}

	if !errs.IsEmpty() {
		return errs.Last()
	}

	metaCache.Set(id, meta)

	return nil
}

// UpdateInstance update instance meta
// For security purposes only some part of fields from given meta is used for update
func UpdateInstance(newMeta *InstanceMeta) error {
	var err error
	var hasChanges bool

	if !IsInstanceExist(newMeta.ID) {
		return fmt.Errorf("Instance with ID %d doesn't exist", newMeta.ID)
	}

	oldMeta, err := GetInstanceMeta(newMeta.ID)

	if err != nil {
		return err
	}

	if oldMeta.Preferencies.ReplicationType != newMeta.Preferencies.ReplicationType {
		if !newMeta.Preferencies.ReplicationType.IsReplica() &&
			!newMeta.Preferencies.ReplicationType.IsStandby() {
			return ErrUnknownReplicationType
		}

		oldMeta.Preferencies.ReplicationType = newMeta.Preferencies.ReplicationType

		hasChanges = true
	}

	if newMeta.Auth != nil {
		if newMeta.Auth.Pepper != "" && newMeta.Auth.Hash != "" {
			if newMeta.Auth.Pepper != oldMeta.Auth.Pepper &&
				newMeta.Auth.Hash != oldMeta.Auth.Hash {

				oldMeta.Auth.Pepper = newMeta.Auth.Pepper
				oldMeta.Auth.Hash = newMeta.Auth.Hash

				hasChanges = true
			}
		}

		if newMeta.Auth.User != "" && newMeta.Auth.User != oldMeta.Auth.User {
			oldMeta.Auth.User = newMeta.Auth.User
			hasChanges = true
		}
	}

	if newMeta.Desc != "" && newMeta.Desc != oldMeta.Desc {
		if len(newMeta.Desc) > MAX_DESC_LENGTH {
			oldMeta.Desc = newMeta.Desc[0:MAX_DESC_LENGTH]
		} else {
			oldMeta.Desc = newMeta.Desc
		}

		hasChanges = true
	}

	if strings.Join(newMeta.Tags, ":") != strings.Join(oldMeta.Tags, ":") {
		oldMeta.Tags = newMeta.Tags
		hasChanges = true
	}

	if !hasChanges {
		return nil
	}

	err = saveInstanceMeta(oldMeta)

	if err != nil {
		return err
	}

	metaCache.Set(oldMeta.ID, oldMeta)

	return nil
}

// GetInstancePID returns PID of given instance
func GetInstancePID(id int) int {
	if !IsInstanceExist(id) {
		return -1
	}

	return pid.Get(strconv.Itoa(id))
}

// GetInstanceIDList returns sorted slice with all instance ID's
func GetInstanceIDList() []int {
	var result []int

	metaFileList := fsutil.List(Config.GetS(PATH_META_DIR), true)

	if len(metaFileList) == 0 {
		return result
	}

	for _, metaFile := range metaFileList {
		idInt, err := strconv.Atoi(metaFile)

		if err == nil {
			result = append(result, idInt)
		}
	}

	sort.Ints(result)

	return result
}

// GetInstanceIDListByState returns sorted slice with instance ID's filtered by state
func GetInstanceIDListByState(state State) ([]int, error) {
	var result []int

	for _, id := range GetInstanceIDList() {
		instanceState, err := GetInstanceState(id, isExtendedState(state))

		if err != nil {
			return nil, err
		}

		if instanceState&state > 0 {
			result = append(result, id)
		}
	}

	return result, nil
}

// ReadInstanceConf read and parse redis config for given instance
func ReadInstanceConfig(id int) (*REDIS.Config, error) {
	if !IsInstanceExist(id) {
		return nil, fmt.Errorf("Instance with ID %d doesn't exist", id)
	}

	return REDIS.ReadConfig(GetInstanceConfigFilePath(id))
}

// GetInstanceConfig read and parse instance in-memory config
func GetInstanceConfig(id int, timeout time.Duration) (*REDIS.Config, error) {
	if !IsInstanceExist(id) {
		return nil, fmt.Errorf("Instance with ID %d doesn't exist", id)
	}

	meta, err := GetInstanceMeta(id)

	if err != nil {
		return nil, err
	}

	return REDIS.GetConfig(
		&REDIS.Request{
			Command: []string{"CONFIG", "GET", "*"},
			Port:    GetInstancePort(id),
			Auth:    REDIS.Auth{REDIS_USER_ADMIN, meta.Preferencies.AdminPassword},
			Timeout: timeout,
		},
	)
}

// GetInstanceConfigChanges returns difference between file config
// and in-memory config
func GetInstanceConfigChanges(id int) ([]REDIS.ConfigPropDiff, error) {
	if !IsInstanceExist(id) {
		return nil, fmt.Errorf("Instance with ID %d doesn't exist", id)
	}

	state, err := GetInstanceState(id, false)

	if err != nil {
		return nil, err
	}

	if !state.IsWorks() {
		return nil, ErrCantDiffConfigs
	}

	fileConfig, err := ReadInstanceConfig(id)

	if err != nil {
		return nil, err
	}

	memConfig, err := GetInstanceConfig(id, 3*time.Second)

	if err != nil {
		return nil, err
	}

	return REDIS.GetConfigsDiff(fileConfig, memConfig), nil
}

// ReloadInstanceConfig reload instance config
func ReloadInstanceConfig(id int) []error {
	diff, err := GetInstanceConfigChanges(id)

	if err != nil {
		return []error{err}
	}

	if len(diff) == 0 {
		return nil
	}

	return applyChangedConfigProps(id, diff)
}

// GetInstanceRDBPath returns path to the instance dump file
func GetInstanceRDBPath(id int) string {
	dataDir := GetInstanceDataDirPath(id)
	defaultRDB := path.Join(dataDir, "dump.rdb")

	if fsutil.CheckPerms("FS", defaultRDB) {
		return defaultRDB
	}

	config, err := ReadInstanceConfig(id)

	if err != nil {
		return defaultRDB
	}

	rdb := config.Get("dbfilename")

	if rdb == "" {
		return defaultRDB
	}

	return path.Join(dataDir, rdb)
}

// GetInstanceAOFPath returns path to the append only file
func GetInstanceAOFPath(id int) string {
	dataDir := GetInstanceDataDirPath(id)
	config, err := ReadInstanceConfig(id)

	// Return default path
	if err != nil {
		return path.Join(dataDir, "appendonly.aof")
	}

	aof := config.Get("dbfilename")

	if aof == "" {
		return path.Join(dataDir, "appendonly.aof")
	}

	return path.Join(dataDir, aof)
}

// GetInstanceInfo returns info from instance
func GetInstanceInfo(id int, timeout time.Duration, all bool) (*REDIS.Info, error) {
	var info *REDIS.Info

	if !IsInstanceExist(id) {
		return info, fmt.Errorf("Instance with ID %d doesn't exist", id)
	}

	meta, err := GetInstanceMeta(id)

	if err != nil {
		return nil, err
	}

	command := []string{"INFO"}

	if all {
		command = append(command, "all")
	}

	return REDIS.GetInfo(
		&REDIS.Request{
			Command: command,
			Port:    GetInstancePort(id),
			Auth:    REDIS.Auth{REDIS_USER_ADMIN, meta.Preferencies.AdminPassword},
			Timeout: timeout,
		},
	)
}

// ExecCommand executes Redis command on given instance
func ExecCommand(id int, req *REDIS.Request) (*REDIS.Resp, error) {
	if !IsInstanceExist(id) {
		return nil, fmt.Errorf("Instance with ID %d doesn't exist", id)
	}

	if req.Port == 0 {
		req.Port = GetInstancePort(id)
	}

	if req.Auth.User == "" {
		meta, err := GetInstanceMeta(id)

		if err != nil {
			return nil, fmt.Errorf("Can't read instance meta: %v", err)
		}

		req.Auth = REDIS.Auth{"admin", meta.Preferencies.AdminPassword}
	}

	resp, err := REDIS.ExecCommand(req)

	if err != nil {
		return nil, fmt.Errorf("Can't execute command: %v", err)
	}

	return resp, nil
}

// ParseIDDBPair parse ID/DB pair (id:db id/db)
func ParseIDDBPair(pair string) (int, int, error) {
	if pair == "" {
		return -1, -1, ErrEmptyIDDBPair
	}

	if !strings.ContainsAny(pair, ":/") {
		id, err := strconv.Atoi(pair)

		if err != nil {
			return -1, -1, fmt.Errorf("Can't parse ID \"%s\"", pair)
		}

		return id, 0, nil
	}

	idStr := strutil.ReadField(pair, 0, false, ':', '/')
	id, err := strconv.Atoi(idStr)

	if err != nil {
		return -1, -1, fmt.Errorf("Can't parse value \"%s\" as ID", idStr)
	}

	dbStr := strutil.ReadField(pair, 1, false, ':', '/')
	db, err := strconv.Atoi(dbStr)

	if err != nil {
		return -1, -1, fmt.Errorf("Can't parse value \"%s\" as DB number", dbStr)
	}

	return id, db, nil
}

// StartInstance starting instance
// If controlLoading set to true, instance marked as started only
// after starting, finishing loading and syncing with master
func StartInstance(id int, controlLoading bool) error {
	if !IsInstanceExist(id) {
		return fmt.Errorf("Instance with ID %d doesn't exist", id)
	}

	err := runAsUser(
		Config.GetS(REDIS_USER),
		GetInstanceLogFilePath(id),
		Config.GetS(REDIS_BINARY),
		GetInstanceConfigFilePath(id),
		"--daemonize", "yes", // Always daemonize server
	)

	if err != nil {
		return err
	}

	if !isProcStarted(strconv.Itoa(id), Config.GetI(DELAY_START)) {
		return fmt.Errorf("Instance PID file %s was not created", GetInstancePIDFilePath(id))
	}

	err = configureSchedulers(id)

	if err != nil {
		return err
	}

	if controlLoading {
		isInstanceFullyStarted(id)
	}

	err = updateCompatibilityInfo(id)

	if err != nil {
		return err
	}

	err = updateConfigInfo(id)

	if err != nil {
		return err
	}

	if IsSentinelActive() {
		meta, err := GetInstanceMeta(id)

		if err != nil {
			return err
		}

		// Sentinel monitoring works only with replicas
		if meta.Preferencies.ReplicationType.IsReplica() {
			err = SentinelStartMonitoring(id)

			if err != nil {
				return fmt.Errorf("Can't start Sentinel monitoring: %w", err)
			}
		}
	}

	return nil
}

// StopInstance stopping instance
func StopInstance(id int, force bool) error {
	var err error

	if !IsInstanceExist(id) {
		return fmt.Errorf("Instance with ID %d doesn't exist", id)
	}

	instancePID := GetInstancePID(id)

	if instancePID == -1 {
		return ErrCantReadPID
	}

	if IsSentinelActive() && IsSentinelMonitors(id) {
		err = SentinelStopMonitoring(id)

		if err != nil {
			return err
		}
	}

	err = execShutdownCommand(id)

	if err != nil {
		return err
	}

	cmdStart := time.Now()
	stopDelay := time.Second * time.Duration(Config.GetI(DELAY_STOP))

	for range time.NewTicker(time.Second).C {
		if pid.IsProcessWorks(instancePID) {
			if isInstanceSavingData(id) {
				time.Sleep(3 * time.Second)
				cmdStart = time.Now()
				continue
			}

			if time.Since(cmdStart) > stopDelay {
				if force {
					syscall.Kill(instancePID, syscall.SIGKILL)
					return nil
				}

				return ErrInstanceStillWorks
			}

			continue
		}

		return nil
	}

	return nil
}

// KillInstance send KILL signal to Redis
func KillInstance(id int) error {
	if !IsInstanceExist(id) {
		return fmt.Errorf("Instance with ID %d doesn't exist", id)
	}

	pid := GetInstancePID(id)

	if pid == -1 {
		return ErrCantReadPID
	}

	syscall.Kill(pid, syscall.SIGKILL)

	pidFile := GetInstancePIDFilePath(id)

	if fsutil.IsExist(pidFile) {
		err := os.RemoveAll(pidFile)

		if err != nil {
			return fmt.Errorf("Can't remove PID file: %v", err)
		}
	}

	return nil
}

// DestroyInstance destroy (delete) instance
func DestroyInstance(id int) error {
	var err error

	if !IsInstanceExist(id) {
		return fmt.Errorf("Instance with ID %d doesn't exist", id)
	}

	if IsSentinelMonitors(id) {
		err = SentinelStopMonitoring(id)

		if err != nil {
			return err
		}
	}

	if !IsSentinel() {
		state, err := GetInstanceState(id, false)

		if err != nil {
			return fmt.Errorf("Can't get instance state: %v", err)
		}

		if state.IsWorks() {
			err = KillInstance(id)

			if err != nil {
				return fmt.Errorf("Can't stop instance: %v", err)
			}
		}

		err = os.RemoveAll(GetInstanceConfigFilePath(id))

		if err != nil {
			return fmt.Errorf("Can't remove configuration file: %v", err)
		}

		err = os.RemoveAll(GetInstanceLogDirPath(id))

		if err != nil {
			return fmt.Errorf("Can't remove log directory: %v", err)
		}

		err = os.RemoveAll(GetInstanceDataDirPath(id))

		if err != nil {
			return fmt.Errorf("Can't remove data directory: %v", err)
		}

		pidFile := GetInstancePIDFilePath(id)

		if fsutil.IsExist(pidFile) {
			err = os.RemoveAll(pidFile)

			if err != nil {
				return fmt.Errorf("Can't remove PID file: %v", err)
			}
		}
	}

	err = os.RemoveAll(GetInstanceMetaFilePath(id))

	if err != nil {
		return fmt.Errorf("Can't remove meta file: %v", err)
	}

	return nil
}

// SentinelStart start (run) Sentinel daemon
func SentinelStart() []error {
	if !IsSentinel() && !IsFailoverMethod(FAILOVER_METHOD_SENTINEL) {
		return []error{ErrIncompatibleFailover}
	}

	if IsSentinelActive() {
		return nil
	}

	currentRedisVer, err := GetRedisVersion()

	if err != nil {
		return []error{fmt.Errorf("Can't get Redis Sentinel version: %w", err)}
	}

	if currentRedisVer.String() == "" || currentRedisVer.Major() < MIN_SENTINEL_VERSION {
		return []error{ErrSentinelWrongVersion}
	}

	sentinelConfigDir := path.Join(Config.GetS(PATH_CONFIG_DIR), "sentinel")

	if !fsutil.IsExist(sentinelConfigDir) {
		err = createSentinelConfigDir(sentinelConfigDir)

		if err != nil {
			return []error{fmt.Errorf("Can't create directory for Sentinel configuration: %w", err)}
		}
	}

	sentinelConfig := path.Join(sentinelConfigDir, "sentinel.conf")

	err = generateSentinelConfig()

	if err != nil {
		return []error{err}
	}

	sentinelLogFile := path.Join(Config.GetS(PATH_LOG_DIR), "sentinel.log")

	err = runAsUser(
		Config.GetS(REDIS_USER),
		sentinelLogFile,
		Config.GetS(SENTINEL_BINARY),
		sentinelConfig,
		"--daemonize", "yes", // Always daemonize server
	)

	if err != nil {
		return []error{err}
	}

	if !isProcStarted(PID_SENTINEL, Config.GetI(DELAY_START)) {
		return []error{ErrSentinelCantStart}
	}

	return addAllReplicasToSentinelMonitoring()
}

// SentinelStop stop (shutdown) Sentinel daemon
func SentinelStop() error {
	if !IsSentinel() && !IsFailoverMethod(FAILOVER_METHOD_SENTINEL) {
		return ErrIncompatibleFailover
	}

	if !IsSentinelActive() {
		return nil
	}

	sentinelPID := pid.Get(PID_SENTINEL)

	if sentinelPID == -1 {
		sentinelPIDFile := path.Join(Config.GetS(PATH_PID_DIR), PID_SENTINEL)
		return fmt.Errorf("Can't read PID from PID file %s", sentinelPIDFile)
	}

	err := syscall.Kill(sentinelPID, syscall.SIGTERM)

	if err != nil {
		return err
	}

	if !isProcStopped(sentinelPID, Config.GetI(DELAY_STOP)) {
		return ErrSentinelCantStop
	}

	return os.RemoveAll(path.Join(Config.GetS(PATH_CONFIG_DIR), "sentinel"))
}

// SentinelCheck returns message about checking Sentinel quorum status
func SentinelCheck(id int) (string, bool) {
	if !IsFailoverMethod(FAILOVER_METHOD_SENTINEL) {
		return ErrIncompatibleFailover.Error(), false
	}

	if !IsSentinelActive() {
		return "Sentinel is stopped", false
	}

	sCfg := &SENTINEL.SentinelConfig{
		Port: Config.GetI(SENTINEL_PORT),
	}

	return SENTINEL.CheckQuorum(sCfg, id)
}

// SentinelReset reset master state in Sentinel for all instances
func SentinelReset() error {
	if !IsSentinel() && !IsFailoverMethod(FAILOVER_METHOD_SENTINEL) {
		return ErrIncompatibleFailover
	}

	if !IsSentinelActive() {
		return ErrSentinelIsStopped
	}

	sCfg := &SENTINEL.SentinelConfig{
		Port: Config.GetI(SENTINEL_PORT),
	}

	return SENTINEL.Reset(sCfg)
}

// SentinelStartMonitoring add instance to Sentinel monitoring
func SentinelStartMonitoring(id int) error {
	if !IsFailoverMethod(FAILOVER_METHOD_SENTINEL) {
		return ErrIncompatibleFailover
	}

	if !IsSentinelActive() {
		return ErrSentinelIsStopped
	}

	if IsSentinelMonitors(id) {
		return nil
	}

	meta, err := GetInstanceMeta(id)

	if err != nil {
		return err
	}

	sCfg := &SENTINEL.SentinelConfig{
		Port: Config.GetI(SENTINEL_PORT),
	}

	iCfg := &SENTINEL.InstanceConfig{
		ID:   id,
		IP:   Config.GetS(REPLICATION_MASTER_IP, netutil.GetIP()),
		Port: GetInstancePort(id),

		Auth: SENTINEL.Auth{REDIS_USER_SENTINEL, meta.Preferencies.SentinelPassword},

		Quorum:                Config.GetI(SENTINEL_QUORUM, 3),
		DownAfterMilliseconds: Config.GetI(SENTINEL_DOWN_AFTER, 10000),
		FailoverTimeout:       Config.GetI(SENTINEL_FAILOVER_TIMEOUT, 180000),
		ParallelSyncs:         Config.GetI(SENTINEL_PARALLEL_SYNCS, 1),
	}

	return SENTINEL.Monitor(sCfg, iCfg)
}

// SentinelStopMonitoring remove instance from Sentinel monitoring
func SentinelStopMonitoring(id int) error {
	if !IsFailoverMethod(FAILOVER_METHOD_SENTINEL) {
		return ErrIncompatibleFailover
	}

	if !IsSentinelActive() {
		return errors.New("Sentinel is stopped")
	}

	if !IsSentinelMonitors(id) {
		return nil
	}

	sCfg := &SENTINEL.SentinelConfig{
		Port: Config.GetI(SENTINEL_PORT),
	}

	return SENTINEL.Remove(sCfg, id)
}

// SentinelMasterSwitch can be used if you want to set role of current
// local instance to master. This command temporary set slave priority to
// 1 and force failover.
func SentinelMasterSwitch(id int) error {
	if !IsFailoverMethod(FAILOVER_METHOD_SENTINEL) {
		return ErrIncompatibleFailover
	}

	if !IsInstanceExist(id) {
		return fmt.Errorf("Instance with ID %d doesn't exist", id)
	}

	if !IsSentinelActive() {
		return ErrSentinelIsStopped
	}

	if !IsMaster() {
		return ErrSentinelRoleSetNotAllowed
	}

	state, err := GetInstanceState(id, false)

	if err != nil {
		return fmt.Errorf("Can't get state of instance: %v", err)
	}

	if !state.IsWorks() {
		return fmt.Errorf("Can't switch Sentinel master to instance %d - instance doesn't work", id)
	}

	info, err := GetInstanceInfo(id, time.Second, false)

	if err != nil {
		return fmt.Errorf("Can't get info about instance %d: %v", id, err)
	}

	if info.Get("replication", "role") != "slave" {
		return ErrSentinelWrongInstanceRole
	}

	priority := info.Get("replication", "slave_priority")

	if priority == "" {
		priority = "100" // Default priority
	}

	return runSentinelFailoverSwitch(id, priority)
}

// SentinelMasterIP returns IP of master instance
func SentinelMasterIP(id int) (string, error) {
	sCfg := &SENTINEL.SentinelConfig{
		Port: Config.GetI(SENTINEL_PORT),
	}

	return SENTINEL.GetMasterIP(sCfg, id)
}

// SentinelInfo returns info from Sentinel about master, replicas and sentinels
func SentinelInfo(id int) (*SENTINEL.Info, error) {
	sCfg := &SENTINEL.SentinelConfig{
		Port: Config.GetI(SENTINEL_PORT),
	}

	return SENTINEL.GetInfo(sCfg, id)
}

// IsSentinelMonitors returns true if Sentinel monitoring instance
// with given ID
func IsSentinelMonitors(id int) bool {
	sCfg := &SENTINEL.SentinelConfig{
		Port: Config.GetI(SENTINEL_PORT),
	}

	return SENTINEL.IsSentinelMonitors(sCfg, id)
}

// SaveStates save all instances states to file
func SaveStates(file string) error {
	if file == "" {
		return ErrStateFileNotDefined
	}

	if fsutil.IsExist(file) {
		err := os.Remove(file)

		if err != nil {
			return err
		}
	}

	idList := GetInstanceIDList()

	if len(idList) == 0 {
		return nil
	}

	statesInfo := &StatesInfo{
		States:   make([]StateInfo, 0),
		Sentinel: IsSentinelActive(),
	}

	for _, id := range idList {
		state, err := GetInstanceState(id, false)

		if err != nil {
			continue
		}

		if state.IsDead() {
			statesInfo.States = append(statesInfo.States, StateInfo{id, INSTANCE_STATE_WORKS})
		} else {
			statesInfo.States = append(statesInfo.States, StateInfo{id, state})
		}
	}

	return jsonutil.Write(file, statesInfo, DEFAULT_FILE_PERMS)
}

// ReadStates read states info from file
func ReadStates(file string) (*StatesInfo, error) {
	if file == "" {
		return nil, ErrStateFileNotDefined
	}

	if !fsutil.CheckPerms("FRS", file) {
		return nil, fmt.Errorf("States file %s doesn't exist or empty", file)
	}

	statesInfo := &StatesInfo{}
	err := jsonutil.Read(file, statesInfo)

	if err != nil {
		return nil, err
	}

	return statesInfo, nil
}

// AddTag adds tag to instance
func AddTag(id int, tag string) error {
	if !IsInstanceExist(id) {
		return fmt.Errorf("Instance with ID %d doesn't exist", id)
	}

	if !IsValidTag(tag) {
		return fmt.Errorf("Tag %s has the wrong format", tag)
	}

	meta, err := GetInstanceMeta(id)

	if err != nil {
		return err
	}

	tagName, _ := ParseTag(tag)

	for _, currentTag := range meta.Tags {
		currentTagName, _ := ParseTag(currentTag)

		if strings.EqualFold(tagName, currentTagName) {
			return fmt.Errorf("Tag \"%s\" already exist", tagName)
		}
	}

	if len(meta.Tags) >= MAX_TAGS {
		return fmt.Errorf("Max number of tags (%d) reached", MAX_TAGS)
	}

	meta.Tags = append(meta.Tags, tag)

	metaCache.Set(meta.ID, meta)

	return saveInstanceMeta(meta)
}

// RemoveTag remove tag associated with instance
func RemoveTag(id int, tag string) error {
	if !IsInstanceExist(id) {
		return fmt.Errorf("Instance with ID %d doesn't exist", id)
	}

	if !IsValidTag(tag) {
		return fmt.Errorf("Tag %s has the wrong format", tag)
	}

	meta, err := GetInstanceMeta(id)

	if err != nil {
		return err
	}

	var tags []string
	var hasChanges bool

	for _, currentTag := range meta.Tags {
		tagName, _ := ParseTag(currentTag)

		if strings.EqualFold(tagName, tag) {
			hasChanges = true
			continue
		}

		tags = append(tags, currentTag)
	}

	if !hasChanges {
		return fmt.Errorf("Tag \"%s\" doesn't exist", tag)
	}

	meta.Tags = tags

	metaCache.Set(meta.ID, meta)

	return saveInstanceMeta(meta)
}

// IsValidTag validates tag
func IsValidTag(tag string) bool {
	tagName, tagColor := ParseTag(tag)

	switch tagColor {
	case "r", "g", "y", "b", "m", "c", "s", "":
		// skip
	default:
		return false
	}

	return tagRegex.MatchString(tagName)
}

// ParseTag parse tag and returns tag and color
func ParseTag(tag string) (string, string) {
	if strings.Contains(tag, ":") {
		return strutil.ReadField(tag, 1, false, ':'),
			strutil.ReadField(tag, 0, false, ':')
	}

	return tag, ""
}

// IsSyncDaemonActive returns true if sync daemon is works
func IsSyncDaemonActive() bool {
	isWorks, _ := initsystem.IsWorks("rds-sync.service")
	return isWorks
}

// IsSyncDaemonInstalled returns true if sync daemon is installed
func IsSyncDaemonInstalled() bool {
	return env.Which("rds-sync") != ""
}

// IsSentinelActive returns true if Sentinel is works
func IsSentinelActive() bool {
	return pid.IsWorks(PID_SENTINEL)
}

// IsFailoverMethod returns true if given failover method is used
func IsFailoverMethod(method FailoverMethod) bool {
	return FailoverMethod(Config.GetS(REPLICATION_FAILOVER_METHOD)) == method
}

// IsOutdated returns true if instance outdated
func IsOutdated(id int) bool {
	meta, err := GetInstanceMeta(id)

	if err != nil {
		return false
	}

	currentRedisVer, err := GetRedisVersion()

	if err == nil && currentRedisVer.String() != "" && meta.Compatible != "" {
		return meta.Compatible != currentRedisVer.String()
	}

	return false
}

// GetSystemConfigurationStatus returns system configuration status
func GetSystemConfigurationStatus(force bool) (SystemStatus, error) {
	status := SystemStatus{}

	if !force && (Config.GetB(MAIN_DISABLE_CONFIGURATION_CHECK, false) || IsSentinel()) {
		return status, nil
	}

	var err error

	status.HasTHPIssues, err = isSystemHasTHPIssues()

	if err != nil {
		return SystemStatus{}, err
	}

	status.HasKernelIssues, err = isSystemHasKernelIssues()

	if err != nil {
		return SystemStatus{}, err
	}

	status.HasLimitsIssues, err = isSystemHasLimitsIssues()

	if err != nil {
		return SystemStatus{}, err
	}

	status.HasFSIssues, err = isSystemHasFSIssues(force)

	if err != nil {
		return SystemStatus{}, err
	}

	status.HasProblems = status.HasTHPIssues || status.HasKernelIssues || status.HasLimitsIssues || status.HasFSIssues

	return status, nil
}

// GetRedisVersion returns current installed Redis version
func GetRedisVersion() (version.Version, error) {
	if !redisVersion.IsZero() {
		return redisVersion, nil
	}

	var err error
	var info *RedisVersionInfo

	versionCacheFile := path.Join(Config.GetS(MAIN_DIR), REDIS_VERSION_DATA_FILE)
	cacheExist := fsutil.IsExist(versionCacheFile)

	if cacheExist {
		info, err = getRedisVersionFromCache(versionCacheFile)

		if err != nil {
			cacheExist = false
		}
	}

	if info == nil {
		info, err = getRedisVersionFromBinary()

		if err != nil {
			return version.Version{}, err
		}
	}

	redisVersion, err = version.Parse(info.Version)

	if err != nil {
		return version.Version{}, err
	}

	if !cacheExist {
		jsonutil.Write(versionCacheFile, info, DEFAULT_FILE_PERMS)
	}

	return redisVersion, nil
}

// GetStats returns overall stats
func GetStats() *Stats {
	var stats = &Stats{
		Instances: &StatsInstances{},
		Clients:   &StatsClients{},
		Memory:    &StatsMemory{},
		Overall:   &StatsOverall{},
		Keys:      &StatsKeys{},
	}

	var mappings = map[string]*uint64{
		"Clients:connected_clients":        &stats.Clients.Connected,
		"Clients:blocked_clients":          &stats.Clients.Blocked,
		"Memory:used_memory":               &stats.Memory.UsedMemory,
		"Memory:used_memory_rss":           &stats.Memory.UsedMemoryRSS,
		"Memory:used_memory_lua":           &stats.Memory.UsedMemoryLua,
		"Stats:total_connections_received": &stats.Overall.TotalConnectionsReceived,
		"Stats:total_commands_processed":   &stats.Overall.TotalCommandsProcessed,
		"Stats:instantaneous_ops_per_sec":  &stats.Overall.InstantaneousOpsPerSec,
		"Stats:instantaneous_input_kbps":   &stats.Overall.InstantaneousInputKbps,
		"Stats:instantaneous_output_kbps":  &stats.Overall.InstantaneousOutputKbps,
		"Stats:rejected_connections":       &stats.Overall.RejectedConnections,
		"Stats:expired_keys":               &stats.Overall.ExpiredKeys,
		"Stats:evicted_keys":               &stats.Overall.EvictedKeys,
		"Stats:keyspace_hits":              &stats.Overall.KeyspaceHits,
		"Stats:keyspace_misses":            &stats.Overall.KeyspaceMisses,
		"Stats:pubsub_channels":            &stats.Overall.PubsubChannels,
		"Stats:pubsub_patterns":            &stats.Overall.PubsubPatterns,
	}

	memUsage, _ := system.GetMemUsage()

	stats.Memory.TotalSystemMemory = memUsage.MemTotal
	stats.Memory.SystemMemory = memUsage.MemUsed
	stats.Memory.TotalSystemSwap = memUsage.SwapTotal
	stats.Memory.SystemSwap = memUsage.SwapUsed
	stats.Memory.IsSwapEnabled = memUsage.SwapTotal != 0

	if !HasInstances() {
		return stats
	}

	for _, id := range GetInstanceIDList() {
		stats.Instances.Total++

		state, err := GetInstanceState(id, false)

		if state.IsDead() {
			stats.Instances.Dead++
			continue
		}

		if !state.IsWorks() || err != nil {
			continue
		}

		if IsOutdated(id) {
			stats.Instances.Outdated++
		}

		hwm, rss, swap := getMemoryUsageFromProcFS(id)

		stats.Memory.UsedSwap += swap

		info, err := GetInstanceInfo(id, time.Second, false)

		if err != nil {
			stats.Memory.UsedMemory += hwm
			stats.Memory.UsedMemoryRSS += rss
			continue
		}

		if info.Get("persistence", "rdb_bgsave_in_progress") != "0" {
			stats.Instances.BgSave++
		}

		if info.Get("persistence", "aof_rewrite_in_progress") != "0" {
			stats.Instances.AOFRewrite++
		}

		if info.Get("persistence", "rdb_last_bgsave_status") != "ok" ||
			info.Get("persistence", "aof_last_bgrewrite_status") != "ok" ||
			info.Get("persistence", "aof_last_write_status") != "ok" {
			stats.Instances.SaveFailed++
		}

		if info.Get("replication", "connected_slaves") != "0" {
			stats.Instances.ActiveMaster++
		}

		if info.Get("replication", "master_link_status") == "up" {
			stats.Instances.ActiveReplica++
		}

		if info.Get("replication", "master_sync_in_progress") == "1" {
			stats.Instances.Syncing++
		}

		stats.Instances.Active++

		stats.Keys.Total += info.Keyspace.Keys()
		stats.Keys.Expires += info.Keyspace.Expires()

		for k, v := range mappings {
			appendStatsData(info, k, v)
		}

		// Remove our connections from stats
		stats.Clients.Connected--
	}

	return stats
}

// GetKeepalivedState returns state of keepalived virtual IP
func GetKeepalivedState() KeepalivedState {
	virtualIP := Config.GetS(KEEPALIVED_VIRTUAL_IP)

	if virtualIP == "" {
		return KEEPALIVED_STATE_UNKNOWN
	}

	addrs, err := net.InterfaceAddrs()

	if err != nil {
		return KEEPALIVED_STATE_UNKNOWN
	}

	for _, addr := range addrs {
		if addr.String() == virtualIP+"/32" {
			return KEEPALIVED_STATE_MASTER
		}
	}

	return KEEPALIVED_STATE_BACKUP
}

// GenPassword generates secure password with random length (16-28)
func GenPassword() string {
	return passwd.GenPassword(16+rand.Intn(6), passwd.STRENGTH_MEDIUM)
}

// Shutdown safely shutdown app
func Shutdown(code int) {
	log.Flush()
	os.Exit(code)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// IsStandby returns true if replication type is standby
func (t ReplicationType) IsStandby() bool {
	return t == REPL_TYPE_STANDBY
}

// IsReplica returns true if replication type is replica
func (t ReplicationType) IsReplica() bool {
	return t == REPL_TYPE_REPLICA
}

// Validate validate meta struct
func (m *InstanceMeta) Validate() error {
	switch {
	case m == nil:
		return ErrMetaIsNil

	case m.Auth == nil:
		return ErrMetaNoAuth

	case m.Preferencies == nil:
		return ErrMetaNoPrefs

	case m.Config == nil:
		return ErrMetaNoConfigInfo

	case m.MetaVersion < 1 || m.MetaVersion > META_VERSION:
		return ErrMetaInvalidVersion

	case m.ID < 1:
		return ErrMetaNoID

	case m.Desc == "":
		return ErrMetaNoDesc
	}

	return nil
}

// Port returns instance port
func (p *instanceConfigData) Port() int {
	return GetInstancePort(p.ID)
}

// MetaFile returns path to meta file for instance with given ID
func (p *instanceConfigData) MetaFile() string {
	return GetInstanceMetaFilePath(p.ID)
}

// ConfigFile returns path to config file for instance with given ID
func (p *instanceConfigData) ConfigFile() string {
	return GetInstanceConfigFilePath(p.ID)
}

// DataDir returns path to data directory for instance with given ID
func (p *instanceConfigData) DataDir() string {
	return GetInstanceDataDirPath(p.ID)
}

// LogDir returns path to logs directory for instance with given ID
func (p *instanceConfigData) LogDir() string {
	return GetInstanceLogDirPath(p.ID)
}

// LogFile returns path to log file for instance with given ID
func (p *instanceConfigData) LogFile() string {
	return GetInstanceLogFilePath(p.ID)
}

// PidFile returns path to PID file for instance with given ID
func (p *instanceConfigData) PidFile() string {
	return GetInstancePIDFilePath(p.ID)
}

// MasterHost returns redis master host (IP)
func (p *instanceConfigData) MasterHost() string {
	return Config.GetS(REPLICATION_MASTER_IP)
}

// MasterPort returns redis master port
func (p *instanceConfigData) MasterPort() int {
	return GetInstancePort(p.ID)
}

// HasTag return true if configuration has given tag
func (c *instanceConfigData) HasTag(tag string) bool {
	if len(c.tags) == 0 {
		return false
	}

	for _, t := range c.tags {
		instanceTag, _ := ParseTag(t)

		if strings.EqualFold(instanceTag, tag) {
			return true
		}
	}

	return false
}

// Storage returns value from instance custom data storage
func (c *instanceConfigData) Storage(key string) string {
	if c.storage == nil {
		return ""
	}

	return c.storage[key]
}

// Version returns struct with Redis version info
func (c *instanceConfigData) Version() version.Version {
	return c.Redis
}

// RedisVersionLess returns true if instance Redis version is less than given
func (c *instanceConfigData) RedisVersionLess(v string) bool {
	ver, err := version.Parse(v)

	if err != nil {
		return false
	}

	return c.Redis.Less(ver)
}

// RedisVersionGreater returns true if instance Redis version is greater than given
func (c *instanceConfigData) RedisVersionGreater(v string) bool {
	ver, err := version.Parse(v)

	if err != nil {
		return false
	}

	return c.Redis.Greater(ver)
}

// RedisVersionEquals returns true if instance Redis version is equal to given
func (c *instanceConfigData) RedisVersionEquals(v string) bool {
	ver, err := version.Parse(v)

	if err != nil {
		return false
	}

	return c.Redis.Equal(ver)
}

// AdminPasswordHash returns SHA-256 hash for admin user password
func (c *instanceConfigData) AdminPasswordHash() string {
	return getSHA256Hash(c.AdminPassword)
}

// SyncPasswordHash returns SHA-256 hash for sync user password
func (c *instanceConfigData) SyncPasswordHash() string {
	return getSHA256Hash(c.SyncPassword)
}

// SentinelPasswordHash returns SHA-256 hash for sentinel user password
func (c *instanceConfigData) SentinelPasswordHash() string {
	return getSHA256Hash(c.SentinelPassword)
}

// ServicePasswordHash returns SHA-256 hash for service user password
func (c *instanceConfigData) ServicePasswordHash() string {
	return getSHA256Hash(c.ServicePassword)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// validateDependencies checks dependencies
func validateDependencies() []error {
	var err error
	var errs []error

	if !fsutil.IsExist(Config.GetS(REDIS_BINARY)) {
		errs = append(errs, fmt.Errorf(
			"Redis is not installed (missing binary %s)",
			Config.GetS(REDIS_BINARY)),
		)
	} else {
		err = fsutil.ValidatePerms("FRX", Config.GetS(REDIS_BINARY))

		if err != nil {
			errs = append(errs, fmt.Errorf("Wrong permissions on Redis binary: %w", err))
		}
	}

	if !fsutil.IsExist(Config.GetS(SENTINEL_BINARY)) {
		errs = append(errs, fmt.Errorf(
			"Sentinel is not installed (missing binary %s)",
			Config.GetS(SENTINEL_BINARY)),
		)
	} else {
		err = fsutil.ValidatePerms("FRX", Config.GetS(SENTINEL_BINARY))

		if err != nil {
			errs = append(errs, fmt.Errorf("Wrong permissions on Sentinel binary: %w", err))
		}
	}

	if len(errs) == 0 {
		currentRedisVer, err := GetRedisVersion()

		if err != nil {
			errs = append(errs, fmt.Errorf("Can't get Redis version: %w", err))
		} else {
			majorRedisVer := fmt.Sprintf("%d.%d", currentRedisVer.Major(), currentRedisVer.Minor())

			if !supportedVersions[majorRedisVer] {
				errs = append(errs, fmt.Errorf("Redis %s is not supported", currentRedisVer))
			}
		}
	}

	return errs
}

// validateConfig validate config values
func validateConfig(c *knf.Config) []error {
	validators := []*knf.Validator{
		{MAIN_MAX_INSTANCES, knfv.Set, nil},
		{MAIN_WARN_USED_MEMORY, knfv.Set, nil},
		{MAIN_MIN_PASS_LENGTH, knfv.Set, nil},

		// MAIN //

		{MAIN_MAX_INSTANCES, knfv.Greater, MIN_INSTANCES},
		{MAIN_MAX_INSTANCES, knfv.Less, MAX_INSTANCES},
		{MAIN_WARN_USED_MEMORY, knfv.Greater, 0},
		{MAIN_WARN_USED_MEMORY, knfv.Less, 100},
		{MAIN_MIN_PASS_LENGTH, knfv.Greater, MIN_PASS_LENGTH},
		{MAIN_MIN_PASS_LENGTH, knfv.Less, MAX_PASS_LENGTH},

		// DELAY //

		{DELAY_START, knfv.Set, nil},
		{DELAY_STOP, knfv.Set, nil},
		{DELAY_STOP, knfv.Greater, MIN_STOP_DELAY},
		{DELAY_STOP, knfv.Less, MAX_STOP_DELAY},
		{DELAY_START, knfv.Greater, MIN_START_DELAY},
		{DELAY_START, knfv.Less, MAX_START_DELAY},

		// REDIS //

		{REDIS_BINARY, knfv.Set, nil},
		{REDIS_USER, knfv.Set, nil},
		{REDIS_USER, knfs.User, nil},
		{REDIS_START_PORT, knfv.Set, nil},
		{REDIS_START_PORT, knfn.Port, nil},
		{REDIS_START_PORT, knfv.Greater, MIN_PORT},
		{REDIS_START_PORT, knfv.Less, MAX_PORT},
		{REDIS_NICE, knfv.Greater, MIN_NICE},
		{REDIS_NICE, knfv.Less, MAX_NICE},
		{REDIS_IONICE_CLASS, knfv.Greater, MIN_IONICE_CLASS},
		{REDIS_IONICE_CLASS, knfv.Less, MAX_IONICE_CLASS},
		{REDIS_IONICE_CLASSDATA, knfv.Greater, MIN_IONICE_CLASSDATA},
		{REDIS_IONICE_CLASSDATA, knfv.Less, MAX_IONICE_CLASSDATA},

		// SENTINEL //

		{SENTINEL_BINARY, knfv.Set, nil},
		{SENTINEL_QUORUM, knfv.Set, nil},
		{SENTINEL_DOWN_AFTER, knfv.Set, nil},
		{SENTINEL_FAILOVER_TIMEOUT, knfv.Set, nil},
		{SENTINEL_PARALLEL_SYNCS, knfv.Set, nil},
		{SENTINEL_PORT, knfn.Port, nil},
		{SENTINEL_PORT, knfv.Greater, MIN_PORT},
		{SENTINEL_PORT, knfv.Less, MAX_PORT},

		// KEEPALIVED

		{KEEPALIVED_VIRTUAL_IP, knfn.IP, nil},

		// TEMPLATES //

		{TEMPLATES_REDIS, knfv.Set, nil},
		{TEMPLATES_SENTINEL, knfv.Set, nil},
		{TEMPLATES_REDIS, knff.Perms, "DRX"},
		{TEMPLATES_SENTINEL, knff.Perms, "DRX"},

		// PATHS //

		{PATH_META_DIR, knfv.Set, nil},
		{PATH_CONFIG_DIR, knfv.Set, nil},
		{PATH_DATA_DIR, knfv.Set, nil},
		{PATH_PID_DIR, knfv.Set, nil},
		{PATH_LOG_DIR, knfv.Set, nil},
		{PATH_META_DIR, knff.Perms, "DRX"},
		{PATH_CONFIG_DIR, knff.Perms, "DRX"},
		{PATH_DATA_DIR, knff.Perms, "DRX"},
		{PATH_PID_DIR, knff.Perms, "DRX"},
		{PATH_LOG_DIR, knff.Perms, "DRX"},
		{PATH_PID_DIR, knff.Owner, Config.GetS(REDIS_USER, "redis")},

		// LOG //

		{LOG_LEVEL, knfv.SetToAnyIgnoreCase, []string{
			"", "debug", "info", "warn", "error", "crit",
		}},
	}

	// REPLICATION //

	if c.GetS(REPLICATION_ROLE) != "" {
		validators = append(validators,
			&knf.Validator{REPLICATION_MASTER_IP, knfn.IP, nil},
			&knf.Validator{REPLICATION_MASTER_PORT, knfv.Set, nil},
			&knf.Validator{REPLICATION_MASTER_PORT, knfv.Greater, MIN_PORT},
			&knf.Validator{REPLICATION_MASTER_PORT, knfv.Less, MAX_PORT},
			&knf.Validator{REPLICATION_AUTH_TOKEN, knfv.LenEquals, TOKEN_LENGTH},
			&knf.Validator{REPLICATION_MAX_SYNC_WAIT, knfv.Greater, MIN_SYNC_WAIT},
			&knf.Validator{REPLICATION_MAX_SYNC_WAIT, knfv.Less, MAX_SYNC_WAIT},
			&knf.Validator{REPLICATION_FAILOVER_METHOD, knfv.SetToAny, []string{
				string(FAILOVER_METHOD_STANDBY), string(FAILOVER_METHOD_SENTINEL),
			}},
			&knf.Validator{REPLICATION_DEFAULT_ROLE, knfv.SetToAny, []string{
				string(REPL_TYPE_STANDBY), string(REPL_TYPE_REPLICA),
			}},
			&knf.Validator{REPLICATION_ROLE, knfv.SetToAny, []string{
				"", ROLE_MASTER, ROLE_MINION, ROLE_SENTINEL,
			}},
		)
	}

	return c.Validate(validators)
}

// createInstanceData create all required files and directories for instance
func createInstanceData(meta *InstanceMeta) error {
	var err error

	redisUser, err := system.LookupUser(Config.GetS(REDIS_USER))

	if err != nil {
		return err
	}

	var (
		logDir   = GetInstanceLogDirPath(meta.ID)
		logFile  = GetInstanceLogFilePath(meta.ID)
		dataDir  = GetInstanceDataDirPath(meta.ID)
		confFile = GetInstanceConfigFilePath(meta.ID)
	)

	err = os.MkdirAll(logDir, 0755)

	if err != nil {
		return err
	}

	err = os.MkdirAll(dataDir, 0755)

	if err != nil {
		return err
	}

	log, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE, 0644)

	if err != nil {
		return err
	}

	log.Close()

	err = createInstanceConfig(meta)

	if err != nil {
		return err
	}

	return errors.NewBundle().Add(
		os.Chown(logDir, redisUser.UID, redisUser.GID),
		os.Chown(dataDir, redisUser.UID, redisUser.GID),
		os.Chown(logFile, redisUser.UID, redisUser.GID),
		os.Chown(confFile, redisUser.UID, redisUser.GID),
	).Last()
}

// saveInstanceMeta save meta data to file
func saveInstanceMeta(meta *InstanceMeta) error {
	return jsonutil.Write(GetInstanceMetaFilePath(meta.ID), meta, DEFAULT_FILE_PERMS)
}

// createInstanceConfig create redis config from template
func createInstanceConfig(meta *InstanceMeta) error {
	cfg := createConfigFromMeta(meta)
	confData, err := generateConfigFromTemplate(TEMPLATE_SOURCE_REDIS, cfg)

	if err != nil {
		return err
	}

	err = os.WriteFile(GetInstanceConfigFilePath(meta.ID), confData, 0640)

	if err != nil {
		return err
	}

	hasher := sha256.New()
	hasher.Write(confData)

	meta.Config.Hash = fmt.Sprintf("%064x", hasher.Sum(nil))
	meta.Config.Date = time.Now().Unix()

	return nil
}

// getFreeInstanceID returns first free instance ID or -1
func getFreeInstanceID() int {
	metaDir := Config.GetS(PATH_META_DIR)

	for id := 1; id <= Config.GetI(MAIN_MAX_INSTANCES); id++ {
		if !fsutil.IsExist(path.Join(metaDir, strconv.Itoa(id))) {
			return id
		}
	}

	return -1
}

// getUnusedInstanceID returns last unused instance ID or -1
func getUnusedInstanceID() int {
	ids := GetInstanceIDList()

	if len(ids) >= Config.GetI(MAIN_MAX_INSTANCES) {
		return -1
	}

	dataFile := path.Join(Config.GetS(MAIN_DIR), IDS_DATA_FILE)

	if fsutil.IsExist(dataFile) {
		info := &IDSInfo{}

		if jsonutil.Read(dataFile, info) != nil {
			return -1
		}

		return info.LastID + 1
	}

	if len(ids) == 0 {
		return 1
	}

	return ids[len(ids)-1] + 1
}

// updateIDSInfo update info about latest used ID
func updateIDSInfo(id int) error {
	info := &IDSInfo{id}
	dataFile := path.Join(Config.GetS(MAIN_DIR), IDS_DATA_FILE)

	return jsonutil.Write(dataFile, info, DEFAULT_FILE_PERMS)
}

// getInstanceExtendedState returns extended instance state
func getInstanceExtendedState(id int) State {
	if isInstanceSavingData(id) {
		return INSTANCE_STATE_SAVING
	}

	info, err := GetInstanceInfo(id, time.Second, true)

	if err != nil {
		return INSTANCE_STATE_HANG
	}

	var state State

	if info.GetI("stats", "instantaneous_ops_per_sec") < 5 {
		state |= INSTANCE_STATE_IDLE

		if !info.Is("persistence", "rdb_last_save_time", 0) &&
			info.Is("persistence", "rdb_changes_since_last_save", 0) &&
			int64(info.GetI("persistence", "rdb_last_save_time")) < time.Now().Unix()-(7*24*3600) {
			state |= INSTANCE_STATE_ABANDONED
		}
	}

	if info.Is("replication", "master_sync_in_progress", true) {
		state |= INSTANCE_STATE_SYNCING
	}

	if info.Is("persistence", "rdb_bgsave_in_progress", true) ||
		info.Is("persistence", "aof_rewrite_in_progress", true) {
		state |= INSTANCE_STATE_SAVING
	}

	if info.Is("persistence", "loading", true) ||
		info.Is("persistence", "async_loading", true) {
		state |= INSTANCE_STATE_LOADING
	}

	if info.Is("replication", "connected_slaves", 0) &&
		info.Is("replication", "connected_replicas", 0) {
		state |= INSTANCE_STATE_NO_REPLICA
	} else {
		state |= INSTANCE_STATE_WITH_REPLICA
	}

	if info.Is("replication", "master_link_status", "up") {
		state |= INSTANCE_STATE_MASTER_UP
	}

	if info.Is("replication", "master_link_status", "down") {
		state |= INSTANCE_STATE_MASTER_DOWN
	}

	if info.Sections["errorstats"] != nil && len(info.Sections["errorstats"].Fields) > 0 {
		state |= INSTANCE_STATE_WITH_ERRORS
	}

	return state
}

// isInstanceSavingData try to find if instance currently performs a synchronous save
func isInstanceSavingData(id int) bool {
	tempRDB := fmt.Sprintf(
		"%s/temp-%d.rdb",
		GetInstanceDataDirPath(id),
		GetInstancePID(id),
	)

	now := time.Now()

	if fsutil.IsExist(tempRDB) {
		mtime, err := fsutil.GetMTime(tempRDB)

		if err != nil {
			return false
		}

		if now.Unix()-mtime.Unix() < 5 {
			return true
		}
	}

	return false
}

// getMemoryUsageFromProcFS returns mem usage from procfs
func getMemoryUsageFromProcFS(id int) (uint64, uint64, uint64) {
	memInfo, err := process.GetMemInfo(GetInstancePID(id))

	if err != nil {
		return 0, 0, 0
	}

	return memInfo.VmHWM, memInfo.VmRSS, memInfo.VmSwap
}

// getRedisVersionFromCache read current redis version info from cache
func getRedisVersionFromCache(cacheFile string) (*RedisVersionInfo, error) {
	info := &RedisVersionInfo{}
	err := jsonutil.Read(cacheFile, info)

	if err != nil {
		return nil, err
	}

	binary := Config.GetS(REDIS_BINARY)
	cDate, err := fsutil.GetCTime(binary)

	if err != nil {
		return nil, err
	}

	if cDate.Unix() != info.CDate {
		return nil, ErrInvalidRedisVersionCache
	}

	return info, nil
}

// getRedisVersionFromBinary read redis version from redis version info output
func getRedisVersionFromBinary() (*RedisVersionInfo, error) {
	binary := Config.GetS(REDIS_BINARY)

	if !fsutil.IsExecutable(binary) {
		return nil, fmt.Errorf("File %s is not an executable binary", binary)
	}

	out, err := exec.Command(binary, "-v").Output()

	if err != nil {
		return nil, err
	}

	verStr := strutil.ReadField(string(out), 2, false, ' ')

	if verStr == "" || !strings.HasPrefix(verStr, "v=") {
		return nil, ErrCantParseRedisVersion
	}

	cDate, err := fsutil.GetCTime(binary)

	if err != nil {
		return nil, ErrCantReadRedisCreationDate
	}

	return &RedisVersionInfo{
		Version: strutil.Substr(verStr, 2, 99),
		CDate:   cDate.Unix(),
	}, nil
}

// updateCompatibilityInfo update compatible redis version info in instance meta
func updateCompatibilityInfo(id int) error {
	meta, err := GetInstanceMeta(id)

	if err != nil {
		return err
	}

	redisVersion, err := GetRedisVersion()

	if err != nil {
		return fmt.Errorf("Can't update instance compatibility info: %w", err)
	}

	switch {
	case err != nil:
		return err
	case meta.Compatible == redisVersion.String():
		return nil
	case redisVersion.String() == "":
		return nil
	}

	meta.Compatible = redisVersion.String()

	metaCache.Set(meta.ID, meta)

	return saveInstanceMeta(meta)
}

// updateConfigInfo update config hash and modification date in instance meta
func updateConfigInfo(id int) error {
	hash, err := GetInstanceConfigHash(id)

	if err != nil {
		return err
	}

	meta, err := GetInstanceMeta(id)

	if err != nil {
		return err
	}

	mtime, err := fsutil.GetMTime(GetInstanceConfigFilePath(id))

	if err != nil {
		return err
	}

	meta.Config.Date = mtime.Unix()
	meta.Config.Hash = hash

	metaCache.Set(meta.ID, meta)

	return saveInstanceMeta(meta)
}

// createSentinelConfigDir creates directory for Sentinel configuration
func createSentinelConfigDir(dir string) error {
	redisUser, err := system.LookupUser(Config.GetS(REDIS_USER))

	if err != nil {
		return err
	}

	err = os.Mkdir(dir, 0770)

	if err != nil {
		return err
	}

	return os.Chown(dir, redisUser.UID, redisUser.GID)
}

// generateSentinelConfig creates and generates Sentinel configuration
func generateSentinelConfig() error {
	var err error

	sentinelConfig := path.Join(Config.GetS(PATH_CONFIG_DIR), "sentinel", "sentinel.conf")
	sentinelPidFile := path.Join(Config.GetS(PATH_PID_DIR), PID_SENTINEL)
	sentinelLogFile := path.Join(Config.GetS(PATH_LOG_DIR), "sentinel.log")
	sentinelPort := Config.GetS(SENTINEL_PORT, "63999")

	// Rewrite configuration
	if fsutil.IsExist(sentinelConfig) {
		err = os.Remove(sentinelConfig)

		if err != nil {
			return err
		}
	}

	redisUser, err := system.LookupUser(Config.GetS(REDIS_USER))

	if err != nil {
		return err
	}

	confData, err := generateConfigFromTemplate(
		TEMPLATE_SOURCE_SENTINEL,
		&sentinelConfigData{
			Port:    sentinelPort,
			PidFile: sentinelPidFile,
			LogFile: sentinelLogFile,
		},
	)

	if err != nil {
		return err
	}

	err = os.WriteFile(sentinelConfig, confData, 0640)

	if err != nil {
		return err
	}

	if !fsutil.IsExist(sentinelLogFile) {
		err = fsutil.TouchFile(sentinelLogFile, 0640)

		if err != nil {
			return err
		}
	}

	return errors.NewBundle().Add(
		os.Chown(sentinelConfig, redisUser.UID, redisUser.GID),
		os.Chown(sentinelLogFile, redisUser.UID, redisUser.GID),
	).Last()
}

// getConfigTemplateData reads configuration data from template
// for currently installed Redis/Sentinel version
func getConfigTemplateData(source TemplateSource) (string, string, error) {
	var err error
	var templateFile, templateFilePath string

	currentRedisVer, err := GetRedisVersion()

	if err != nil {
		return "", "", fmt.Errorf("Can't get Redis version: %w", err)
	}

	majorRedisVer := fmt.Sprintf("%d.%d", currentRedisVer.Major(), currentRedisVer.Minor())

	switch source {
	case TEMPLATE_SOURCE_REDIS:
		templateFile = "redis-" + majorRedisVer + ".conf"
		templateFilePath, err = path.JoinSecure(Config.GetS(TEMPLATES_REDIS), templateFile)
	case TEMPLATE_SOURCE_SENTINEL:
		templateFile = "sentinel-" + majorRedisVer + ".conf"
		templateFilePath, err = path.JoinSecure(Config.GetS(TEMPLATES_SENTINEL), templateFile)
	default:
		return "", "", ErrUnknownTemplateSource
	}

	if err != nil {
		return "", "", fmt.Errorf("Can't create path to configuration template: %w", err)
	}

	data, err := os.ReadFile(templateFilePath)

	if err != nil {
		return "", "", fmt.Errorf("Can't read configuration template data: %w", err)
	}

	return templateFile, string(data), nil
}

// generateConfigFromTemplate generates configuration from template
func generateConfigFromTemplate(source TemplateSource, data any) ([]byte, error) {
	templateFile, templateData, err := getConfigTemplateData(source)

	if err != nil {
		return nil, err
	}

	t, err := template.New(templateFile).Parse(templateData)

	if err != nil {
		return nil, fmt.Errorf("Can't parse template data: %w", err)
	}

	var bf bytes.Buffer

	err = t.Execute(&bf, data)

	if err != nil {
		return nil, fmt.Errorf("Can't render template data: %w", err)
	}

	return bf.Bytes(), nil
}

// addAllReplicasToSentinelMonitoring adds all relicas to Sentinel monitoring
func addAllReplicasToSentinelMonitoring() []error {
	if !HasInstances() {
		return nil
	}

	var errs []error

	for _, id := range GetInstanceIDList() {
		state, err := GetInstanceState(id, false)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		if !state.IsWorks() {
			continue
		}

		meta, err := GetInstanceMeta(id)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		if !meta.Preferencies.ReplicationType.IsReplica() {
			continue
		}

		err = SentinelStartMonitoring(id)

		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

// isProcStarted returns true if PID file exist and process with PID from
// this file is works
func isProcStarted(pidFile string, delay int) bool {
	cmdStart := time.Now()
	delaySec := time.Second * time.Duration(delay)

	for range time.NewTicker(time.Second).C {
		if pid.IsWorks(pidFile) {
			return true
		}

		if time.Since(cmdStart) >= delaySec {
			break
		}
	}

	return false
}

// isProcStopped returns true if process with PID from PID file is not
// works
func isProcStopped(appPID, delay int) bool {
	cmdStart := time.Now()
	delaySec := time.Second * time.Duration(delay)

	for range time.NewTicker(time.Second).C {
		if !pid.IsProcessWorks(appPID) {
			return true
		}

		if time.Since(cmdStart) >= delaySec {
			break
		}
	}

	return false
}

// isInstanceFullyStarted if instance complete loading and syncing
func isInstanceFullyStarted(id int) bool {
	for i := 0; i < MAX_FULL_START_DELAY; i++ {
		state, err := GetInstanceState(id, true)

		if err == nil {
			if !state.IsLoading() && !state.IsSyncing() {
				return true
			}
		}

		time.Sleep(5 * time.Second)
	}

	return false
}

// runAsUser run binary as defined user
func runAsUser(user, logFile string, args ...string) error {
	if !fsutil.IsRegular(BIN_RUNUSER) {
		return fmt.Errorf("%s is not found on this system", BIN_RUNUSER)
	}

	if !fsutil.IsExecutable(BIN_RUNUSER) {
		return fmt.Errorf("%s is not executable", BIN_RUNUSER)
	}

	if !fsutil.IsExist(logFile) {
		return fmt.Errorf("Log file %s doesn't exist", logFile)
	}

	logFd, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	defer logFd.Close()

	oldUMask := syscall.Umask(027)
	defer syscall.Umask(oldUMask)

	w := bufio.NewWriter(logFd)
	cmdArgs := []string{"-s", "/bin/bash", user, "-c", strings.Join(args, " ")}
	cmd := exec.Command(BIN_RUNUSER, cmdArgs...)

	cmd.Stderr = w

	err = cmd.Run()

	w.Flush()

	if err != nil {
		return fmt.Errorf("Error while executing binary. Details have been saved to log file %s", logFile)
	}

	return nil
}

// configureSchedulers configures CPU and IO schedulers
func configureSchedulers(id int) error {
	instancePID := GetInstancePID(id)

	if instancePID == -1 {
		return fmt.Errorf("Can't get instance PID for instance with ID %d", id)
	}

	err := configureCPUScheduler(id, instancePID)

	if err != nil {
		return err
	}

	return configureIOScheduler(id, instancePID)
}

// configureCPUScheduler configures CPU scheduler (nice)
func configureCPUScheduler(id, instancePID int) error {
	if Config.GetI(REDIS_NICE) == 0 {
		return nil
	}

	pids, err := getRedisTreePIDs(id, instancePID)

	if err != nil {
		return err
	}

	niceness := Config.GetI(REDIS_NICE)

	for _, ppid := range pids {
		err = process.SetCPUPriority(ppid, niceness)

		if err != nil {
			return fmt.Errorf("Can't set CPU priority for instance with ID %d: %w", id, err)
		}
	}

	return nil
}

// configureIOScheduler configures IO scheduler (ionice)
func configureIOScheduler(id, instancePID int) error {
	if Config.GetI(REDIS_IONICE_CLASS) == 0 && Config.GetI(REDIS_IONICE_CLASSDATA) == 0 {
		return nil
	}

	pids, err := getRedisTreePIDs(id, instancePID)

	if err != nil {
		return err
	}

	class := Config.GetI(REDIS_IONICE_CLASS)
	classdata := Config.GetI(REDIS_IONICE_CLASSDATA)

	for _, ppid := range pids {
		err = process.SetIOPriority(ppid, class, classdata)

		if err != nil {
			return fmt.Errorf("Can't set IO priority for instance with ID %d: %w", id, err)
		}
	}

	return nil
}

// getRedisTreePIDs returns all PIDs of Redis instance
func getRedisTreePIDs(id, instancePID int) ([]int, error) {
	tree, err := process.GetTree(instancePID)

	if err != nil {
		return nil, fmt.Errorf("Can't find Redis instance PIDs for instance with ID %d: %w", id, err)
	}

	pids := []int{instancePID}

	for _, proc := range tree.Children {
		if strings.Contains(proc.Command, "redis-server") {
			pids = append(pids, proc.PID)
		}
	}

	return pids, nil
}

// runSentinelFailoverSwitch run failover on Sentinel for switching master
func runSentinelFailoverSwitch(id int, priority string) error {
	meta, err := GetInstanceMeta(id)

	if err != nil {
		return err
	}

	req := &REDIS.Request{
		Port:    GetInstancePort(id),
		Auth:    REDIS.Auth{REDIS_USER_ADMIN, meta.Preferencies.AdminPassword},
		Timeout: time.Second,
	}

	// Set instance priority to 1 (will be preferred by Sentinel)
	req.Command = []string{"CONFIG", "SET", "slave-priority", "1"}

	_, err = REDIS.ExecCommand(req)

	if err != nil {
		return fmt.Errorf("Can't set slave priority: %v", err)
	}

	sCfg := &SENTINEL.SentinelConfig{
		Port: Config.GetI(SENTINEL_PORT),
	}

	// Force Sentinel failover
	sentinelErr := SENTINEL.Failover(sCfg, id)

	if sentinelErr == nil {
		roleSet := false

		for sec := 0; sec < MAX_SWITCH_WAIT; sec++ {
			info, _ := GetInstanceInfo(id, time.Second, false)

			if info.Get("replication", "role") == "master" {
				roleSet = true
				break
			}

			time.Sleep(time.Second)
		}

		if !roleSet {
			sentinelErr = ErrSentinelCantSetRole
		}
	}

	// Restore replica priority
	req.Command = []string{"CONFIG", "SET", "slave-priority", priority}

	_, err = REDIS.ExecCommand(req)

	if sentinelErr != nil {
		return fmt.Errorf("Can't force Sentinel failover: %v", sentinelErr)
	}

	if err != nil {
		return fmt.Errorf("Can't restore slave priority: %v", err)
	}

	return nil
}

// appendStatsData parse info property and append it to given value
func appendStatsData(info *REDIS.Info, prop string, value *uint64) {
	propCategory := strutil.ReadField(prop, 0, false, ':')
	propName := strutil.ReadField(prop, 1, false, ':')

	if propCategory == "" || propName == "" {
		return
	}

	var vu uint64
	var vf float64

	propValue := info.Get(propCategory, propName)

	if strings.Contains(propValue, ".") {
		vf = info.GetF(propCategory, propName)
		vu = uint64(mathutil.Round(vf, 0))
	} else {
		vu = info.GetU(propCategory, propName)
	}

	*value += vu
}

// createConfigFromMeta create config struct based on instance preferencies
func createConfigFromMeta(meta *InstanceMeta) *instanceConfigData {
	result := &instanceConfigData{
		ID:               meta.ID,
		AdminPassword:    meta.Preferencies.AdminPassword,
		SyncPassword:     meta.Preferencies.SyncPassword,
		SentinelPassword: meta.Preferencies.SentinelPassword,
		ServicePassword:  meta.Preferencies.ServicePassword,
		IsSecure:         meta.Preferencies.ServicePassword != "",
		IsSaveDisabled:   meta.Preferencies.IsSaveDisabled,

		RDS: &instanceConfigRDSData{
			HasReplication:     Config.GetS(REPLICATION_ROLE) != "",
			IsMaster:           IsMaster(),
			IsMinion:           IsMinion(),
			IsSentinel:         IsSentinel(),
			IsFailoverStandby:  IsFailoverMethod(FAILOVER_METHOD_STANDBY),
			IsFailoverSentinel: IsFailoverMethod(FAILOVER_METHOD_SENTINEL),
		},

		tags:    meta.Tags,
		storage: meta.Storage,
	}

	if IsInstanceExist(meta.ID) {
		result.Redis = GetInstanceVersion(meta.ID)
	} else {
		result.Redis, _ = GetRedisVersion()
	}

	if IsMinion() && meta.Preferencies.ReplicationType.IsReplica() &&
		Config.GetB(REPLICATION_ALLOW_REPLICAS) {
		result.IsReplica = true
	}

	return result
}

// applyChangedConfigProps apply changed config props
func applyChangedConfigProps(id int, diff []REDIS.ConfigPropDiff) []error {
	meta, err := GetInstanceMeta(id)

	if err != nil {
		return []error{err}
	}

	var errs []error

	req := &REDIS.Request{
		Port:    GetInstancePort(id),
		Auth:    REDIS.Auth{REDIS_USER_ADMIN, meta.Preferencies.AdminPassword},
		Timeout: time.Second,
	}

	for _, info := range diff {
		newValue := info.FileValue

		if strings.Contains(newValue, "\"") {
			newValue = strings.Trim(newValue, "\"")
		}

		req.Command = []string{"CONFIG", "SET", info.PropName, newValue}

		_, err := REDIS.ExecCommand(req)

		if err != nil {
			errs = append(errs, fmt.Errorf("Can't update property: %v", err))
		}
	}

	return errs
}

// isExtendedState returns true if given state contains extended info
func isExtendedState(state State) bool {
	switch {
	case state.IsIdle(), state.IsLoading(), state.IsSaving(), state.IsHang(),
		state.IsAbandoned(), state.IsMasterUp(), state.IsMasterDown(),
		state.NoReplica(), state.WithReplica(), state.WithErrors():
		return true
	}

	return false
}

// execShutdownCommand execute SHUTDOWN command on instance
func execShutdownCommand(id int) error {
	meta, err := GetInstanceMeta(id)

	if err != nil {
		return err
	}

	// We use a small timeout because SHUTDOWN is synchronous command
	// and we don't want to wait for a response
	req := &REDIS.Request{
		Command: []string{"SHUTDOWN"},
		Auth:    REDIS.Auth{REDIS_USER_ADMIN, meta.Preferencies.AdminPassword},
		Timeout: time.Second * 1,
	}

	if knf.GetB(REDIS_SAVE_ON_STOP, true) && !meta.Preferencies.IsSaveDisabled {
		req.Command = append(req.Command, "SAVE")
	}

	ExecCommand(id, req)

	return nil
}

// getSHA256Hash returns SHA-256 hash for given data
func getSHA256Hash(data string) string {
	return fmt.Sprintf("%064x", sha256.Sum256([]byte(data)))
}
