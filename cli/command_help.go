package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"strings"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/pager"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type helpInfoArgument struct {
	name     string
	desc     string
	optional bool
}

type helpInfoExample struct {
	command   string
	arguments string
	desc      string
}

type helpInfo struct {
	command   string
	desc      string
	arguments []helpInfoArgument
	options   []helpInfoArgument
	examples  []helpInfoExample
}

// ////////////////////////////////////////////////////////////////////////////////// //

// HelpCommand is "go" command handler
func HelpCommand(args CommandArgs) int {
	commandName := args.Get(0)
	commandList := map[string]func(){
		COMMAND_BACKUP_CREATE:        helpCommandBackupCreate,
		COMMAND_BACKUP_RESTORE:       helpCommandBackupRestore,
		COMMAND_BACKUP_CLEAN:         helpCommandBackupClean,
		COMMAND_BACKUP_LIST:          helpCommandBackupList,
		COMMAND_BATCH_CREATE:         helpCommandBatchCreate,
		COMMAND_BATCH_EDIT:           helpCommandBatchEdit,
		COMMAND_CHECK:                helpCommandCheck,
		COMMAND_CLI:                  helpCommandCli,
		COMMAND_CLIENTS:              helpCommandClients,
		COMMAND_CONF:                 helpCommandConf,
		COMMAND_CPU:                  helpCommandCPU,
		COMMAND_CREATE:               helpCommandCreate,
		COMMAND_DELETE:               helpCommandDestroy,
		COMMAND_DESTROY:              helpCommandDestroy,
		COMMAND_EDIT:                 helpCommandEdit,
		COMMAND_GEN_TOKEN:            helpCommandGenToken,
		COMMAND_GO:                   helpCommandGo,
		COMMAND_INFO:                 helpCommandInfo,
		COMMAND_INIT:                 helpCommandCreate,
		COMMAND_KILL:                 helpCommandKill,
		COMMAND_LIST:                 helpCommandList,
		COMMAND_MAINTENANCE:          helpCommandMaintenance,
		COMMAND_MEMORY:               helpCommandMemory,
		COMMAND_REGEN:                helpCommandRegen,
		COMMAND_RELEASE:              helpCommandDestroy,
		COMMAND_RELOAD:               helpCommandReload,
		COMMAND_REMOVE:               helpCommandDestroy,
		COMMAND_REPLICATION:          helpCommandReplication,
		COMMAND_REPLICATION_ROLE_SET: helpCommandReplicationRoleSet,
		COMMAND_RESTART:              helpCommandRestart,
		COMMAND_RESTART_ALL:          helpCommandRestartAll,
		COMMAND_RESTART_ALL_PROP:     helpCommandRestartAll,
		COMMAND_RESTART_PROP:         helpCommandRestart,
		COMMAND_SENTINEL_CHECK:       helpCommandSentinelCheck,
		COMMAND_SENTINEL_INFO:        helpCommandSentinelInfo,
		COMMAND_SENTINEL_MASTER:      helpCommandSentinelMaster,
		COMMAND_SENTINEL_RESET:       helpCommandSentinelReset,
		COMMAND_SENTINEL_START:       helpCommandSentinelStart,
		COMMAND_SENTINEL_STATUS:      helpCommandSentinelStatus,
		COMMAND_SENTINEL_STOP:        helpCommandSentinelStop,
		COMMAND_SENTINEL_SWITCH:      helpCommandSentinelSwitch,
		COMMAND_SETTINGS:             helpCommandSettings,
		COMMAND_SLOWLOG_GET:          helpCommandSlowlogGet,
		COMMAND_SLOWLOG_RESET:        helpCommandSlowlogReset,
		COMMAND_START:                helpCommandStart,
		COMMAND_START_ALL:            helpCommandStartAll,
		COMMAND_START_ALL_PROP:       helpCommandStartAll,
		COMMAND_START_PROP:           helpCommandStart,
		COMMAND_STATE_RESTORE:        helpCommandStateRestore,
		COMMAND_STATE_SAVE:           helpCommandStateSave,
		COMMAND_STATS:                helpCommandStats,
		COMMAND_STATS_COMMAND:        helpCommandStatsCommand,
		COMMAND_STATS_LATENCY:        helpCommandStatsLatency,
		COMMAND_STATS_ERROR:          helpCommandStatsError,
		COMMAND_STATUS:               helpCommandStatus,
		COMMAND_STOP:                 helpCommandStop,
		COMMAND_STOP_ALL:             helpCommandStopAll,
		COMMAND_STOP_ALL_PROP:        helpCommandStopAll,
		COMMAND_STOP_PROP:            helpCommandStop,
		COMMAND_TAG_ADD:              helpCommandTagAdd,
		COMMAND_TAG_REMOVE:           helpCommandTagRemove,
		COMMAND_TOP:                  helpCommandTop,
		COMMAND_TOP_DIFF:             helpCommandTopDiff,
		COMMAND_TOP_DUMP:             helpCommandTopDump,
		COMMAND_TRACK:                helpCommandTrack,
		COMMAND_VALIDATE_TEMPLATES:   helpCommandValidateTemplates,
	}

	helpFunc, hasInfo := commandList[commandName]

	if !hasInfo {
		terminal.Warn("Unknown command %s", commandName)
		return EC_ERROR
	}

	if options.GetB(OPT_PAGER) {
		if pager.Setup() == nil {
			defer pager.Complete()
		}
	}

	helpFunc()

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// helpCommandCreate prints info about "create" command usage
func helpCommandCreate() {
	helpInfo{
		command: COMMAND_CREATE,
		desc:    "Command read user input and create a new instance.",
		options: []helpInfoArgument{
			{getNiceOptions(OPT_TAGS), "List of tags", false},
			{getNiceOptions(OPT_SECURE), "Create instance with service ACL", false},
			{getNiceOptions(OPT_DISABLE_SAVES), "Disable saving for created instance", false},
		},
		examples: []helpInfoExample{
			{"", "", "Create new instance"},
			{"", "--disable-saves", "Create new instance with saves disabled"},
			{"", "--tags r:important,myapp", "Create new instance with tags"},
		},
	}.render()
}

// helpCommandDestroy prints info about "destroy" command usage
func helpCommandDestroy() {
	helpInfo{
		command: COMMAND_RELEASE,
		desc:    "Command to destroy the instance associated with the given ID. Command deletes all instance data, logs and configuration files on the master and all minions.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Destroy instance with ID 1"},
		},
	}.render()
}

// helpCommandEdit prints info about "edit" command usage
func helpCommandEdit() {
	helpInfo{
		command: COMMAND_EDIT,
		desc:    "This command allows you to change some information about the instance. At the moment you can change the owner, description and password.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Edit metadata for instance with ID 1"},
		},
	}.render()
}

// helpCommandBatchCreate prints info about "batch-create" command usage
func helpCommandBatchCreate() {
	info := helpInfo{
		command: COMMAND_BATCH_CREATE,
		arguments: []helpInfoArgument{
			{"csv-file", "CSV file with instances data", false},
		},
		examples: []helpInfoExample{
			{"", "instances.csv", "Create instances with data from instances.csv"},
		},
	}

	info.renderUsage()

	fmtc.Println("{*}Description{!}\n")
	fmtc.Println(`  This command allows you to create many instances at once. The CSV file must contain records in the next format:

  {m}owner;password;replication-type;auth-password;description{!}

  {*s@} example.csv {!}
  {s}┃{!}
  {s}┃ john;test1234!;replica;;Instance for John{!}
  {s}┃ bob;test1234!;replica;;Instance for Bob{!}
  {s}┃ bob;test1234!;replica;redisAuth1234;Instance for Bob with auth{!}
  {s}┃{!}
`)

	info.renderArguments()
	info.renderOptions()
	info.renderExamples()
}

// helpCommandBatchEdit prints info about "batch-edit" command usage
func helpCommandBatchEdit() {
	helpInfo{
		command: COMMAND_BATCH_EDIT,
		desc:    "This command allows you to change metadata for many instances at once.",
		arguments: []helpInfoArgument{
			{"id", "Instance ID list", false},
		},
		examples: []helpInfoExample{
			{"", "1 2 5-7 9", "Edit metadata for instance with IDs 1, 2, 5, 6, 7, 9"},
		},
	}.render()
}

// helpCommandStart prints info about "start" command usage
func helpCommandStart() {
	helpInfo{
		command: COMMAND_START,
		desc:    "Start (run) instance.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
		},
		examples: []helpInfoExample{
			{COMMAND_START, "1", "Start instance with ID 1 (only on master)"},
			{COMMAND_START_PROP, "1", "Start instance with ID 1 (on master and all minions)"},
		},
	}.render()
}

// helpCommandStop prints info about "stop" command usage
func helpCommandStop() {
	helpInfo{
		command: COMMAND_STOP,
		desc:    "Stop (shutdown) instance.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
			{"force", fmtc.Sprintf("Kill instance if it not stop after %d seconds", CORE.Config.GetI(CORE.DELAY_STOP)), true},
		},
		examples: []helpInfoExample{
			{COMMAND_STOP, "1", "Stop instance with ID 1 (only on master)"},
			{COMMAND_STOP, "1 force", "Force stop instance with ID 1 (only on master)"},
			{COMMAND_STOP_PROP, "1", "Stop instance with ID 1 (on master and all minions)"},
		},
	}.render()
}

// helpCommandKill prints info about "kill" command usage
func helpCommandKill() {
	helpInfo{
		command: COMMAND_KILL,
		desc:    "Kill (force shutdown) instance.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Kill instance with ID 1"},
		},
	}.render()
}

// helpCommandRestart prints info about "restart" command usage
func helpCommandRestart() {
	helpInfo{
		command: COMMAND_RESTART,
		desc:    "Restart (stop + start) instance.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
		},
		examples: []helpInfoExample{
			{COMMAND_RESTART, "1", "Restart instance with ID 1 (only on master)"},
			{COMMAND_RESTART_PROP, "1", "Restart instance with ID 1 (on master and all minions)"},
		},
	}.render()
}

// helpCommandStatus prints info about "status" command usage
func helpCommandStatus() {
	helpInfo{
		command: COMMAND_STATUS,
		desc:    "Check the current status of the instance (working/stopped).",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Show status of instance with ID 1"},
		},
	}.render()
}

// helpCommandCli prints info about "cli" command usage
func helpCommandCli() {
	helpInfo{
		command: COMMAND_CLI,
		desc:    "Run interactive shell or execute Redis command on some instance.",
		arguments: []helpInfoArgument{
			{"id:db", "Instance unique ID and database number", false},
			{"command", "Command", true},
		},
		options: []helpInfoArgument{
			{getNiceOptions(OPT_PRIVATE), "Enable \"private\" features (auto authentication & renamed commands support)", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Start interactive shell for instance with ID 1"},
			{"", "1 SET ABC 123", "Execute command \"SET ABC 123\" on instance 1"},
			{"", "1:10 SET ABC 123", "Execute command \"SET ABC 123\" on instance 1 and database 10"},
			{"", "-p 1 CONFIG SET databases 128", "Renamed \"CONFIG\" command usage example"},
		},
	}.render()
}

// helpCommandClients prints info about "clients" command usage
func helpCommandClients() {
	helpInfo{
		command: COMMAND_CLIENTS,
		desc:    "Show list of clients connected to the instance.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
			{"filter", "Clients filter", true},
		},
		options: []helpInfoArgument{
			{getNiceOptions(OPT_PAGER), "Enable pager for long output", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Show all clients connected to instance with ID 1"},
			{"", "1 test1", `Show all clients with name "test1" connected to instance with ID 1`},
			{"", "1 bob", `Show all clients with user "bob" connected to instance with ID 1`},
			{"", "1 192.168.1.123", `Show all clients connected to instance with ID 1 from 192.168.1.123`},
		},
	}.render()
}

// helpCommandCPU prints info about "cpu" command usage
func helpCommandCPU() {
	helpInfo{
		command: COMMAND_CPU,
		desc:    "Calculate instance CPU usage.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
			{"period", "Period for calculation in seconds (1-3600)", true},
		},
		examples: []helpInfoExample{
			{"", "1", "Calculate instance CPU for default 3 second period"},
			{"", "1 60", "Calculate instance CPU for 1 minute period"},
		},
	}.render()
}

// helpCommandInfo prints info about "info" command usage
func helpCommandInfo() {
	helpInfo{
		command: COMMAND_INFO,
		desc:    "Show info about Redis instance.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
			{"section…", "Info section", true},
		},
		options: []helpInfoArgument{
			{getNiceOptions(OPT_FORMAT), "Output format (json|text|xml)", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Show basic info about instance with ID 1"},
			{"", "1 memory cpu", "Show memory and cpu section of instance with ID 1"},
			{"", "1 all", "Show all info (including non-default sections) about instance with ID 1"},
		},
	}.render()
}

// helpCommandTrack prints info about "track" command usage
func helpCommandTrack() {
	helpInfo{
		command: COMMAND_TRACK,
		desc:    "Show interactive info about Redis instance.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
			{"interval", "Update interval in seconds (3 by default)", true},
		},
		examples: []helpInfoExample{
			{"", "1", "Show interactive info about instance 1"},
			{"", "1 10", "Show interactive info about instance 1 and update info every 10 seconds"},
		},
	}.render()
}

// helpCommandTagAdd prints info about "tag-add" command usage
func helpCommandTagAdd() {
	info := helpInfo{
		command: COMMAND_TAG_ADD,
		desc:    fmt.Sprintf("Add some tags (up to %d) to instance.", CORE.MAX_TAGS),
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
			{"tag", "Tag with or without color (color:tag)", false},
		},
		examples: []helpInfoExample{
			{"", "1 test", "Add tag \"test\" (grey) to instance 1"},
			{"", "1 red:test", "Add tag \"test\" (red) to instance 1"},
		},
	}

	info.renderUsage()
	info.renderDescription()
	info.renderArguments()

	fmtc.Println("  Avialalbe colors:\n")
	fmtc.Println("    {r}red{!} or {r}r{!}")
	fmtc.Println("    {g}green{!} or {g}g{!}")
	fmtc.Println("    {y}yellow{!} or {y}y{!}")
	fmtc.Println("    {b}blue{!} or {b}b{!}")
	fmtc.Println("    {c}cyan{!} or {c}c{!}")
	fmtc.Println("    {m}magenta{!} or {m}m{!}")
	fmtc.NewLine()

	info.renderExamples()
}

// helpCommandTagRemove prints info about "tag-remove" command usage
func helpCommandTagRemove() {
	helpInfo{
		command: COMMAND_TAG_REMOVE,
		desc:    "Remove a tag from the instance.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
			{"tag", "Tag", false},
		},
		examples: []helpInfoExample{
			{"", "1 test", "Remove tag \"test\" from instance 1"},
		},
	}.render()
}

// helpCommandConf prints info about "conf" command usage
func helpCommandConf() {
	helpInfo{
		command: COMMAND_CONF,
		desc:    "Print values from the configuration file and in-memory configuration.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
			{"filter…", "Property name filters", true},
		},
		options: []helpInfoArgument{
			{getNiceOptions(OPT_PAGER), "Enable pager for long output", false},
			{getNiceOptions(OPT_PRIVATE), "Show private info", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Show configuration of instance with ID 1"},
			{"", "1 append sync", "Show properties with 'append' or 'sync' in the name of instance with ID 1"},
		},
	}.render()
}

// helpCommandReload prints info about "reload" command usage
func helpCommandReload() {
	helpInfo{
		command: COMMAND_RELOAD,
		desc:    "Reload the configuration for one or all instances. Use this command if the configuration file has been updated. Use this command with care.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Reload configuration for instance with ID 1"},
			{"", "all", "Reload configuration for all instances"},
		},
	}.render()
}

// helpCommandList prints info about "list" command usage
func helpCommandList() {
	info := helpInfo{
		command: COMMAND_LIST,
		desc:    "Show list of all Redis instances.",
		arguments: []helpInfoArgument{
			{"filter…", "Listing filters", true},
		},
		options: []helpInfoArgument{
			{getNiceOptions(OPT_EXTRA), "Print extra info", false},
			{getNiceOptions(OPT_PAGER), "Enable pager for long output", false},
		},
		examples: []helpInfoExample{
			{"", "", "Show list of all instances"},
			{"", "--extra", "Show list of all instances with extra info"},
			{"", "my", "Show list of your instances"},
			{"", "bob active", "Show list of active instances owned by user bob"},
			{"", "bob active @staging", `Show list of active instances with tag "staging" owned by user bob`},
		},
	}

	info.renderUsage()
	info.renderDescription()
	info.renderArguments()

	fmtc.Println("  Avialalbe filter values:\n")
	fmtc.Printf("    {b}%-13s{!} %s\n", "my", "My (owned by current user) instances")
	fmtc.Printf("    {b}%-13s{!} %s\n", "works", "Working instances")
	fmtc.Printf("    {b}%-13s{!} %s\n", "stopped", "Stopped instances")
	fmtc.Printf("    {b}%-13s{!} %s\n", "idle", "Instances with no traffic (less than 5 commands per second)")
	fmtc.Printf("    {b}%-13s{!} %s\n", "active", "Instances with traffic")
	fmtc.Printf("    {b}%-13s{!} %s\n", "dead", "Dead instances")
	fmtc.Printf("    {b}%-13s{!} %s\n", "hang", "Hang instances")
	fmtc.Printf("    {b}%-13s{!} %s\n", "syncing", "Syncing instances")
	fmtc.Printf("    {b}%-13s{!} %s\n", "saving", "Saving instances")
	fmtc.Printf("    {b}%-13s{!} %s\n", "loading", "Loading instances")
	fmtc.Printf("    {b}%-13s{!} %s\n", "abandoned", "Instances with no traffic for a long time (≥ week)")
	fmtc.Printf("    {b}%-13s{!} %s\n", "master-up", "Instances with working syncing")
	fmtc.Printf("    {b}%-13s{!} %s\n", "master-down", "Instances with broken syncing")
	fmtc.Printf("    {b}%-13s{!} %s\n", "no-replica", "Instances with no replicas")
	fmtc.Printf("    {b}%-13s{!} %s\n", "with-replica", "Instances with one or more replicas")
	fmtc.Printf("    {b}%-13s{!} %s\n", "with-errors", "Instances with errors")
	fmtc.Printf("    {b}%-13s{!} %s\n", "outdated", "Instances which require restart for update")
	fmtc.Printf("    {b}%-13s{!} %s\n", "orphan", "Instances owned by deleted users")
	fmtc.Printf("    {b}%-13s{!} %s\n", "standby", "Instances with standby replication")
	fmtc.Printf("    {b}%-13s{!} %s\n", "replica", "Instances with real replicas")
	fmtc.Printf("    {b}%-13s{!} %s\n", "secure", "Instances with enabled authentication")
	fmtc.Printf("    {b}%-13s{!} %s\n", "@{tag}", "Instances tagged by given tag")
	fmtc.Printf("    {b}%-13s{!} %s\n", "{username}", "Instances owned by given user")
	fmtc.NewLine()
	fmtc.Println("  If no instances are found using the property search, rds will try to find instances using")
	fmtc.Println("  a full-text search of instance descriptions.")
	fmtc.NewLine()

	info.renderOptions()
	info.renderExamples()
}

// helpCommandMaintenance prints info about "maintenance" command usage
func helpCommandMaintenance() {
	helpInfo{
		command: COMMAND_MAINTENANCE,
		desc:    "Enable or disable maintenance mode.",
		arguments: []helpInfoArgument{
			{"flag", "Maintenance mode flag (true/false or yes/no or enable/disable)", false},
		},
		examples: []helpInfoExample{
			{"", "enable", "Enable maintenance mode"},
			{"", "no", "Disable maintenance mode"},
		},
	}.render()
}

// helpCommandMemory prints info about "memory" command usage
func helpCommandMemory() {
	helpInfo{
		command: COMMAND_MEMORY,
		desc:    "Show detailed information about instance memory usage.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
		},
		options: []helpInfoArgument{
			{getNiceOptions(OPT_PAGER), "Enable pager for long output", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Show memory usage of instance with ID 1"},
		},
	}.render()
}

// helpCommandStats prints info about "stats" command usage
func helpCommandStats() {
	helpInfo{
		command: COMMAND_STATS,
		desc:    "Show overall statistics.",
		options: []helpInfoArgument{
			{getNiceOptions(OPT_FORMAT), "Output format (json|text|xml)", false},
		},
		examples: []helpInfoExample{
			{"", "", "Show all available statistics info"},
		},
	}.render()
}

// helpCommandStatsCommand prints info about "stats-command" command usage
func helpCommandStatsCommand() {
	helpInfo{
		command: COMMAND_STATS_COMMAND,
		desc:    "Show statistics based on the command type.",
		options: []helpInfoArgument{
			{getNiceOptions(OPT_PAGER), "Enable pager for long output", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Show statistics of instance with ID 1"},
		},
	}.render()
}

// helpCommandStatsLatency prints info about "stats-command" command usage
func helpCommandStatsLatency() {
	helpInfo{
		command: COMMAND_STATS_LATENCY,
		desc:    "Show latency statistics based on the command type.",
		options: []helpInfoArgument{
			{getNiceOptions(OPT_PAGER), "Enable pager for long output", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Show statistics of instance with ID 1"},
		},
	}.render()
}

// helpCommandStatsError prints info about "stats-command" command usage
func helpCommandStatsError() {
	helpInfo{
		command: COMMAND_STATS_ERROR,
		desc:    "Show error statistics.",
		options: []helpInfoArgument{
			{getNiceOptions(OPT_PAGER), "Enable pager for long output", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Show statistics of instance with ID 1"},
		},
	}.render()
}

// helpCommandTop prints info about "top" command usage
func helpCommandTop() {
	helpInfo{
		command: COMMAND_TOP,
		desc:    "Show top for any field available in the INFO command output. Command without arguments show top 10 by memory usage. Also this command can calculate CPU usage (\"cpu\", \"cpu_children\", \"cpu_sys\", \"cpu_user\", \"cpu_sys_children\", \"cpu_user_children\")",
		arguments: []helpInfoArgument{
			{"field", "Field  name", true},
			{"num", "Number of results", true},
		},
		options: []helpInfoArgument{
			{getNiceOptions(OPT_PAGER), "Enable pager for long output", false},
		},
		examples: []helpInfoExample{
			{"", "", "Show top 10 by memory usage"},
			{"", "- 20", "Show top 20 by memory usage"},
			{"", "connected_clients 5", "Show top 5 instances by connected clients"},
			{"", "^connected_clients 5", "Show last 5 instances by connected clients"},
			{"", "cpu", "Show top 10 by CPU usage for 5 sec"},
		},
	}.render()
}

// helpCommandTopDump prints info about "top-dump" command usage
func helpCommandTopDump() {
	helpInfo{
		command: COMMAND_TOP_DUMP,
		desc:    "Dump top data to file. The output file must have a .gz extension (all data saved as a gzipped JSON file) and must not exist before saving. Date control sequences can be used for the output name (see 'man date').",
		arguments: []helpInfoArgument{
			{"file", "Output file", false},
		},
		examples: []helpInfoExample{
			{"", "rds-top.gz", "Save top data to file rds-top.gz"},
			{"", `rds-top-%Y-%m-%d-%H%M.gz`, "Save top data to file with current date and time as part of the name"},
		},
	}.render()
}

// helpCommandTopDiff prints info about "top-diff" command usage
func helpCommandTopDiff() {
	helpInfo{
		command: COMMAND_TOP_DIFF,
		desc:    "Show the difference between current and dumped top data.",
		arguments: []helpInfoArgument{
			{"file", "Dump file", false},
			{"field", "Field  name", true},
			{"num", "Number of results", true},
		},
		options: []helpInfoArgument{
			{getNiceOptions(OPT_PAGER), "Enable pager for long output", false},
		},
		examples: []helpInfoExample{
			{"", "rds-top.gz", "Compare data and show top 10 by increased memory usage"},
			{"", "rds-top.gz - 5", "Compare data and show top 5 by increased memory usage"},
			{"", "rds-top.gz ^connected_clients 5", "Compare data and show top 5 by decreased number of connected clients"},
		},
	}.render()
}

// helpCommandSettings prints info about "settings" command usage
func helpCommandSettings() {
	helpInfo{
		command: COMMAND_SETTINGS,
		desc:    "Show settings from global configuration file.",
		options: []helpInfoArgument{
			{getNiceOptions(OPT_PAGER), "Enable pager for long output", false},
			{getNiceOptions(OPT_PRIVATE), "Show private info", false},
		},
		arguments: []helpInfoArgument{
			{"section…", "Settings option section and name", true},
		},
		examples: []helpInfoExample{
			{"", "", "Show all settings"},
			{"", "replcation", "Show specific settings section"},
		},
	}.render()
}

// helpCommandSlowlogGet prints info about "slowlog-get" command usage
func helpCommandSlowlogGet() {
	helpInfo{
		command: COMMAND_SLOWLOG_GET,
		desc:    "Show last entries from the slow log.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
			{"num", "Number of results", true},
		},
		options: []helpInfoArgument{
			{getNiceOptions(OPT_PAGER), "Enable pager for long output", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Show last 10 entries from instance 1 slow log"},
			{"", "1 30", "Show last 30 entries from instance 1 slow log"},
		},
	}.render()
}

// helpCommandSlowlogReset prints info about "slowlog-reset" command usage
func helpCommandSlowlogReset() {
	helpInfo{
		command: COMMAND_SLOWLOG_RESET,
		desc:    "Clear slow log.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Clear instance 1 slow log"},
		},
	}.render()
}

// helpCommandCheck prints info about "check" command usage
func helpCommandCheck() {
	helpInfo{
		command: COMMAND_CHECK,
		desc:    "Check for dead instances. The utility would return a non-zero exit code if dead instances were found.",
		examples: []helpInfoExample{
			{"", "", "Check for dead instances"},
		},
	}.render()
}

// helpCommandGo prints info about "go" command usage
func helpCommandGo() {
	helpInfo{
		command: COMMAND_GO,
		desc:    "Generate password for superuser.",
		examples: []helpInfoExample{
			{"", "", "Generate password for superuser"},
		},
	}.render()
}

// helpCommandStartAll prints info about "start-all" command usage
func helpCommandStartAll() {
	helpInfo{
		command: COMMAND_START_ALL,
		desc:    "Start all stopped instances.",
		examples: []helpInfoExample{
			{COMMAND_START_ALL, "", "Start all stopped instances (only on master)"},
			{COMMAND_START_ALL_PROP, "", "Start all stopped instances (on master and all minions)"},
		},
	}.render()
}

// helpCommandStopAll prints info about "stop-all" command usage
func helpCommandStopAll() {
	helpInfo{
		command: COMMAND_STOP_ALL,
		desc:    "Stop all working instances.",
		examples: []helpInfoExample{
			{COMMAND_STOP_ALL, "", "Stop all working instances (only on master)"},
			{COMMAND_STOP_ALL_PROP, "", "Stop all working instances (on master and all minions)"},
		},
	}.render()
}

// helpCommandRestartAll prints info about "restart-all" command usage
func helpCommandRestartAll() {
	helpInfo{
		command: COMMAND_RESTART_ALL,
		desc:    "Restart (stop + start) all working instances.",
		examples: []helpInfoExample{
			{COMMAND_RESTART_ALL, "", "Restart all working instances (only on master)"},
			{COMMAND_RESTART_ALL_PROP, "", "Restart all working instances (on master and all minions)"},
		},
	}.render()
}

// helpCommandStateSave prints info about "state-save" command usage
func helpCommandStateSave() {
	helpInfo{
		command: COMMAND_STATE_SAVE,
		desc:    "Save states of all instances to file.",
		arguments: []helpInfoArgument{
			{"file", "Path to file", false},
		},
		examples: []helpInfoExample{
			{"", "states.dat", "Save states to file states.dat"},
		},
	}.render()
}

// helpCommandStateRestore prints info about "state-restore" command usage
func helpCommandStateRestore() {
	helpInfo{
		command: COMMAND_STATE_RESTORE,
		desc:    "Read states from a file and restore states of all instances.",
		arguments: []helpInfoArgument{
			{"file", "Path to file", true},
		},
		examples: []helpInfoExample{
			{"", "", "Restore state from internal state file (useful after server reboot)"},
			{"", "states.dat", "Restore states saved into states.dat"},
		},
	}.render()
}

// helpCommandRegen prints info about "regen" command usage
func helpCommandRegen() {
	helpInfo{
		command: COMMAND_REGEN,
		desc:    "Regenerate the configuration file for one or all instances. Use this command if the configuration template has been updated. Use this command with care.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Regenerate configuration file for instance with ID 1"},
			{"", "all", "Regenerate configuration files for all instances"},
		},
	}.render()
}

// helpCommandReplication prints info about "replication" command usage
func helpCommandReplication() {
	helpInfo{
		command: COMMAND_REPLICATION,
		desc:    "Show information about RDS replication with other nodes in the cluster.",
		options: []helpInfoArgument{
			{getNiceOptions(OPT_FORMAT), "Output format (json|text|xml)", false},
		},
		examples: []helpInfoExample{
			{"", "", "Show info about master, minions and sentinel nodes"},
		},
	}.render()
}

func helpCommandReplicationRoleSet() {
	helpInfo{
		command: COMMAND_REPLICATION_ROLE_SET,
		desc:    "Change node role.",
		arguments: []helpInfoArgument{
			{"target-role", fmt.Sprintf("Target role (%s/%s)", CORE.ROLE_MASTER, CORE.ROLE_MINION), false},
		},
		examples: []helpInfoExample{
			{"", "master", "Change node role from minion to master"},
			{"", "minion", "Change node role from master to minion"},
		},
	}.render()
}

// helpCommandSentinelStart prints info about "sentinel-start" command usage
func helpCommandSentinelStart() {
	helpInfo{
		command: COMMAND_SENTINEL_START,
		desc:    "Start Redis Sentinel daemon.",
		examples: []helpInfoExample{
			{"", "", "Start Redis Sentinel daemon"},
		},
	}.render()
}

// helpCommandSentinelStop prints info about "sentinel-stop" command usage
func helpCommandSentinelStop() {
	helpInfo{
		command: COMMAND_SENTINEL_STOP,
		desc:    "Stop Redis Sentinel daemon.",
		examples: []helpInfoExample{
			{"", "", "Stop Redis Sentinel daemon"},
		},
	}.render()
}

// helpCommandSentinelStatus prints info about "sentinel-status" command usage
func helpCommandSentinelStatus() {
	helpInfo{
		command: COMMAND_SENTINEL_STATUS,
		desc:    "Show status of Redis Sentinel daemon.",
		examples: []helpInfoExample{
			{"", "", "Show status of Redis Sentinel daemon"},
		},
	}.render()
}

// helpCommandSentinelInfo prints info about "sentinel-info" command usage
func helpCommandSentinelInfo() {
	helpInfo{
		command: COMMAND_SENTINEL_INFO,
		desc:    "Show info from Sentinel for some instance.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
		},
		options: []helpInfoArgument{
			{getNiceOptions(OPT_PAGER), "Enable pager for long output", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Show info from Sentinel for instance with ID 1"},
		},
	}.render()
}

// helpCommandSentinelMaster prints info about "sentinel-master" command usage
func helpCommandSentinelMaster() {
	helpInfo{
		command: COMMAND_SENTINEL_MASTER,
		desc:    "Show IP of master instance.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Show IP of master instance with ID 1"},
		},
	}.render()
}

// helpCommandSentinelCheck prints info about "sentinel-check" command usage
func helpCommandSentinelCheck() {
	helpInfo{
		command: COMMAND_SENTINEL_CHECK,
		desc:    "Check Sentinel configuration.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Check if Sentinel configuration is ok for instance with ID 1"},
		},
	}.render()
}

// helpCommandSentinelCheck prints info about "sentinel-switch-master" command usage
func helpCommandSentinelSwitch() {
	helpInfo{
		command: COMMAND_SENTINEL_SWITCH,
		desc:    "Switch instance to master role.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
		},
		examples: []helpInfoExample{
			{"", "1", "Switch instance 1 to master role"},
		},
	}.render()
}

// helpCommandSentinelReset prints info about "sentinel-reset" command usage
func helpCommandSentinelReset() {
	helpInfo{
		command: COMMAND_SENTINEL_RESET,
		desc:    "Reset state in Sentinel for all instances.",
		examples: []helpInfoExample{
			{"", "", "Reset state for all instances"},
		},
	}.render()
}

// helpCommandGenToken prints info about "gen-token" command usage
func helpCommandGenToken() {
	helpInfo{
		command: COMMAND_GEN_TOKEN,
		desc:    "Generate authentication token for sync daemon.",
		examples: []helpInfoExample{
			{"", "", "Generate random token"},
		},
	}.render()
}

// helpCommandValidateTemplates prints info about "validate-templates" command usage
func helpCommandValidateTemplates() {
	helpInfo{
		command: COMMAND_VALIDATE_TEMPLATES,
		desc:    "Validate Redis and Sentinel configuration file templates.",
		examples: []helpInfoExample{
			{"", "", "Validate templates"},
		},
	}.render()
}

// helpCommandBackupCreate prints info about "backup-create" command usage
func helpCommandBackupCreate() {
	helpInfo{
		command: COMMAND_BACKUP_CREATE,
		desc:    "Create snapshot of RDB file.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
		},
		examples: []helpInfoExample{
			{"", "7", "Create an RDB file snapshot of the instance with the ID 7"},
		},
	}.render()
}

// helpCommandBackupRestore prints info about "backup-restore" command usage
func helpCommandBackupRestore() {
	helpInfo{
		command: COMMAND_BACKUP_RESTORE,
		desc:    "Restore previously created snapshot of instance data.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
		},
		examples: []helpInfoExample{
			{"", "7", "Restore a data snapshot of the instance with the ID 7"},
		},
	}.render()
}

// helpCommandBackupClean prints info about "backup-clean" command usage
func helpCommandBackupClean() {
	helpInfo{
		command: COMMAND_BACKUP_CLEAN,
		desc:    "Delete all backup snapshots.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
		},
		examples: []helpInfoExample{
			{"", "7", "Delete all RDB snapshots of the instance with the ID 7"},
		},
	}.render()
}

// helpCommandBackupList prints info about "backup-list" command usage
func helpCommandBackupList() {
	helpInfo{
		command: COMMAND_BACKUP_LIST,
		desc:    "Show information regarding all backup snapshots.",
		arguments: []helpInfoArgument{
			{"id", "Instance unique ID", false},
		},
		examples: []helpInfoExample{
			{"", "7", "Show information about all snapshots of the instance with the ID 7"},
		},
	}.render()
}

// ////////////////////////////////////////////////////////////////////////////////// //

// getNiceOptions parse option and return formatted string
func getNiceOptions(name string) string {
	long, short := options.ParseOptionName(name)

	if short != "" {
		return fmtc.Sprintf("--%s, -%s", long, short)
	}

	return fmtc.Sprintf("--%s", long)
}

// getArgumentFormatting return argument name formatting string
func getArgumentFormatting(data []helpInfoArgument) string {
	var maxSize int

	for _, argument := range data {
		argumentLen := len(argument.name)

		if argumentLen > maxSize {
			maxSize = argumentLen
		}
	}

	return fmtc.Sprintf("%%-%ds", maxSize+1)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// render render all info
func (i helpInfo) render() {
	i.renderUsage()
	i.renderDescription()
	i.renderArguments()
	i.renderOptions()
	i.renderExamples()
}

// renderUsage render usage info
func (i helpInfo) renderUsage() {
	fmtc.Println("{*}Usage{!}\n")

	if len(i.arguments) == 0 {
		fmtc.Printf("  rds %s\n\n", i.command)
		return
	}

	var arguments []string

	for _, cmdArg := range i.arguments {
		arguments = append(arguments, cmdArg.name)
	}

	fmtc.Printf("  rds {y}%s{!} {#45}%s{!}\n\n", i.command, strings.Join(arguments, " "))
}

// renderDescription render description
func (i helpInfo) renderDescription() {
	fmtc.Println("{*}Description{!}\n")

	fmtc.Println(fmtutil.Wrap(i.desc, "  ", 88))
	fmtc.NewLine()
}

// renderArguments render arguments
func (i helpInfo) renderArguments() {
	if len(i.arguments) == 0 {
		return
	}

	fmtc.Println("{*}Arguments{!}\n")

	fmtStr := getArgumentFormatting(i.arguments)

	for _, argument := range i.arguments {
		fmtc.Printf("  {#45}"+fmtStr+"{!} %s", argument.name, argument.desc)

		if argument.optional {
			fmtc.Printf(" {s-}(optional){!}")
		}

		fmtc.NewLine()
	}

	fmtc.NewLine()
}

// renderOptions render options
func (i helpInfo) renderOptions() {
	if len(i.options) == 0 {
		return
	}

	fmtc.Println("{*}Options{!}\n")

	fmtStr := getArgumentFormatting(i.options)

	for _, option := range i.options {
		fmtc.Printf("  {g}"+fmtStr+"{!} %s\n", option.name, option.desc)
	}

	fmtc.NewLine()
}

// renderExamples render examples
func (i helpInfo) renderExamples() {
	if len(i.examples) == 0 {
		return
	}

	fmtc.Println("{*}Examples{!}\n")

	for index, example := range i.examples {
		if example.command == "" {
			fmtc.Printf("  rds %s\n", strings.Join([]string{i.command, example.arguments}, " "))
		} else {
			fmtc.Printf("  rds %s\n", strings.Join([]string{example.command, example.arguments}, " "))
		}

		fmtc.Printf("  {s-}%s{!}\n", example.desc)

		if index+1 != len(i.examples) {
			fmtc.NewLine()
		}
	}
}
