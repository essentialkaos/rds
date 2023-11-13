package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/essentialkaos/ek/v12/fmtutil/table"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/strutil"
	"github.com/essentialkaos/ek/v12/terminal"
	"github.com/essentialkaos/ek/v12/timeutil"

	API "github.com/essentialkaos/rds/api"
	CORE "github.com/essentialkaos/rds/core"
	SC "github.com/essentialkaos/rds/sync/client"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// ReplicationCommand is "replication" command handler
func ReplicationCommand(args CommandArgs) int {
	format := options.GetS(OPT_FORMAT)

	if !CORE.IsSyncDaemonActive() {
		switch format {
		case FORMAT_TEXT, FORMAT_JSON, FORMAT_XML:
			fmt.Print(formatReplicationErrorMessage(format))
		default:
			terminal.Warn("Can't show replication info: sync daemon is not working")
		}

		return EC_WARN
	}

	info, err := SC.GetReplicationInfo()

	if err != nil {
		switch format {
		case FORMAT_TEXT, FORMAT_JSON, FORMAT_XML:
			fmt.Print(formatReplicationErrorMessage(format))
		default:
			terminal.Error(err)
		}

		return EC_ERROR
	}

	if format == "" && useRawOutput {
		format = FORMAT_TEXT
	}

	switch format {
	case FORMAT_TEXT:
		renderReplicationInfoText(info)
	case FORMAT_JSON:
		renderReplicationInfoJSON(info)
	case FORMAT_XML:
		renderReplicationInfoXML(info)
	default:
		renderReplicationInfo(info)
	}

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// renderReplicationInfo print info about master and clients
func renderReplicationInfo(info *API.ReplicationInfo) {
	t := table.NewTable("CID", "ROLE", "STATE", "VERSION", "LAG", "HOST")

	t.SetSizes(8, 12, 14, 10, 8, 8)
	t.SetAlignments(
		table.ALIGN_RIGHT,
		table.ALIGN_RIGHT,
		table.ALIGN_RIGHT,
		table.ALIGN_RIGHT,
		table.ALIGN_RIGHT,
	)

	printSyncMasterInfo(t, info.Master, info.SuppliantCID)

	if len(info.Clients) == 0 {
		return
	}

	for _, client := range info.Clients {
		if client.Role == CORE.ROLE_MINION {
			printSyncClientInfo(t, client, info.SuppliantCID)
		}
	}

	if hasSentinelNodes(info.Clients) {
		t.Separator()
		for _, client := range info.Clients {
			if client.Role == CORE.ROLE_SENTINEL {
				printSyncClientInfo(t, client, info.SuppliantCID)
			}
		}
	}

	t.Separator()
}

// printSyncMasterInfo print info about sync RDS Sync master
func printSyncMasterInfo(t *table.Table, master *API.MasterInfo, suppliantCID string) {
	isSuppliant := suppliantCID == ""

	t.Print(
		"{s-}--------{!}", getSyncClientRole("master", isSuppliant), "{g}online{!}",
		getColoredVersion(master.Version), "{s-}—{!}", getSyncClientHost(master.Hostname, master.IP),
	).Separator()
}

// printSyncClientInfo print info about RDS Sync client
func printSyncClientInfo(t *table.Table, client *API.ClientInfo, suppliantCID string) {
	isSuppliant := client.CID == suppliantCID
	lag := "{s-}—{!}"

	if client.LastSyncLag > 0 {
		lag = timeutil.MiniDuration(timeutil.SecondsToDuration(client.LastSeenLag))
	}

	t.Print(
		client.CID, getSyncClientRole(client.Role, isSuppliant),
		getSyncClientState(client.State), getColoredVersion(client.Version),
		lag, getSyncClientHost(client.Hostname, client.IP),
	)
}

// getColoredVersion returns colored version info
func getColoredVersion(version string) string {
	app, core, ok := strings.Cut(version, "/")

	if !ok {
		return version
	}

	return fmt.Sprintf("%s{s}/%s{!}", app, core)
}

// getSyncClientState returns client state for command output
func getSyncClientState(state API.ClientState) string {
	switch state {
	case API.STATE_SYNCING:
		return "{c}syncing{!}"
	case API.STATE_ONLINE:
		return "{g}online{!}"
	case API.STATE_POSSIBLE_DOWN:
		return "{y}possibly down{!}"
	default:
		return "{r}down{!}"
	}
}

// getClientHost returns client hostname for command output
func getSyncClientHost(hostname, ip string) string {
	var result string

	switch hostname {
	case "":
		result = ip
	default:
		result = hostname
	}

	if hostname != "" {
		result += " {s-}(" + ip + "){!}"
	}

	return result
}

// getSyncClientRole returns client role for command output
func getSyncClientRole(typ string, isSuppliant bool) string {
	if !isSuppliant {
		return typ
	}

	return "{s}•{!} " + typ
}

// hasSentinelNodes returns true if given slice contains sentinel node
func hasSentinelNodes(clients []*API.ClientInfo) bool {
	for _, client := range clients {
		if client.Role == CORE.ROLE_SENTINEL {
			return true
		}
	}

	return false
}

// formatReplicationErrorMessage returns error message data
func formatReplicationErrorMessage(format string) string {
	switch format {
	case FORMAT_TEXT:
		return fmt.Sprint("")
	case FORMAT_JSON:
		return fmt.Sprint("{}\n")
	case FORMAT_XML:
		return fmt.Sprint("<?xml version=\"1.0\" encoding=\"UTF-8\" ?>\n<replication></replication>\n")
	}

	return ""
}

// renderReplicationInfoText prints replication info in text format
func renderReplicationInfoText(info *API.ReplicationInfo) {
	fmt.Printf(
		"00000000 master %s %s %s online 0 0 0\n",
		info.Master.IP, info.Master.Hostname,
		strutil.Exclude(info.Master.Version, " "),
	)

	for _, c := range info.Clients {
		fmt.Printf(
			"%s %s %s %s %s %s %g %g %d\n",
			c.CID, c.Role, c.IP, c.Hostname, strutil.Exclude(c.Version, " "),
			c.State, c.LastSeenLag, c.LastSyncLag, c.ConnectionDate,
		)
	}
}

// renderReplicationInfoXML prints replication info in XML format
func renderReplicationInfoXML(info *API.ReplicationInfo) {
	fmt.Println(`<?xml version="1.0" encoding="UTF-8" ?>`)
	fmt.Println("<replication>")

	fmt.Printf(
		"  <master ip=\"%s\" hostname=\"%s\" version=\"%s\" />\n",
		info.Master.IP, info.Master.Hostname, info.Master.Version,
	)

	fmt.Println("  <clients>")

	for _, c := range info.Clients {
		fmt.Printf(
			"    <client cid=\"%s\" role=\"%s\" ip=\"%s\" hostname=\"%s\" version=\"%s\" state=\"%s\" seen-lag=\"%g\" sync-lag=\"%g\" connected=\"%d\" />\n",
			c.CID, c.Role, c.IP, c.Hostname, c.Version, c.State,
			c.LastSeenLag, c.LastSyncLag, c.ConnectionDate,
		)
	}

	fmt.Println("  </clients>")
	fmt.Println("</replication>")
}

// renderReplicationInfoJSON prints replication info in JSON format
func renderReplicationInfoJSON(info *API.ReplicationInfo) {
	jd, _ := json.MarshalIndent(info, "", "  ")
	fmt.Println(string(jd))
}
