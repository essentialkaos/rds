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
	"strconv"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/fmtutil/table"
	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/netutil"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/pluralize"
	"github.com/essentialkaos/ek/v12/sliceutil"
	"github.com/essentialkaos/ek/v12/terminal"
	"github.com/essentialkaos/ek/v12/timeutil"

	CORE "github.com/essentialkaos/rds/core"
	REDIS "github.com/essentialkaos/rds/redis"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// InfoCommand is "cli" command handler
func InfoCommand(args CommandArgs) int {
	var err error
	var sections []string

	format := options.GetS(OPT_FORMAT)

	if format == "" && useRawOutput {
		format = FORMAT_TEXT
	}

	if len(args) == 0 {
		renderInfoDataError(format, "You must define instance ID for this command")
		return EC_ERROR
	}

	id, _, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		renderInfoDataError(format, err.Error())
		return EC_ERROR
	}

	if !CORE.IsInstanceExist(id) {
		renderInfoDataError(format, fmt.Sprintf("Instance with ID %d does not exist", id))
		return EC_ERROR
	}

	if args.Has(1) {
		for _, section := range args[1:] {
			sections = append(sections, strings.ToLower(section))
		}
	}

	t := table.NewTable().SetSizes(33, 96)
	state, _ := CORE.GetInstanceState(id, true)

	if !state.IsWorks() {
		// Print basic instance info for stopped or dead instances
		if isInfoSectionRequired(sections, "instance") {
			showInstanceBasicInfo(t, id, nil, state)
			fmtutil.Separator(true)
			return EC_OK
		}

		renderInfoDataError(format, "Instance must work for executing this command")
		return EC_OK
	}

	info, err := CORE.GetInstanceInfo(id, 5*time.Second, sliceutil.Contains(sections, "all"))

	if err != nil {
		renderInfoDataError(format, err.Error())
		return EC_ERROR
	}

	if isInfoSectionRequired(sections, "instance") {
		if format == "" {
			showInstanceBasicInfo(t, id, info, state)
		}
	}

	switch format {
	case FORMAT_TEXT:
		renderInfoDataAsText(info, sections)
	case FORMAT_JSON:
		renderInfoDataAsJSON(info, sections)
	case FORMAT_XML:
		renderInfoDataAsXML(info, sections)
	default:
		renderInfoData(t, info, sections)
	}

	if format == "" {
		t.Separator()
	}

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// renderInfoDataError print error for different formats
func renderInfoDataError(format, message string) {
	switch format {
	case FORMAT_TEXT:
		renderInfoDataAsText(nil, nil)
	case FORMAT_JSON:
		renderInfoDataAsJSON(nil, nil)
	case FORMAT_XML:
		renderInfoDataAsXML(nil, nil)
	default:
		terminal.Error(message)
	}
}

// showInstanceBasicInfo print info about instance
func showInstanceBasicInfo(t *table.Table, id int, info *REDIS.Info, state CORE.State) {
	var (
		size      int64
		modTime   time.Time
		lastSave  time.Duration
		dumpExist bool
	)

	meta, err := CORE.GetInstanceMeta(id)

	if err != nil {
		return
	}

	host := CORE.Config.GetS(CORE.MAIN_HOSTNAME, netutil.GetIP())
	dumpFile := CORE.GetInstanceRDBPath(id)

	if fsutil.IsExist(dumpFile) {
		size = fsutil.GetSize(dumpFile)
		modTime, _ = fsutil.GetMTime(dumpFile)
		lastSave = time.Since(modTime)

		dumpExist = true
	}

	compatible := "Not checked"

	if meta.Compatible != "" {
		compatible = meta.Compatible
	}

	redisVersionInfo := ""
	currentRedisVer, err := CORE.GetRedisVersion()

	if err != nil && currentRedisVer.String() != "" {
		redisVersionInfo = "(current: " + currentRedisVer.String() + ")"
	}

	db := "0"

	if info != nil && len(info.Keyspace.Databases) != 0 {
		db = strconv.Itoa(info.Keyspace.Databases[0])
	}

	created := time.Unix(meta.Created, 0)

	uri := fmt.Sprintf(
		"redis://%s:%d/%s",
		host, CORE.GetInstancePort(id), db,
	)

	t.Separator()
	fmtc.Println(" ▾ {*}INSTANCE{!}")
	t.Separator()

	t.Print("ID", id)
	t.Print("Owner", getInstanceOwnerWithColor(meta, false))
	t.Print("Description", getInstanceDescWithTags(meta))
	t.Print("State", getInstanceStateWithColor(state))
	t.Print("Created", timeutil.Format(created, "%Y/%m/%d %H:%M:%S"))
	t.Print("Replication type", meta.ReplicationType)
	t.Print("URI", uri)
	t.Print("Compatibility", compatible+" {s-}"+redisVersionInfo+"{!}")

	if dumpExist {
		t.Print("Dump size", fmtutil.PrettySize(size))

		switch {
		case info != nil && info.Get("persistence", "rdb_bgsave_in_progress") == "1":
			t.Print("Last save", fmt.Sprintf("In progress {s-}(started %s ago){!}",
				formatLastSaveDate(info.GetI("persistence", "rdb_current_bgsave_time_sec")),
			))

		case info != nil && info.Get("persistence", "aof_rewrite_in_progress") == "1":
			t.Print("Last save", fmt.Sprintf("In progress {s-}(started %s ago){!}",
				formatLastSaveDate(info.GetI("persistence", "aof_current_rewrite_time_sec")),
			))

		default:
			t.Print("Last save", fmt.Sprintf("%s {s-}(%s ago)",
				timeutil.Format(modTime, "%Y/%m/%d %H:%M:%S"),
				formatLastSaveDate(int(lastSave/time.Second)),
			))
		}
	}
}

// renderInfoData print instance info
func renderInfoData(t *table.Table, info *REDIS.Info, sections []string) {
	if info == nil {
		return
	}

	for _, sectionName := range info.SectionNames {
		section := info.Sections[sectionName]
		sectionName = strings.ToLower(section.Header)

		if len(section.Fields) == 0 || !isInfoSectionRequired(sections, sectionName) {
			continue
		}

		t.Separator()
		fmtc.Printf(" ▾ {*}%s{!}\n", strings.ToUpper(section.Header))
		t.Separator()

		for _, v := range section.Fields {
			t.Print(v, section.Values[v])
		}
	}
}

// renderInfoDataAsText print info data in "key value" format
func renderInfoDataAsText(info *REDIS.Info, sections []string) {
	if info == nil {
		return
	}

	for _, sectionName := range info.SectionNames {
		section := info.Sections[sectionName]
		sectionName = strings.ToLower(section.Header)

		if len(section.Fields) == 0 || !isInfoSectionRequired(sections, sectionName) {
			continue
		}

		for _, v := range section.Fields {
			fmt.Printf("%s %s\n", v, section.Values[v])
		}
	}
}

// renderInfoDataAsJSON print info data as json
func renderInfoDataAsJSON(info *REDIS.Info, sections []string) {
	if info == nil {
		fmt.Println("{}")
		return
	}

	result := make(map[string]map[string]any)

	for _, sectionName := range info.SectionNames {
		section := info.Sections[sectionName]
		sectionName = strings.ToLower(section.Header)

		if len(section.Fields) == 0 || !isInfoSectionRequired(sections, sectionName) {
			continue
		}

		result[sectionName] = make(map[string]any)
		sectionResult := result[sectionName]

		for _, v := range section.Fields {
			sectionResult[v] = convertInfoValueType(section.Values[v])
		}
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")

	if err != nil {
		fmt.Println("{}")
		return
	}

	fmt.Println(string(jsonData))
}

// renderInfoDataAsJSON print info data as xml
func renderInfoDataAsXML(info *REDIS.Info, sections []string) {
	fmt.Println("<?xml version=\"1.0\" encoding=\"UTF-8\" ?>")

	if info == nil {
		fmt.Println("<info></info>")
		return
	}

	fmt.Println("<info>")

	for _, sectionName := range info.SectionNames {
		section := info.Sections[sectionName]
		sectionName = strings.ToLower(section.Header)

		if len(section.Fields) == 0 || !isInfoSectionRequired(sections, sectionName) {
			continue
		}

		fmt.Printf("  <%s>\n", sectionName)

		for _, v := range section.Fields {
			renderInfoPropXML(v, section.Values[v])
		}

		fmt.Printf("  </%s>\n", sectionName)
	}

	fmt.Println("</info>")
}

// renderInfoPropXML render info property as xml node
func renderInfoPropXML(name, value string) {
	fmt.Printf("    <%s>%v</%s>\n", name, convertInfoValueType(value), name)
}

// convertInfoValueType convert string info value to int, float or string
func convertInfoValueType(value string) any {
	var vi int
	var vf float64
	var err error

	if strings.Contains(value, ".") {
		vf, err = strconv.ParseFloat(value, 64)

		if err != nil {
			return value
		}

		return vf
	}

	vi, err = strconv.Atoi(value)

	if err != nil {
		return value
	}

	return vi
}

// isInfoSectionRequired return true if sections list contains given section or if
// sections list is empty
func isInfoSectionRequired(sections []string, section string) bool {
	if len(sections) == 0 || sliceutil.Contains(sections, "all") {
		return true
	}

	return sliceutil.Contains(sections, section)
}

// formatLastSaveDate format time since last save
func formatLastSaveDate(d int) string {
	switch {
	case d < 60:
		return pluralize.P("%d %s", d, "second", "seconds")
	case d < 3600:
		return pluralize.P("%d %s", d/60, "minute", "minutes")
	case d < 86400:
		return pluralize.P("%d %s", d/3600, "hour", "hours")
	default:
		return pluralize.P("%d %s", d/86400, "day", "days")
	}
}
