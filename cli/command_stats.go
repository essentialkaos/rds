package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"encoding/json"
	"fmt"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/mathutil"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// StatsCommand is "stats" command handler
func StatsCommand(args CommandArgs) int {
	if !CORE.HasInstances() {
		switch options.GetS(OPT_FORMAT) {
		case FORMAT_TEXT:
			fmt.Println("")
		case FORMAT_JSON:
			fmt.Println("{}")
		case FORMAT_XML:
			fmt.Sprintln(`<?xml version="1.0" encoding="UTF-8" ?>\n<stats></stats>`)
		default:
			terminal.Warn("No instances are created")
		}

		return EC_WARN
	}

	stats := CORE.GetStats()
	format := options.GetS(OPT_FORMAT)

	if format == "" && useRawOutput {
		format = FORMAT_TEXT
	}

	switch format {
	case FORMAT_TEXT:
		renderStatsAsText(stats)
	case FORMAT_JSON:
		renderStatsAsJSON(stats)
	case FORMAT_XML:
		renderStatsAsXML(stats)
	default:
		renderStats(stats)
	}

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// renderStats print stats
func renderStats(stats *CORE.Stats) {
	fmtutil.Separator(true)
	fmtc.Println(" ▾ {*}INSTANCES{!}")
	fmtutil.Separator(true)

	renderStatsValue("Total", stats.Instances.Total, false)
	renderStatsValue("Active", stats.Instances.Active, false)
	renderStatsValue("Dead", stats.Instances.Dead, false)
	renderStatsValue("Bg Save", stats.Instances.BgSave, false)
	renderStatsValue("AOF Rewrite", stats.Instances.AOFRewrite, false)
	renderStatsValue("Syncing", stats.Instances.Syncing, false)
	renderStatsValue("Save Failed", stats.Instances.SaveFailed, false)
	renderStatsValue("Active Master", stats.Instances.ActiveMaster, false)
	renderStatsValue("Active Replica", stats.Instances.ActiveReplica, false)
	renderStatsValue("Outdated", stats.Instances.Outdated, false)

	fmtutil.Separator(true)
	fmtc.Println(" ▾ {*}CLIENTS{!}")
	fmtutil.Separator(true)

	renderStatsValue("Connected", stats.Clients.Connected, false)
	renderStatsValue("Blocked", stats.Clients.Blocked, false)

	fmtutil.Separator(true)
	fmtc.Println(" ▾ {*}MEMORY{!}")
	fmtutil.Separator(true)

	renderStatsValue("Total System Memory", stats.Memory.TotalSystemMemory, true)
	fmtc.Printf(
		" %-26s {s}|{!} %s {s-}(%s ~ %s){!}\n", "System Memory",
		fmtutil.PrettyNum(stats.Memory.SystemMemory),
		fmtutil.PrettySize(stats.Memory.SystemMemory),
		fmtutil.PrettyPerc(mathutil.Perc(stats.Memory.SystemMemory, stats.Memory.TotalSystemMemory)),
	)

	switch stats.Memory.IsSwapEnabled {
	case true:
		renderStatsValue("Total System Swap", stats.Memory.TotalSystemSwap, true)
		fmtc.Printf(
			" %-26s {s}|{!} %s {s-}(%s ~ %s){!}\n", "System Swap",
			fmtutil.PrettyNum(stats.Memory.SystemSwap),
			fmtutil.PrettySize(stats.Memory.SystemSwap),
			fmtutil.PrettyPerc(mathutil.Perc(stats.Memory.SystemSwap, stats.Memory.TotalSystemSwap)),
		)
	default:
		fmtc.Printf(" %-26s {s}|{!} {s}—{!} {s-}(disabled){!}\n", "Total System Swap")
		fmtc.Printf(" %-26s {s}|{!} {s}—{!} {s-}(disabled){!}\n", "System Swap")
	}

	renderStatsValue("Used Memory", stats.Memory.UsedMemory, true)
	renderStatsValue("Used Memory (RSS)", stats.Memory.UsedMemoryRSS, true)
	renderStatsValue("Used Memory (Lua)", stats.Memory.UsedMemoryLua, true)

	switch stats.Memory.IsSwapEnabled {
	case true:
		renderStatsValue("Used Swap", stats.Memory.UsedSwap, true)
	default:
		fmtc.Printf(" %-26s {s}|{!} {s}—{!} {s-}(disabled){!}\n", "Used Swap")
	}

	fmtutil.Separator(true)
	fmtc.Println(" ▾ {*}STATS{!}")
	fmtutil.Separator(true)

	renderStatsValue("Total Connections Received", stats.Overall.TotalConnectionsReceived, false)
	renderStatsValue("Total Commands Processed", stats.Overall.TotalCommandsProcessed, false)
	renderStatsValue("Instantaneous Op/s", stats.Overall.InstantaneousOpsPerSec, false)
	renderStatsValue("Instantaneous Input Kb/s", stats.Overall.InstantaneousInputKbps, false)
	renderStatsValue("Instantaneous Output Kb/s", stats.Overall.InstantaneousOutputKbps, false)
	renderStatsValue("Rejected Connections", stats.Overall.RejectedConnections, false)
	renderStatsValue("Expired Keys", stats.Overall.ExpiredKeys, false)
	renderStatsValue("Evicted Keys", stats.Overall.EvictedKeys, false)
	renderStatsValue("Keyspace Hits", stats.Overall.KeyspaceHits, false)
	renderStatsValue("Keyspace Misses", stats.Overall.KeyspaceMisses, false)
	renderStatsValue("Pubsub Channels", stats.Overall.PubsubChannels, false)
	renderStatsValue("Pubsub Patterns", stats.Overall.PubsubPatterns, false)

	fmtutil.Separator(true)
	fmtc.Println(" ▾ {*}KEYS{!}")
	fmtutil.Separator(true)

	renderStatsValue("Total", stats.Keys.Total, false)
	renderStatsValue("Expires", stats.Keys.Expires, false)

	fmtutil.Separator(true)
}

// renderStatsAsText print stats data in "key value" format
func renderStatsAsText(stats *CORE.Stats) {
	fmt.Println("total_instances", stats.Instances.Total)
	fmt.Println("active_instances", stats.Instances.Active)
	fmt.Println("dead_instances", stats.Instances.Dead)
	fmt.Println("bgsave_instances", stats.Instances.BgSave)
	fmt.Println("aof_rewrite_instances", stats.Instances.AOFRewrite)
	fmt.Println("syncing_instances", stats.Instances.Syncing)
	fmt.Println("save_failed_instances", stats.Instances.SaveFailed)
	fmt.Println("active_master_instances", stats.Instances.ActiveMaster)
	fmt.Println("active_replica_instances", stats.Instances.ActiveReplica)
	fmt.Println("outdated_instances", stats.Instances.Outdated)
	fmt.Println("connected_clients", stats.Clients.Connected)
	fmt.Println("blocked_clients", stats.Clients.Blocked)
	fmt.Println("total_system_memory", stats.Memory.TotalSystemMemory)
	fmt.Println("system_memory", stats.Memory.SystemMemory)
	fmt.Println("total_system_swap", stats.Memory.TotalSystemSwap)
	fmt.Println("system_swap", stats.Memory.SystemSwap)
	fmt.Println("used_memory", stats.Memory.UsedMemory)
	fmt.Println("used_memory_rss", stats.Memory.UsedMemoryRSS)
	fmt.Println("used_memory_lua", stats.Memory.UsedMemoryLua)
	fmt.Println("used_swap", stats.Memory.UsedSwap)
	fmt.Println("total_connections_received", stats.Overall.TotalConnectionsReceived)
	fmt.Println("total_commands_processed", stats.Overall.TotalCommandsProcessed)
	fmt.Println("instantaneous_ops_per_sec", stats.Overall.InstantaneousOpsPerSec)
	fmt.Println("instantaneous_input_kbps", stats.Overall.InstantaneousInputKbps)
	fmt.Println("instantaneous_output_kbps", stats.Overall.InstantaneousOutputKbps)
	fmt.Println("rejected_connections", stats.Overall.RejectedConnections)
	fmt.Println("expired_keys", stats.Overall.ExpiredKeys)
	fmt.Println("evicted_keys", stats.Overall.EvictedKeys)
	fmt.Println("keyspace_hits", stats.Overall.KeyspaceHits)
	fmt.Println("keyspace_misses", stats.Overall.KeyspaceMisses)
	fmt.Println("pubsub_channels", stats.Overall.PubsubChannels)
	fmt.Println("pubsub_patterns", stats.Overall.PubsubPatterns)
	fmt.Println("total_keys", stats.Keys.Total)
	fmt.Println("expires_keys", stats.Keys.Expires)
}

// renderStatsAsXML print stats data as xml
func renderStatsAsXML(stats *CORE.Stats) {
	fmt.Println(`<?xml version="1.0" encoding="UTF-8" ?>`)
	fmt.Println("<stats>")

	fmt.Println("  <instances>")
	renderStatsAsXMLValue("total_instances", stats.Instances.Total)
	renderStatsAsXMLValue("active_instances", stats.Instances.Active)
	renderStatsAsXMLValue("dead_instances", stats.Instances.Dead)
	renderStatsAsXMLValue("bgsave_instances", stats.Instances.BgSave)
	renderStatsAsXMLValue("aof_rewrite_instances", stats.Instances.AOFRewrite)
	renderStatsAsXMLValue("syncing_instances", stats.Instances.Syncing)
	renderStatsAsXMLValue("save_failed_instances", stats.Instances.SaveFailed)
	renderStatsAsXMLValue("active_master_instances", stats.Instances.ActiveMaster)
	renderStatsAsXMLValue("active_replica_instances", stats.Instances.ActiveReplica)
	renderStatsAsXMLValue("outdated_instances", stats.Memory.UsedSwap)
	fmt.Println("  </instances>")

	fmt.Println("  <clients>")
	renderStatsAsXMLValue("connected_clients", stats.Clients.Connected)
	renderStatsAsXMLValue("blocked_clients", stats.Clients.Blocked)
	fmt.Println("  </clients>")

	fmt.Println("  <memory>")
	renderStatsAsXMLValue("total_system_memory", stats.Memory.TotalSystemMemory)
	renderStatsAsXMLValue("system_memory", stats.Memory.SystemMemory)
	renderStatsAsXMLValue("total_system_swap", stats.Memory.TotalSystemSwap)
	renderStatsAsXMLValue("system_swap", stats.Memory.SystemSwap)
	renderStatsAsXMLValue("used_memory", stats.Memory.UsedMemory)
	renderStatsAsXMLValue("used_memory_rss", stats.Memory.UsedMemoryRSS)
	renderStatsAsXMLValue("used_memory_lua", stats.Memory.UsedMemoryLua)
	renderStatsAsXMLValue("used_swap", stats.Memory.UsedSwap)
	fmt.Println("  </memory>")

	fmt.Println("  <overall>")
	renderStatsAsXMLValue("total_connections_received", stats.Overall.TotalConnectionsReceived)
	renderStatsAsXMLValue("total_commands_processed", stats.Overall.TotalCommandsProcessed)
	renderStatsAsXMLValue("instantaneous_ops_per_sec", stats.Overall.InstantaneousOpsPerSec)
	renderStatsAsXMLValue("instantaneous_input_kbps", stats.Overall.InstantaneousInputKbps)
	renderStatsAsXMLValue("instantaneous_output_kbps", stats.Overall.InstantaneousOutputKbps)
	renderStatsAsXMLValue("rejected_connections", stats.Overall.RejectedConnections)
	renderStatsAsXMLValue("expired_keys", stats.Overall.ExpiredKeys)
	renderStatsAsXMLValue("evicted_keys", stats.Overall.EvictedKeys)
	renderStatsAsXMLValue("keyspace_hits", stats.Overall.KeyspaceHits)
	renderStatsAsXMLValue("keyspace_misses", stats.Overall.KeyspaceMisses)
	renderStatsAsXMLValue("pubsub_channels", stats.Overall.PubsubChannels)
	renderStatsAsXMLValue("pubsub_patterns", stats.Overall.PubsubPatterns)
	fmt.Println("  </overall>")

	fmt.Println("  <keys>")
	renderStatsAsXMLValue("total_keys", stats.Keys.Total)
	renderStatsAsXMLValue("expires_keys", stats.Keys.Expires)
	fmt.Println("  </keys>")

	fmt.Println("</stats>")
}

// renderStatsAsJSON print stats data as json
func renderStatsAsJSON(stats *CORE.Stats) {
	jd, _ := json.MarshalIndent(stats, "", "  ")
	fmt.Println(string(jd))
}

// renderStatsAsXMLValue print stats property as xml node
func renderStatsAsXMLValue(name string, value uint64) {
	fmt.Printf("    <%s>%d</%s>\n", name, value, name)
}

// renderStatsValue print stats property for pretty output
func renderStatsValue(name string, value uint64, isSize bool) {
	if isSize {
		fmtc.Printf(" %-26s {s}|{!} %s {s-}(%s){!}\n", name, fmtutil.PrettyNum(value), fmtutil.PrettySize(value))
	} else {
		fmtc.Printf(" %-26s {s}|{!} %s\n", name, fmtutil.PrettyNum(value))
	}
}
