package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"runtime"
	"slices"

	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fmtutil"
	"github.com/essentialkaos/ek/v13/fmtutil/panel"
	"github.com/essentialkaos/ek/v13/fmtutil/table"
	"github.com/essentialkaos/ek/v13/fsutil"
	"github.com/essentialkaos/ek/v13/options"
	"github.com/essentialkaos/ek/v13/pager"
	"github.com/essentialkaos/ek/v13/passwd"
	"github.com/essentialkaos/ek/v13/req"
	"github.com/essentialkaos/ek/v13/spellcheck"
	"github.com/essentialkaos/ek/v13/spinner"
	"github.com/essentialkaos/ek/v13/strutil"
	"github.com/essentialkaos/ek/v13/system"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/terminal/input"
	"github.com/essentialkaos/ek/v13/terminal/tty"
	"github.com/essentialkaos/ek/v13/usage"
	"github.com/essentialkaos/ek/v13/usage/completion/bash"
	"github.com/essentialkaos/ek/v13/usage/completion/fish"
	"github.com/essentialkaos/ek/v13/usage/completion/zsh"
	"github.com/essentialkaos/ek/v13/usage/man"
	"github.com/essentialkaos/ek/v13/usage/update"

	"github.com/essentialkaos/rds/support"

	CORE "github.com/essentialkaos/rds/core"
	RC "github.com/essentialkaos/rds/redis/client"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	APP  = "RDS"
	VER  = "1.11.2"
	DESC = "Tool for Redis orchestration"
)

// CONFIG_FILE is path to configuration file
const CONFIG_FILE = "/etc/rds.knf"

// MAINTENANCE_LOCK_FILE is name of lock file from maintenance mode
const MAINTENANCE_LOCK_FILE = ".maintenance"

// MAX_DESC_LENGTH is maximum width of description line
const MAX_DESC_LENGTH = 64

// EMPTY_RESULT is default value for an empty result from Redis
const EMPTY_RESULT = "-none-"

// Command line options list
const (
	OPT_PRIVATE       = "p:private"
	OPT_EXTRA         = "x:extra"
	OPT_TAGS          = "t:tags"
	OPT_FORMAT        = "f:format"
	OPT_SECURE        = "s:secure"
	OPT_DISABLE_SAVES = "ds:disable-saves"
	OPT_YES           = "y:yes"
	OPT_PAGER         = "P:pager"
	OPT_SIMPLE        = "S:simple"
	OPT_RAW           = "R:raw"
	OPT_NO_COLOR      = "nc:no-color"
	OPT_HELP          = "h:help"
	OPT_VERSION       = "v:version"

	OPT_VERBOSE_VERSION = "vv:verbose-version"
	OPT_GENERATE_MAN    = "generate-man"
	OPT_COMPLETION      = "completion"
)

// Supported commands
const (
	COMMAND_BACKUP_CREATE        = "backup-create"
	COMMAND_BACKUP_RESTORE       = "backup-restore"
	COMMAND_BACKUP_CLEAN         = "backup-clean"
	COMMAND_BACKUP_LIST          = "backup-list"
	COMMAND_BATCH_CREATE         = "batch-create"
	COMMAND_BATCH_EDIT           = "batch-edit"
	COMMAND_CHECK                = "check"
	COMMAND_CLI                  = "cli"
	COMMAND_CLIENTS              = "clients"
	COMMAND_CPU                  = "cpu"
	COMMAND_CONF                 = "conf"
	COMMAND_CREATE               = "create"
	COMMAND_DELETE               = "delete"
	COMMAND_DESTROY              = "destroy"
	COMMAND_EDIT                 = "edit"
	COMMAND_GEN_TOKEN            = "gen-token"
	COMMAND_GO                   = "go"
	COMMAND_HELP                 = "help"
	COMMAND_INFO                 = "info"
	COMMAND_INIT                 = "init"
	COMMAND_KILL                 = "kill"
	COMMAND_LIST                 = "list"
	COMMAND_LOG                  = "log"
	COMMAND_MAINTENANCE          = "maintenance"
	COMMAND_MEMORY               = "memory"
	COMMAND_REGEN                = "regen"
	COMMAND_RELEASE              = "release"
	COMMAND_RELOAD               = "reload"
	COMMAND_REMOVE               = "remove"
	COMMAND_REPLICATION          = "replication"
	COMMAND_REPLICATION_ROLE_SET = "replication-role-set"
	COMMAND_RESTART              = "restart"
	COMMAND_RESTART_ALL          = "restart-all"
	COMMAND_RESTART_ALL_PROP     = "@" + COMMAND_RESTART_ALL
	COMMAND_RESTART_PROP         = "@" + COMMAND_RESTART
	COMMAND_SENTINEL_CHECK       = "sentinel-check"
	COMMAND_SENTINEL_INFO        = "sentinel-info"
	COMMAND_SENTINEL_MASTER      = "sentinel-master"
	COMMAND_SENTINEL_RESET       = "sentinel-reset"
	COMMAND_SENTINEL_START       = "sentinel-start"
	COMMAND_SENTINEL_STATUS      = "sentinel-status"
	COMMAND_SENTINEL_STOP        = "sentinel-stop"
	COMMAND_SENTINEL_SWITCH      = "sentinel-switch-master"
	COMMAND_SETTINGS             = "settings"
	COMMAND_SLOWLOG_GET          = "slowlog-get"
	COMMAND_SLOWLOG_RESET        = "slowlog-reset"
	COMMAND_START                = "start"
	COMMAND_START_ALL            = "start-all"
	COMMAND_START_ALL_PROP       = "@" + COMMAND_START_ALL
	COMMAND_START_PROP           = "@" + COMMAND_START
	COMMAND_STATE_RESTORE        = "state-restore"
	COMMAND_STATE_SAVE           = "state-save"
	COMMAND_STATS                = "stats"
	COMMAND_STATS_COMMAND        = "stats-command"
	COMMAND_STATS_LATENCY        = "stats-latency"
	COMMAND_STATS_ERROR          = "stats-error"
	COMMAND_STATUS               = "status"
	COMMAND_STOP                 = "stop"
	COMMAND_STOP_ALL             = "stop-all"
	COMMAND_STOP_ALL_PROP        = "@" + COMMAND_STOP_ALL
	COMMAND_STOP_PROP            = "@" + COMMAND_STOP
	COMMAND_TAG_ADD              = "tag-add"
	COMMAND_TAG_REMOVE           = "tag-remove"
	COMMAND_TOP                  = "top"
	COMMAND_TOP_DIFF             = "top-diff"
	COMMAND_TOP_DUMP             = "top-dump"
	COMMAND_TRACK                = "track"
	COMMAND_VALIDATE_TEMPLATES   = "validate-templates"
	COMMAND_UPTIME               = "uptime"
)

const (
	FORMAT_TEXT = "text"
	FORMAT_JSON = "json"
	FORMAT_XML  = "xml"
)

const (
	EC_OK     = 0
	EC_ERROR  = 1
	EC_WARN   = 2
	EC_CANCEL = 3
)

// ////////////////////////////////////////////////////////////////////////////////// //

type AuthFlag uint8

func (f AuthFlag) Has(flag AuthFlag) bool { return f&flag == flag }

const (
	AUTH_NO AuthFlag = 1 << iota
	AUTH_INSTANCE
	AUTH_SUPERUSER
	AUTH_STRICT
)

type CommandRoutine struct {
	Handler      CommandHandler
	Auth         AuthFlag
	PrettyOutput bool
}

// ////////////////////////////////////////////////////////////////////////////////// //

// optMap is map with options data
var optMap = options.Map{
	OPT_PRIVATE:       {Type: options.BOOL},
	OPT_EXTRA:         {Type: options.BOOL},
	OPT_TAGS:          {},
	OPT_FORMAT:        {},
	OPT_SECURE:        {Type: options.BOOL},
	OPT_DISABLE_SAVES: {Type: options.BOOL},
	OPT_PAGER:         {Type: options.BOOL},
	OPT_SIMPLE:        {Type: options.BOOL},
	OPT_RAW:           {Type: options.BOOL},
	OPT_YES:           {Type: options.BOOL},
	OPT_NO_COLOR:      {Type: options.BOOL},
	OPT_HELP:          {Type: options.BOOL},
	OPT_VERSION:       {Type: options.MIXED},

	OPT_VERBOSE_VERSION: {Type: options.BOOL},
	OPT_GENERATE_MAN:    {Type: options.BOOL},
	OPT_COMPLETION:      {},
}

// aliases is map [alias → command]
var aliases = map[string]string{
	COMMAND_INIT:    COMMAND_CREATE,
	COMMAND_REMOVE:  COMMAND_DESTROY,
	COMMAND_DELETE:  COMMAND_DESTROY,
	COMMAND_RELEASE: COMMAND_DESTROY,
}

// safeCommands is slice with safe commands that can be executed before
// initialization ('rds go' command)
var safeCommands = []string{
	COMMAND_GEN_TOKEN, COMMAND_GO, COMMAND_HELP, COMMAND_SETTINGS,
}

// dangerousCommands is slice with dangerous commands
var dangerousCommands = []string{
	COMMAND_START, COMMAND_STOP, COMMAND_RESTART, COMMAND_KILL, COMMAND_START_ALL,
	COMMAND_STOP_ALL, COMMAND_RESTART_ALL, COMMAND_RELOAD, COMMAND_REGEN,
	COMMAND_BACKUP_RESTORE, COMMAND_BACKUP_CLEAN,
}

// logger is CLI logger
var logger *Logger

// commands is list of command handlers
var commands map[string]*CommandRoutine

// prefs contains user-specific preferences
var prefs Preferences

// colors for usage info
var colorTagApp, colorTagVer, colorTagRel string

// useRawOutput is raw output flag (for cli command)
var useRawOutput = false

// isTipsEnabled is protip usage flag
var isTipsEnabled bool

// ////////////////////////////////////////////////////////////////////////////////// //

// Init is main function
func Init(gitRev string, gomod []byte) {
	runtime.GOMAXPROCS(2)

	preConfigureUI()

	args := parseOptions()

	if options.Has(OPT_COMPLETION) {
		os.Exit(genCompletion())
	}

	if options.Has(OPT_GENERATE_MAN) {
		genManPage()
		os.Exit(EC_OK)
	}

	prefs = GetPreferences()

	configureUI()
	validateOptions()

	if options.GetB(OPT_HELP) && !isSmartUsageAvailable() {
		showBasicUsage()
		os.Exit(EC_OK)
	}

	if options.GetB(OPT_VERSION) {
		genAbout(gitRev).Print(options.GetS(OPT_VERSION))
		os.Exit(EC_OK)
	}

	if !isUserRoot() {
		terminal.Error("This CLI requires superuser (root) privileges")
		os.Exit(EC_ERROR)
	}

	initRDSCore()

	if options.GetB(OPT_VERBOSE_VERSION) {
		support.Print(APP, VER, gitRev, gomod)
		os.Exit(EC_OK)
	}

	if options.GetB(OPT_HELP) {
		showSmartUsage()
		os.Exit(EC_OK)
	}

	setupLogger()
	disableProxy()

	req.Global.SetUserAgent(APP, VER)

	if len(args) >= 1 {
		initCommands()
		runCommand(args)
	} else {
		showSmartUsage()
	}

	CORE.Shutdown(EC_OK)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// preConfigureUI preconfigures UI
func preConfigureUI() {
	if !tty.IsTTY() {
		fmtc.DisableColors = true
		useRawOutput = true
	}

	switch {
	case fmtc.IsTrueColorSupported():
		colorTagApp, colorTagVer, colorTagRel = "{*}{#DC382C}", "{#A32422}", "{#777777}"
	case fmtc.Is256ColorsSupported():
		colorTagApp, colorTagVer, colorTagRel = "{*}{#160}", "{#124}", "{#244}"
	default:
		colorTagApp, colorTagVer, colorTagRel = "{r*}", "{r}", "{s}"
	}
}

// configureUI configure user interface
func configureUI() {
	if options.GetB(OPT_NO_COLOR) {
		fmtc.DisableColors = true
	}

	input.TitleColorTag = "{s}"
	input.NewLine = true

	if !options.GetB(OPT_SIMPLE) && !prefs.SimpleUI {
		input.Prompt = "{s}›{!} "
		input.MaskSymbol = "•"
		input.MaskSymbolColorTag = "{s-}"

		RC.Prompt = "{s}›{!} "
		RC.UseColoredPrompt = true

		fmtutil.SeparatorSymbol = "–"
		table.BorderSymbol = "–"
	} else {
		fmtutil.SeparatorSymbol = "-"
		table.BorderSymbol = "-"
		spinner.DisableAnimation = true
	}

	table.HeaderCapitalize = true
	fmtutil.SeparatorFullscreen = true
	fmtutil.SizeSeparator = " "
	strutil.EllipsisSuffix = "…"

	if options.GetB(OPT_YES) {
		input.AlwaysYes = true
	}

	if options.GetB(OPT_RAW) {
		useRawOutput = true
	}

	panel.DefaultOptions = append(panel.DefaultOptions, panel.BOTTOM_LINE)

	if prefs.EnablePowerline && !prefs.SimpleUI {
		panel.DefaultOptions = append(panel.DefaultOptions, panel.LABEL_POWERLINE)
	}
}

// parseOptions parse command line options
func parseOptions() options.Arguments {
	args, errs := options.Parse(optMap)

	if len(errs) == 0 {
		return args
	}

	terminal.Error("Options parsing errors:")

	for _, err := range errs {
		terminal.Error("  %v", err)
	}

	os.Exit(EC_ERROR)
	return nil
}

// setupLogger setup logger
func setupLogger() {
	err := CORE.SetLogOutput(CORE.LOG_FILE_CLI, CORE.Config.GetS(CORE.LOG_LEVEL), false)

	if err != nil {
		terminal.Error(err)
		os.Exit(EC_ERROR)
	}

	logger = &Logger{}
}

// disableProxy disable proxy for requests to sync daemon
func disableProxy() {
	os.Setenv("http_proxy", "")
	os.Setenv("https_proxy", "")
	os.Setenv("HTTP_PROXY", "")
	os.Setenv("HTTPS_PROXY", "")
}

// validateOptions validate options
func validateOptions() {
	if options.Has(OPT_FORMAT) {
		switch options.GetS(OPT_FORMAT) {
		case FORMAT_JSON, FORMAT_TEXT, FORMAT_XML:
			// nop
		default:
			terminal.Error("Format %s is not supported", options.GetS(OPT_FORMAT))
			os.Exit(EC_ERROR)
		}
	}
}

// initRDSCore runs RDS core initialization
func initRDSCore() {
	errs := CORE.Init(CONFIG_FILE)

	if len(errs) == 0 {
		return
	}

	terminal.Error("Errors while initialization:")

	for _, err := range errs {
		terminal.Error("  %v", err)
	}

	os.Exit(EC_ERROR)
}

// initCommands initialize supported commands
func initCommands() {
	commands = make(map[string]*CommandRoutine)

	role := CORE.Config.GetS(CORE.REPLICATION_ROLE)
	isMaster := role == CORE.ROLE_MASTER || role == ""
	isMinion := role == CORE.ROLE_MINION
	isSentinel := role == CORE.ROLE_SENTINEL
	isSentinelFailover := CORE.IsFailoverMethod(CORE.FAILOVER_METHOD_SENTINEL)
	allowCommands := CORE.Config.GetB(CORE.REPLICATION_ALLOW_COMMANDS)

	if isMaster {
		commands[COMMAND_BACKUP_CLEAN] = &CommandRoutine{BackupCleanCommand, AUTH_INSTANCE | AUTH_SUPERUSER, true}
		commands[COMMAND_BACKUP_CREATE] = &CommandRoutine{BackupCreateCommand, AUTH_INSTANCE | AUTH_SUPERUSER, true}
		commands[COMMAND_BACKUP_LIST] = &CommandRoutine{BackupListCommand, AUTH_INSTANCE | AUTH_SUPERUSER, true}
		commands[COMMAND_BACKUP_RESTORE] = &CommandRoutine{BackupRestoreCommand, AUTH_INSTANCE | AUTH_SUPERUSER, true}
		commands[COMMAND_KILL] = &CommandRoutine{KillCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_MAINTENANCE] = &CommandRoutine{MaintenanceCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_REGEN] = &CommandRoutine{RegenCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_RELOAD] = &CommandRoutine{ReloadCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_RESTART] = &CommandRoutine{RestartCommand, AUTH_INSTANCE | AUTH_SUPERUSER, true}
		commands[COMMAND_RESTART_ALL] = &CommandRoutine{RestartAllCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_START] = &CommandRoutine{StartCommand, AUTH_INSTANCE | AUTH_SUPERUSER, true}
		commands[COMMAND_START_ALL] = &CommandRoutine{StartAllCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_STOP] = &CommandRoutine{StopCommand, AUTH_INSTANCE | AUTH_SUPERUSER, true}
		commands[COMMAND_STOP_ALL] = &CommandRoutine{StopAllCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
	} else if isMinion && allowCommands {
		commands[COMMAND_BACKUP_CLEAN] = &CommandRoutine{BackupCleanCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_BACKUP_CREATE] = &CommandRoutine{BackupCreateCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_BACKUP_LIST] = &CommandRoutine{BackupListCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_BACKUP_RESTORE] = &CommandRoutine{BackupRestoreCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_KILL] = &CommandRoutine{KillCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_MAINTENANCE] = &CommandRoutine{MaintenanceCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_REGEN] = &CommandRoutine{RegenCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_RELOAD] = &CommandRoutine{ReloadCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_RESTART] = &CommandRoutine{RestartCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_RESTART_ALL] = &CommandRoutine{RestartAllCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_START] = &CommandRoutine{StartCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_START_ALL] = &CommandRoutine{StartAllCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_STOP] = &CommandRoutine{StopCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_STOP_ALL] = &CommandRoutine{StopAllCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
	}

	if isMaster {
		commands[COMMAND_BATCH_CREATE] = &CommandRoutine{BatchCreateCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_BATCH_EDIT] = &CommandRoutine{BatchEditCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_CREATE] = &CommandRoutine{CreateCommand, AUTH_NO, true}
		commands[COMMAND_DESTROY] = &CommandRoutine{DestroyCommand, AUTH_INSTANCE | AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_EDIT] = &CommandRoutine{EditCommand, AUTH_INSTANCE | AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_STATE_RESTORE] = &CommandRoutine{RestoreStateCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_STATE_SAVE] = &CommandRoutine{SaveStateCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_TAG_ADD] = &CommandRoutine{TagAddCommand, AUTH_INSTANCE | AUTH_SUPERUSER, true}
		commands[COMMAND_TAG_REMOVE] = &CommandRoutine{TagRemoveCommand, AUTH_INSTANCE | AUTH_SUPERUSER, true}
	}

	if CORE.IsSyncDaemonInstalled() {
		commands[COMMAND_REPLICATION] = &CommandRoutine{ReplicationCommand, AUTH_NO, options.GetS(OPT_FORMAT) == "" && !useRawOutput}

		if !isSentinelFailover {
			commands[COMMAND_REPLICATION_ROLE_SET] = &CommandRoutine{ReplicationRoleSetCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		}
	}

	if options.GetB(OPT_PRIVATE) {
		commands[COMMAND_CONF] = &CommandRoutine{ConfCommand, AUTH_INSTANCE | AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_CLI] = &CommandRoutine{CliCommand, AUTH_INSTANCE | AUTH_SUPERUSER | AUTH_STRICT, !useRawOutput}

		if CORE.HasSUAuth() {
			commands[COMMAND_SETTINGS] = &CommandRoutine{SettingsCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		} else {
			commands[COMMAND_SETTINGS] = &CommandRoutine{SettingsCommand, AUTH_NO, true}
		}
	} else {
		commands[COMMAND_CONF] = &CommandRoutine{ConfCommand, AUTH_NO, true}
		commands[COMMAND_CLI] = &CommandRoutine{CliCommand, AUTH_NO, !useRawOutput}
		commands[COMMAND_SETTINGS] = &CommandRoutine{SettingsCommand, AUTH_NO, true}
	}

	if isSentinel {
		commands[COMMAND_CONF] = nil
		commands[COMMAND_CLI] = nil
	}

	if !isSentinel {
		commands[COMMAND_CHECK] = &CommandRoutine{CheckCommand, AUTH_NO, true}
		commands[COMMAND_CLIENTS] = &CommandRoutine{ClientsCommand, AUTH_NO, true}
		commands[COMMAND_CPU] = &CommandRoutine{CPUCommand, AUTH_NO, true}
		commands[COMMAND_GO] = &CommandRoutine{GoCommand, AUTH_NO, true}
		commands[COMMAND_INFO] = &CommandRoutine{InfoCommand, AUTH_NO, options.GetS(OPT_FORMAT) == "" && !useRawOutput}
		commands[COMMAND_LIST] = &CommandRoutine{ListCommand, AUTH_NO, !useRawOutput}
		commands[COMMAND_LOG] = &CommandRoutine{LogCommand, AUTH_NO, !useRawOutput}
		commands[COMMAND_MEMORY] = &CommandRoutine{MemoryCommand, AUTH_NO, !useRawOutput}
		commands[COMMAND_SLOWLOG_GET] = &CommandRoutine{SlowlogGetCommand, AUTH_NO, true}
		commands[COMMAND_SLOWLOG_RESET] = &CommandRoutine{SlowlogResetCommand, AUTH_INSTANCE | AUTH_SUPERUSER, true}
		commands[COMMAND_STATS] = &CommandRoutine{StatsCommand, AUTH_NO, options.GetS(OPT_FORMAT) == "" && !useRawOutput}
		commands[COMMAND_STATS_COMMAND] = &CommandRoutine{StatsCommandCommand, AUTH_NO, true}
		commands[COMMAND_STATS_ERROR] = &CommandRoutine{StatsErrorCommand, AUTH_NO, true}
		commands[COMMAND_STATS_LATENCY] = &CommandRoutine{StatsLatencyCommand, AUTH_NO, true}
		commands[COMMAND_STATUS] = &CommandRoutine{StatusCommand, AUTH_NO, true}
		commands[COMMAND_TOP] = &CommandRoutine{TopCommand, AUTH_NO, !useRawOutput}
		commands[COMMAND_TOP_DIFF] = &CommandRoutine{TopDiffCommand, AUTH_NO, true}
		commands[COMMAND_TOP_DUMP] = &CommandRoutine{TopDumpCommand, AUTH_NO, true}
		commands[COMMAND_TRACK] = &CommandRoutine{TrackCommand, AUTH_NO, true}
		commands[COMMAND_UPTIME] = &CommandRoutine{UptimeCommand, AUTH_NO, true}
	}

	if isSentinelFailover {
		if isMaster {
			commands[COMMAND_SENTINEL_START] = &CommandRoutine{SentinelStartCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
			commands[COMMAND_SENTINEL_STOP] = &CommandRoutine{SentinelStopCommand, AUTH_SUPERUSER | AUTH_STRICT, true}

			if CORE.IsSentinelActive() {
				commands[COMMAND_SENTINEL_SWITCH] = &CommandRoutine{SentinelSwitchMasterCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
			}
		}

		commands[COMMAND_SENTINEL_STATUS] = &CommandRoutine{SentinelStatusCommand, AUTH_NO, true}

		if CORE.IsSentinelActive() {
			commands[COMMAND_SENTINEL_CHECK] = &CommandRoutine{SentinelCheckCommand, AUTH_NO, true}
			commands[COMMAND_SENTINEL_INFO] = &CommandRoutine{SentinelInfoCommand, AUTH_NO, true}
			commands[COMMAND_SENTINEL_MASTER] = &CommandRoutine{SentinelMasterCommand, AUTH_NO, true}
			commands[COMMAND_SENTINEL_RESET] = &CommandRoutine{SentinelResetCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		}
	}

	if isMaster {
		if CORE.Config.GetB(CORE.REPLICATION_ALWAYS_PROPAGATE) {
			commands[COMMAND_RESTART] = &CommandRoutine{RestartPropCommand, AUTH_INSTANCE | AUTH_SUPERUSER, true}
			commands[COMMAND_RESTART_ALL] = &CommandRoutine{RestartAllPropCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
			commands[COMMAND_START] = &CommandRoutine{StartPropCommand, AUTH_INSTANCE | AUTH_SUPERUSER, true}
			commands[COMMAND_START_ALL] = &CommandRoutine{StartAllPropCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
			commands[COMMAND_STOP] = &CommandRoutine{StopPropCommand, AUTH_INSTANCE | AUTH_SUPERUSER, true}
			commands[COMMAND_STOP_ALL] = &CommandRoutine{StopAllPropCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		}

		commands[COMMAND_RESTART_ALL_PROP] = &CommandRoutine{RestartAllPropCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_RESTART_PROP] = &CommandRoutine{RestartPropCommand, AUTH_INSTANCE | AUTH_SUPERUSER, true}
		commands[COMMAND_START_ALL_PROP] = &CommandRoutine{StartAllPropCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_START_PROP] = &CommandRoutine{StartPropCommand, AUTH_INSTANCE | AUTH_SUPERUSER, true}
		commands[COMMAND_STOP_ALL_PROP] = &CommandRoutine{StopAllPropCommand, AUTH_SUPERUSER | AUTH_STRICT, true}
		commands[COMMAND_STOP_PROP] = &CommandRoutine{StopPropCommand, AUTH_INSTANCE | AUTH_SUPERUSER, true}
	}

	commands[COMMAND_VALIDATE_TEMPLATES] = &CommandRoutine{ValidateTemplatesCommand, AUTH_NO, true}
	commands[COMMAND_GEN_TOKEN] = &CommandRoutine{GenTokenCommand, AUTH_NO, !useRawOutput}
	commands[COMMAND_HELP] = &CommandRoutine{HelpCommand, AUTH_NO, true}

	for a, c := range aliases {
		if commands[c] != nil {
			commands[a] = commands[c]
		}
	}

	if isMaintenanceLockSet() {
		enableMaintenanceMode()
	}
}

// runCommand authorize user and execute command handler
func runCommand(args options.Arguments) {
	cmd := args.Get(0).String()

	if !checkCommand(cmd) {
		CORE.Shutdown(EC_ERROR)
	}

	if cmd == COMMAND_HELP && len(args) == 1 {
		showSmartUsage()
		return
	}

	if !CORE.HasSUAuth() && !CORE.IsMinion() {
		if !slices.Contains(safeCommands, cmd) {
			fmtc.Println("\nBefore usage you must execute {*}rds go{!} command.\n")
			return
		}
	}

	cr := commands[cmd]

	if isMaintenanceLockSet() {
		fmtc.NewLine()
		panel.Warn(
			"Node in maintenance mode",
			`This RDS node is currently in maintenance mode. The superuser password is required
for {*_}ANY{!} command.

{s}Maintenance mode is used as a protection when devops/sysadmins perform some dangerous{!}
{s}actions like Redis update or node reconfiguration. Please be patient, it usually{!}
{s}doesn't take long.{!}`,
		)
	}

	if CORE.IsMinion() && slices.Contains(dangerousCommands, cmd) {
		fmtc.NewLine()
		panel.Warn("Executing commands on a minion node",
			`Note that you are running a dangerous command on a {*c}minion{!} node. You must do so
with {*}extreme care{!}.`,
		)
	}

	isTipsEnabled = checkForTips(cmd)

	executeCommandRoutine(cr, args.Strings()[1:])
}

// executeCommandRoutine execute command
func executeCommandRoutine(cr *CommandRoutine, args []string) {
	var err error

	if cr.PrettyOutput {
		fmtc.NewLine()
	}

	if !cr.Auth.Has(AUTH_NO) {
		var ok bool

		if cr.Auth.Has(AUTH_INSTANCE) {
			if len(args) == 0 {
				terminal.Error("You must provide instance ID for this command")
				fmtc.NewLine()
				return
			}

			ok, err = authenticate(cr.Auth, cr.Auth.Has(AUTH_STRICT), args[0])
		} else {
			ok, err = authenticate(cr.Auth, cr.Auth.Has(AUTH_STRICT), "")
		}

		if err != nil {
			terminal.Error(err)

			if cr.PrettyOutput {
				fmtc.NewLine()
			}

			return
		}

		if !ok {
			terminal.Error("Can't authenticate you with given password")

			if cr.PrettyOutput {
				fmtc.NewLine()
			}

			return
		}
	}

	ec := cr.Handler(CommandArgs(args))

	if cr.PrettyOutput {
		if ec == 0 && isTipsEnabled {
			if !showTip() {
				fmtc.NewLine()
			}
		} else {
			fmtc.NewLine()
		}
	}

	CORE.Shutdown(ec)
}

// authenticate authenticate user
func authenticate(auth AuthFlag, strict bool, instanceID string) (bool, error) {
	var iAuth *CORE.InstanceAuth
	var sAuth *CORE.SuperuserAuth

	var password string
	var err error
	var id int

	var message = "Please enter superuser password"

	if auth.Has(AUTH_INSTANCE) {
		id, _, err = CORE.ParseIDDBPair(instanceID)

		if err != nil {
			return false, err
		}

		if !CORE.IsInstanceExist(id) {
			return false, fmt.Errorf("Instance with ID %d does not exist", id)
		}

		meta, err := CORE.GetInstanceMeta(id)

		if err != nil {
			return false, nil
		}

		iAuth = meta.Auth

		if !strict && !CORE.Config.GetB(CORE.MAIN_STRICT_SECURE) {
			if CORE.User.RealName == iAuth.User {
				return true, nil
			}
		}

		message = "Please enter instance password or superuser password"
	}

	password, err = input.ReadPassword(message, input.NotEmpty)

	if err != nil {
		CORE.Shutdown(EC_OK)
	}

	passwordVariations := passwd.GenPasswordVariations(password)

	if iAuth != nil {
		if passwd.Check(password, iAuth.Pepper, iAuth.Hash) {
			return true, nil
		}

		for _, pwd := range passwordVariations {
			if passwd.Check(pwd, iAuth.Pepper, iAuth.Hash) {
				return true, nil
			}
		}
	}

	sAuth, err = CORE.ReadSUAuth()

	if err != nil {
		return false, nil
	}

	if sAuth != nil {
		if passwd.Check(password, sAuth.Pepper, sAuth.Hash) {
			return true, nil
		}

		for _, pwd := range passwordVariations {
			if passwd.Check(pwd, sAuth.Pepper, sAuth.Hash) {
				return true, nil
			}
		}
	}

	return false, nil
}

// checkForTips checks if protip must be shown
func checkForTips(cmd string) bool {
	if CORE.Config.GetB(CORE.MAIN_DISABLE_TIPS) || prefs.DisableTips {
		return false
	}

	switch cmd {
	case COMMAND_CREATE, COMMAND_DESTROY, COMMAND_EDIT,
		COMMAND_START, COMMAND_STOP, COMMAND_RESTART, COMMAND_KILL,
		COMMAND_STATUS, COMMAND_CPU, COMMAND_MEMORY, COMMAND_INFO,
		COMMAND_CLIENTS, COMMAND_CONF, COMMAND_LIST, COMMAND_STATS,
		COMMAND_STATS_COMMAND, COMMAND_STATS_LATENCY, COMMAND_STATS_ERROR,
		COMMAND_TOP, COMMAND_TOP_DIFF, COMMAND_TOP_DUMP, COMMAND_SLOWLOG_GET,
		COMMAND_SLOWLOG_RESET, COMMAND_TAG_ADD, COMMAND_TAG_REMOVE,
		COMMAND_CHECK, COMMAND_BACKUP_CREATE, COMMAND_BACKUP_RESTORE,
		COMMAND_BACKUP_CLEAN, COMMAND_BACKUP_LIST:
		return true
	}

	return false
}

// getSpellcheckModel train spellchecker with supported commands
func getSpellcheckModel() *spellcheck.Model {
	return spellcheck.Train([]string{
		COMMAND_BACKUP_CREATE, COMMAND_BACKUP_RESTORE, COMMAND_BACKUP_CLEAN,
		COMMAND_BACKUP_LIST, COMMAND_BATCH_CREATE, COMMAND_BATCH_EDIT, COMMAND_CHECK,
		COMMAND_CLI, COMMAND_CPU, COMMAND_CONF, COMMAND_CREATE, COMMAND_DELETE,
		COMMAND_DESTROY, COMMAND_EDIT, COMMAND_GEN_TOKEN, COMMAND_GO, COMMAND_HELP,
		COMMAND_INFO, COMMAND_INIT, COMMAND_KILL, COMMAND_LIST, COMMAND_MAINTENANCE,
		COMMAND_MEMORY, COMMAND_REGEN, COMMAND_RELEASE, COMMAND_RELOAD,
		COMMAND_REMOVE, COMMAND_REPLICATION, COMMAND_REPLICATION_ROLE_SET,
		COMMAND_RESTART, COMMAND_RESTART_ALL, COMMAND_RESTART_ALL_PROP,
		COMMAND_RESTART_PROP, COMMAND_SENTINEL_CHECK, COMMAND_SENTINEL_INFO,
		COMMAND_SENTINEL_MASTER, COMMAND_SENTINEL_RESET, COMMAND_SENTINEL_START,
		COMMAND_SENTINEL_STATUS, COMMAND_SENTINEL_STOP, COMMAND_SENTINEL_SWITCH,
		COMMAND_SETTINGS, COMMAND_SLOWLOG_GET, COMMAND_SLOWLOG_RESET, COMMAND_START,
		COMMAND_START_ALL, COMMAND_START_ALL_PROP, COMMAND_START_PROP,
		COMMAND_STATE_RESTORE, COMMAND_STATE_SAVE, COMMAND_STATS,
		COMMAND_STATS_COMMAND, COMMAND_STATS_LATENCY, COMMAND_STATS_ERROR,
		COMMAND_STATUS, COMMAND_STOP, COMMAND_STOP_ALL, COMMAND_STOP_ALL_PROP,
		COMMAND_STOP_PROP, COMMAND_TAG_ADD, COMMAND_TAG_REMOVE, COMMAND_TOP,
		COMMAND_TOP_DIFF, COMMAND_TOP_DUMP, COMMAND_TRACK, COMMAND_VALIDATE_TEMPLATES,
		COMMAND_UPTIME,
	})
}

// checkCommand checks if command with given name is presented
func checkCommand(cmd string) bool {
	_, ok := commands[cmd]

	if ok {
		return true
	}

	scm := getSpellcheckModel()
	cmdCrt := scm.Correct(cmd)

	if cmdCrt == cmd {
		terminal.Error("\nUnknown command %q\n", cmd)
	} else {
		terminal.Error("\nUnknown command %q. Maybe you meant %q?\n", cmd, cmdCrt)
	}

	switch {
	case CORE.IsMinion():
		panel.Warn("Available commands on Minion node",
			`Note that you are running the command on the {c*}minion{!} node, and some destructive
commands aren't available on it (see '{*}rds{!} {g}--help{!}').`)
		fmtc.NewLine()
	case CORE.IsSentinel():
		panel.Warn("Available commands on Sentinel node",
			`Note that you are running the command on the {m*}sentinel{!} node, and some destructive
commands aren't available on it (see '{*}rds{!} {g}--help{!}').`)
		fmtc.NewLine()
	}

	return false
}

// enableMaintenanceMode enables maintenance mode
func enableMaintenanceMode() {
	for _, c := range commands {
		c.Auth = AUTH_SUPERUSER
	}
}

// isMaintenanceLockSet returns true if maintenance mode is enabled
func isMaintenanceLockSet() bool {
	return fsutil.IsExist(getMaintenanceLockPath())
}

// createMaintenanceLock creates maintenance mode lock file
func createMaintenanceLock() error {
	if isMaintenanceLockSet() {
		return nil
	}

	fd, err := os.OpenFile(getMaintenanceLockPath(), os.O_CREATE, 0600)

	if err != nil {
		return fmt.Errorf("Can't create lock file: %w", err)
	}

	fd.Close()

	return nil
}

// removeMaintenanceLock removes maintenance mode lock file
func removeMaintenanceLock() error {
	if !isMaintenanceLockSet() {
		return nil
	}

	err := os.Remove(getMaintenanceLockPath())

	if err != nil {
		return fmt.Errorf("Can't remove lock file: %w", err)
	}

	return nil
}

// getMaintenanceLockPath returns path to maintenance lock file
func getMaintenanceLockPath() string {
	lockFile := CORE.Config.GetS(CORE.MAIN_DIR)
	return lockFile + "/" + MAINTENANCE_LOCK_FILE
}

// isUserRoot returns true if current user is root
func isUserRoot() bool {
	curUser, err := system.CurrentUser()

	return err == nil && curUser.IsRoot()
}

// isSmartUsageAvailable returns true if we can show smart usage info
func isSmartUsageAvailable() bool {
	return isUserRoot() && fsutil.IsExist(CONFIG_FILE)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// showBasicUsage prints basic usage info
func showBasicUsage() {
	if options.GetB(OPT_PAGER) || prefs.AutoPaging {
		if pager.Setup() == nil {
			defer pager.Complete()
		}
	}

	genUsage().Print()
}

// showSmartUsage prints usage info based on current settings
func showSmartUsage() {
	info := usage.NewInfo()

	role := CORE.Config.GetS(CORE.REPLICATION_ROLE)
	isMaster := role == CORE.ROLE_MASTER || role == ""
	isMinion := role == CORE.ROLE_MINION
	isSentinel := role == CORE.ROLE_SENTINEL
	isSentinelFailover := CORE.IsFailoverMethod(CORE.FAILOVER_METHOD_SENTINEL)
	allowCommands := CORE.Config.GetB(CORE.REPLICATION_ALLOW_COMMANDS)

	if options.GetB(OPT_PAGER) || prefs.AutoPaging {
		if pager.Setup() == nil {
			defer pager.Complete()
		}
	}

	if isMaster {
		if !CORE.Config.GetB(CORE.REPLICATION_ALWAYS_PROPAGATE) && CORE.IsSyncDaemonActive() {
			info.AddSpoiler("  Use prefix @ for propagating command to RDS minions (works with {y}start{!},\n" +
				"  {y}stop{!}, {y}restart{!}, {y}start-all{!}, {y}stop-all{!} and {y}restart-all{!} commands).")
		}
	}

	info.AddGroup("Basic commands")

	if isMaster {
		info.AddCommand(COMMAND_CREATE, "Create new Redis instance")
		info.AddCommand(COMMAND_DESTROY, "Destroy {s}(delete){!} Redis instance", "id")
		info.AddCommand(COMMAND_EDIT, "Edit metadata for instance", "id")
	}

	if isMaster || (isMinion && allowCommands) {
		info.AddCommand(COMMAND_START, "Start Redis instance", "id")
		info.AddCommand(COMMAND_STOP, "Stop Redis instance", "id", "?force")
		info.AddCommand(COMMAND_RESTART, "Restart Redis instance", "id")
		info.AddCommand(COMMAND_KILL, "Kill Redis instance", "id")
	}

	if !isSentinel {
		info.AddCommand(COMMAND_STATUS, "Show current status of Redis instance", "id")
		info.AddCommand(COMMAND_CLI, "Run CLI connected to Redis instance", "id:db", "?command")
		info.AddCommand(COMMAND_CPU, "Calculate instance CPU usage", "id", "?period")
		info.AddCommand(COMMAND_MEMORY, "Show instance memory usage", "id")
		info.AddCommand(COMMAND_INFO, "Show system info about Redis instance", "id", "?section…")
		info.AddCommand(COMMAND_CLIENTS, "Show list of connected clients", "id", "?filter")
		info.AddCommand(COMMAND_TRACK, "Show interactive info about Redis instance", "id", "?interval")
		info.AddCommand(COMMAND_CONF, "Show configuration of Redis instance", "id", "?filter…")
		info.AddCommand(COMMAND_LIST, "Show list of all Redis instances", "?filter…")
		info.AddCommand(COMMAND_LOG, "Show RDS or Redis instance logs", "source")
		info.AddCommand(COMMAND_STATS, "Show overall statistics")
		info.AddCommand(COMMAND_STATS_COMMAND, "Show statistics based on the command type", "id")
		info.AddCommand(COMMAND_STATS_LATENCY, "Show latency statistics based on the command type", "id")
		info.AddCommand(COMMAND_STATS_ERROR, "Show error statistics", "id")
		info.AddCommand(COMMAND_UPTIME, "Show instances uptime info")
		info.AddCommand(COMMAND_TOP, "Show instances top", "?field", "?num")
		info.AddCommand(COMMAND_TOP_DIFF, "Compare current and dumped top data", "file", "?field", "?num")
		info.AddCommand(COMMAND_TOP_DUMP, "Dump top data to file", "file")
		info.AddCommand(COMMAND_SLOWLOG_GET, "Show last entries from slow log", "id", "?num")
		info.AddCommand(COMMAND_SLOWLOG_RESET, "Clear slow log", "id")

		if isMaster {
			info.AddCommand(COMMAND_TAG_ADD, "Add tag to instance", "id", "tag")
			info.AddCommand(COMMAND_TAG_REMOVE, "Remove tag from instance", "id", "tag")
		}

		info.AddCommand(COMMAND_CHECK, "Check for dead instances")
	}

	info.AddGroup("Backup commands")

	info.AddCommand(COMMAND_BACKUP_CREATE, "Create snapshot of RDB file", "id")
	info.AddCommand(COMMAND_BACKUP_RESTORE, "Restore instance data from snapshot", "id")
	info.AddCommand(COMMAND_BACKUP_CLEAN, "Remove all backup snapshots", "id")
	info.AddCommand(COMMAND_BACKUP_LIST, "List backup snapshots", "id")

	info.AddGroup("Superuser commands")

	if isMaster {
		info.AddCommand(COMMAND_GO, "Generate superuser access credentials")
		info.AddCommand(COMMAND_BATCH_CREATE, "Create many instances at once", "csv-file")
		info.AddCommand(COMMAND_BATCH_EDIT, "Edit many instances at once", "id…")
	}

	if isMaster || (isMinion && allowCommands) {
		info.AddCommand(COMMAND_MAINTENANCE, "Enable or disable maintenance mode", "flag")
		info.AddCommand(COMMAND_STOP_ALL, "Stop all instances")
		info.AddCommand(COMMAND_START_ALL, "Start all instances")
		info.AddCommand(COMMAND_RESTART_ALL, "Restart all instances")
		info.AddCommand(COMMAND_RELOAD, "Reload configuration for one or all instances", "id")
		info.AddCommand(COMMAND_REGEN, "Regenerate configuration file for one or all instances", "id")

		if !isMinion {
			info.AddCommand(COMMAND_STATE_SAVE, "Save state of all instances", "file")
			info.AddCommand(COMMAND_STATE_RESTORE, "Restore state of all instances", "?file")
		}
	}

	if CORE.IsSyncDaemonInstalled() {
		info.AddGroup("Replication commands")

		info.AddCommand(COMMAND_REPLICATION, "Show replication info")

		if !isSentinelFailover {
			info.AddCommand(COMMAND_REPLICATION_ROLE_SET, "Change node role", "target-role")
		}
	}

	if isSentinelFailover {
		info.AddGroup("Sentinel commands")

		if isMaster {
			info.AddCommand(COMMAND_SENTINEL_START, "Start Redis Sentinel daemon")
			info.AddCommand(COMMAND_SENTINEL_STOP, "Stop Redis Sentinel daemon")
		}

		if !isSentinel {
			info.AddCommand(COMMAND_SENTINEL_STATUS, "Show status of Redis Sentinel daemon")

			if CORE.IsSentinelActive() {
				info.AddCommand(COMMAND_SENTINEL_INFO, "Show info from Sentinel for some instance", "id")
				info.AddCommand(COMMAND_SENTINEL_MASTER, "Show IP of master instance", "id")
				info.AddCommand(COMMAND_SENTINEL_CHECK, "Check Sentinel configuration", "id")
				info.AddCommand(COMMAND_SENTINEL_RESET, "Reset state in Sentinel for all instances")

				if isMaster {
					info.AddCommand(COMMAND_SENTINEL_SWITCH, "Switch instance to master role", "id")
				}
			}
		}
	}

	info.AddGroup("Common commands")

	info.AddCommand(COMMAND_HELP, "Show command usage info", "command")
	info.AddCommand(COMMAND_SETTINGS, "Show settings from global configuration file", "?section…")
	info.AddCommand(COMMAND_GEN_TOKEN, "Generate authentication token for sync daemon")
	info.AddCommand(COMMAND_VALIDATE_TEMPLATES, "Validate Redis and Sentinel templates")

	if isMaster {
		info.AddOption(OPT_SECURE, "Create secure Redis instance with auth support ({y}create{!})")
		info.AddOption(OPT_DISABLE_SAVES, "Disable saves for created instance ({y}create{!})")
	}

	info.AddOption(OPT_PRIVATE, "Force access to private data ({y}conf{!}/{y}cli{!}/{y}settings{!})")
	info.AddOption(OPT_EXTRA, "Print extra info ({y}list{!})")
	info.AddOption(OPT_TAGS, "List of tags ({y}create{!})", "tag…")
	info.AddOption(OPT_FORMAT, "Output format {s-}(text/json/xml){!}", "format")
	info.AddOption(OPT_YES, "Automatically answer yes for all questions")
	info.AddOption(OPT_PAGER, "Enable pager for long output")
	info.AddOption(OPT_SIMPLE, "Simplify output {s-}(useful for copy-paste){!}")
	info.AddOption(OPT_RAW, "Force raw output {s-}(useful for scripts){!}")
	info.AddOption(OPT_NO_COLOR, "Disable colors in output")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VERSION, "Show information about version")
	info.AddOption(OPT_VERBOSE_VERSION, "Show verbose information about version")

	info.BoundOptions(COMMAND_CLI, OPT_TAGS)
	info.BoundOptions(COMMAND_CLIENTS, OPT_PAGER)
	info.BoundOptions(COMMAND_CONF, OPT_TAGS, OPT_PAGER)
	info.BoundOptions(COMMAND_CREATE, OPT_SECURE, OPT_DISABLE_SAVES, OPT_TAGS)
	info.BoundOptions(COMMAND_HELP, OPT_PAGER)
	info.BoundOptions(COMMAND_INFO, OPT_FORMAT, OPT_PAGER)
	info.BoundOptions(COMMAND_LIST, OPT_EXTRA, OPT_PAGER)
	info.BoundOptions(COMMAND_MEMORY, OPT_FORMAT)
	info.BoundOptions(COMMAND_REPLICATION, OPT_FORMAT)
	info.BoundOptions(COMMAND_SENTINEL_INFO, OPT_PAGER)
	info.BoundOptions(COMMAND_SETTINGS, OPT_TAGS, OPT_PAGER)
	info.BoundOptions(COMMAND_SLOWLOG_GET, OPT_PAGER)
	info.BoundOptions(COMMAND_STATS, OPT_FORMAT)
	info.BoundOptions(COMMAND_STATS_COMMAND, OPT_PAGER)
	info.BoundOptions(COMMAND_STATS_ERROR, OPT_PAGER)
	info.BoundOptions(COMMAND_STATS_LATENCY, OPT_PAGER)
	info.BoundOptions(COMMAND_TOP, OPT_PAGER)
	info.BoundOptions(COMMAND_TOP_DIFF, OPT_PAGER)

	info.Print()

	fmtc.Println("{s-}This content is dynamic and based on current RDS settings.{!}\n")
}

// genUsage generates basic usage info used for completion generation
func genUsage() *usage.Info {
	info := usage.NewInfo()

	info.AppNameColorTag = colorTagApp

	info.AddGroup("Basic commands")

	info.AddCommand(COMMAND_CREATE, "Create new Redis instance")
	info.AddCommand(COMMAND_DESTROY, "Destroy {s}(delete){!} Redis instance", "id")
	info.AddCommand(COMMAND_EDIT, "Edit metadata for instance", "id")
	info.AddCommand(COMMAND_START, "Start Redis instance", "id")
	info.AddCommand(COMMAND_STOP, "Stop Redis instance", "id", "?force")
	info.AddCommand(COMMAND_RESTART, "Restart Redis instance", "id")
	info.AddCommand(COMMAND_KILL, "Kill Redis instance", "id")
	info.AddCommand(COMMAND_STATUS, "Show current status of Redis instance", "id")
	info.AddCommand(COMMAND_CLI, "Run CLI connected to Redis instance", "id:db", "?command")
	info.AddCommand(COMMAND_CPU, "Calculate instance CPU usage", "id", "?period")
	info.AddCommand(COMMAND_MEMORY, "Show instance memory usage", "id")
	info.AddCommand(COMMAND_INFO, "Show system info about Redis instance", "id", "?section…")
	info.AddCommand(COMMAND_STATS_COMMAND, "Show statistics based on the command type", "id")
	info.AddCommand(COMMAND_STATS_LATENCY, "Show latency statistics based on the command type", "id")
	info.AddCommand(COMMAND_STATS_ERROR, "Show error statistics", "id")
	info.AddCommand(COMMAND_CLIENTS, "Show list of connected clients", "id", "?filter")
	info.AddCommand(COMMAND_TRACK, "Show interactive info about Redis instance", "id", "?interval")
	info.AddCommand(COMMAND_CONF, "Show configuration of Redis instance", "id", "?filter…")
	info.AddCommand(COMMAND_LIST, "Show list of all Redis instances", "?filter…")
	info.AddCommand(COMMAND_LOG, "Show RDS or Redis instance logs", "source")
	info.AddCommand(COMMAND_STATS, "Show overall statistics")
	info.AddCommand(COMMAND_TOP, "Show instances top", "?field", "?num")
	info.AddCommand(COMMAND_TOP_DIFF, "Compare current and dumped top data", "file", "?field", "?num")
	info.AddCommand(COMMAND_TOP_DUMP, "Dump top data to file", "file")
	info.AddCommand(COMMAND_UPTIME, "Show instances uptime info")
	info.AddCommand(COMMAND_SLOWLOG_GET, "Show last entries from slow log", "id", "?num")
	info.AddCommand(COMMAND_SLOWLOG_RESET, "Clear slow log", "id")
	info.AddCommand(COMMAND_TAG_ADD, "Add tag to instance", "id", "tag")
	info.AddCommand(COMMAND_TAG_REMOVE, "Remove tag from instance", "id", "tag")
	info.AddCommand(COMMAND_CHECK, "Check for dead instances")

	info.AddGroup("Backup commands")

	info.AddCommand(COMMAND_BACKUP_CREATE, "Create snapshot of RDB file", "id")
	info.AddCommand(COMMAND_BACKUP_RESTORE, "Restore instance data from snapshot", "id")
	info.AddCommand(COMMAND_BACKUP_CLEAN, "Remove all backup snapshots", "id")
	info.AddCommand(COMMAND_BACKUP_LIST, "List backup snapshots", "id")

	info.AddGroup("Superuser commands")

	info.AddCommand(COMMAND_GO, "Generate superuser access credentials")
	info.AddCommand(COMMAND_BATCH_CREATE, "Create many instances at once", "csv-file")
	info.AddCommand(COMMAND_BATCH_EDIT, "Edit many instances at once", "id…")
	info.AddCommand(COMMAND_STOP_ALL, "Stop all instances")
	info.AddCommand(COMMAND_START_ALL, "Start all instances")
	info.AddCommand(COMMAND_RESTART_ALL, "Restart all instances")
	info.AddCommand(COMMAND_RELOAD, "Reload configuration for one or all instances", "id")
	info.AddCommand(COMMAND_REGEN, "Regenerate configuration file for one or all instances", "id")
	info.AddCommand(COMMAND_STATE_SAVE, "Save state of all instances", "file")
	info.AddCommand(COMMAND_STATE_RESTORE, "Restore state of all instances", "?file")
	info.AddCommand(COMMAND_MAINTENANCE, "Enable or disable maintenance mode", "flag")

	info.AddGroup("Replication commands")

	info.AddCommand(COMMAND_REPLICATION, "Show replication info")
	info.AddCommand(COMMAND_REPLICATION_ROLE_SET, "Change node role", "target-role")

	info.AddGroup("Sentinel commands")

	info.AddCommand(COMMAND_SENTINEL_START, "Start Redis Sentinel daemon")
	info.AddCommand(COMMAND_SENTINEL_STOP, "Stop Redis Sentinel daemon")
	info.AddCommand(COMMAND_SENTINEL_STATUS, "Show status of Redis Sentinel daemon")
	info.AddCommand(COMMAND_SENTINEL_INFO, "Show info from Sentinel for some instance", "id")
	info.AddCommand(COMMAND_SENTINEL_MASTER, "Show IP of master instance", "id")
	info.AddCommand(COMMAND_SENTINEL_CHECK, "Check Sentinel configuration", "id")
	info.AddCommand(COMMAND_SENTINEL_RESET, "Reset state in Sentinel for all instances")
	info.AddCommand(COMMAND_SENTINEL_SWITCH, "Switch instance to master role", "id")

	info.AddGroup("Common commands")

	info.AddCommand(COMMAND_HELP, "Show command usage info", "command")
	info.AddCommand(COMMAND_SETTINGS, "Show settings from global configuration file", "?section…")
	info.AddCommand(COMMAND_GEN_TOKEN, "Generate authentication token for sync daemon")
	info.AddCommand(COMMAND_VALIDATE_TEMPLATES, "Validate Redis and Sentinel templates")

	info.AddOption(OPT_SECURE, "Create secure Redis instance with auth support ({y}create{!})")
	info.AddOption(OPT_DISABLE_SAVES, "Disable saves for created instance ({y}create{!})")
	info.AddOption(OPT_PRIVATE, "Force access to private data ({y}conf{!}/{y}cli{!}/{y}settings{!})")
	info.AddOption(OPT_EXTRA, "Print extra info ({y}list{!})")
	info.AddOption(OPT_TAGS, "List of tags ({y}create{!})", "tag…")
	info.AddOption(OPT_FORMAT, "Output format {s-}(text/json/xml){!}", "format")
	info.AddOption(OPT_YES, "Automatically answer yes for all questions")
	info.AddOption(OPT_PAGER, "Enable pager for long output")
	info.AddOption(OPT_SIMPLE, "Simplify output {s-}(useful for copy-paste){!}")
	info.AddOption(OPT_RAW, "Force raw output {s-}(useful for scripts){!}")
	info.AddOption(OPT_NO_COLOR, "Disable colors in output")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VERSION, "Show information about version")
	info.AddOption(OPT_VERBOSE_VERSION, "Show verbose information about version")

	info.BoundOptions(COMMAND_CLI, OPT_TAGS)
	info.BoundOptions(COMMAND_CLIENTS, OPT_PAGER)
	info.BoundOptions(COMMAND_CONF, OPT_TAGS, OPT_PAGER)
	info.BoundOptions(COMMAND_CREATE, OPT_SECURE, OPT_DISABLE_SAVES, OPT_TAGS)
	info.BoundOptions(COMMAND_HELP, OPT_PAGER)
	info.BoundOptions(COMMAND_INFO, OPT_FORMAT, OPT_PAGER)
	info.BoundOptions(COMMAND_LIST, OPT_EXTRA, OPT_PAGER)
	info.BoundOptions(COMMAND_MEMORY, OPT_FORMAT)
	info.BoundOptions(COMMAND_REPLICATION, OPT_FORMAT)
	info.BoundOptions(COMMAND_SENTINEL_INFO, OPT_PAGER)
	info.BoundOptions(COMMAND_SETTINGS, OPT_TAGS, OPT_PAGER)
	info.BoundOptions(COMMAND_SLOWLOG_GET, OPT_PAGER)
	info.BoundOptions(COMMAND_STATS, OPT_FORMAT)
	info.BoundOptions(COMMAND_STATS_COMMAND, OPT_PAGER)
	info.BoundOptions(COMMAND_STATS_ERROR, OPT_PAGER)
	info.BoundOptions(COMMAND_STATS_LATENCY, OPT_PAGER)
	info.BoundOptions(COMMAND_TOP, OPT_PAGER)
	info.BoundOptions(COMMAND_TOP_DIFF, OPT_PAGER)

	return info
}

// genCompletion generates completion for different shells
func genCompletion() int {
	info := genUsage()

	switch options.GetS(OPT_COMPLETION) {
	case "bash":
		fmt.Print(bash.Generate(info, "rds"))
	case "fish":
		fmt.Print(fish.Generate(info, "rds"))
	case "zsh":
		fmt.Print(zsh.Generate(info, optMap, "rds"))
	default:
		return 1
	}

	return 0
}

// genManPage generates man page for app
func genManPage() {
	fmt.Println(man.Generate(genUsage(), genAbout("")))
}

// genAbout generates basic info about app
func genAbout(gitRev string) *usage.About {
	about := &usage.About{
		App:     APP,
		Version: VER,
		Release: CORE.VERSION,
		Desc:    DESC,
		Year:    2009,

		AppNameColorTag: colorTagApp,
		VersionColorTag: colorTagVer,
		ReleaseColorTag: colorTagRel,

		ReleaseSeparator: "/",
		DescSeparator:    "—",

		License:    "Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>",
		Owner:      "ESSENTIAL KAOS",
		BugTracker: "https://kaos.sh/rds",
	}

	if gitRev != "" {
		about.Build = "git:" + gitRev
		about.UpdateChecker = usage.UpdateChecker{"essentialkaos/rds", update.GitHubChecker}
	}

	return about
}
