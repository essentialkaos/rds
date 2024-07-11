package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fmtutil"
	"github.com/essentialkaos/ek/v13/fmtutil/table"
	"github.com/essentialkaos/ek/v13/fsutil"
	"github.com/essentialkaos/ek/v13/jsonutil"
	"github.com/essentialkaos/ek/v13/mathutil"
	"github.com/essentialkaos/ek/v13/options"
	"github.com/essentialkaos/ek/v13/pager"
	"github.com/essentialkaos/ek/v13/strutil"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/timeutil"

	CORE "github.com/essentialkaos/rds/core"
	REDIS "github.com/essentialkaos/rds/redis"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type topItem struct {
	ID      int
	Value   float64
	IsFloat bool
}

type topItems []topItem

type topDump struct {
	Data []*topDumpItem `json:"data"`
}

type topDumpItem struct {
	ID   int         `json:"id"`
	Info [][2]string `json:"info"`
}

// ////////////////////////////////////////////////////////////////////////////////// //

func (s topItems) Len() int           { return len(s) }
func (s topItems) Less(i, j int) bool { return s[i].Value > s[j].Value }
func (s topItems) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// ////////////////////////////////////////////////////////////////////////////////// //

// TopCommand is "top" command handler
func TopCommand(args CommandArgs) int {
	if !CORE.HasInstances() {
		terminal.Warn("No instances are created")
		return EC_WARN
	}

	field, resultNum, reverse := parseTopCommandArguments(args)
	items, err := collectTopData(field)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	if len(items) == 0 {
		terminal.Warn("All instances are stopped")
		return EC_OK
	}

	if reverse {
		sort.Sort(sort.Reverse(items))
	} else {
		sort.Sort(items)
	}

	if (options.GetB(OPT_PAGER) || prefs.AutoPaging) && !useRawOutput {
		if pager.Setup() == nil {
			defer pager.Complete()
		}
	}

	printTopInfo(items, resultNum, false)

	return EC_OK
}

// TopDumpCommand is "top-dump" command handler
func TopDumpCommand(args CommandArgs) int {
	if !CORE.HasInstances() {
		terminal.Warn("No instances are created")
		return EC_WARN
	}

	if !args.Has(0) {
		terminal.Warn("You must define output file")
		return EC_WARN
	}

	dump := collectTopDump()

	if len(dump.Data) == 0 {
		terminal.Warn("All instances are stopped")
		return EC_OK
	}

	output := args.Get(0)

	if strings.Contains(output, "%") {
		output = timeutil.Format(time.Now(), output)
	}

	err := writeTopDump(output, dump)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	fmtc.Printf(
		"{g}Dump successfully saved as %s (%s){!}\n",
		output, fmtutil.PrettySize(fsutil.GetSize(output)),
	)

	return EC_OK
}

// TopDiffCommand is "top-diff" command handler
func TopDiffCommand(args CommandArgs) int {
	var err error

	if !CORE.HasInstances() {
		terminal.Warn("No instances are created")
		return EC_WARN
	}

	if !args.Has(0) {
		terminal.Warn("You must define path to dump")
		return EC_WARN
	}

	dumpFile := args.Get(0)
	err = fsutil.ValidatePerms("FRS", dumpFile)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	topDump, err := readTopDump(dumpFile)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	field, resultNum, reverse := parseTopCommandArguments(args[1:])
	items, err := collectTopData(field)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	if (options.GetB(OPT_PAGER) || prefs.AutoPaging) && !useRawOutput {
		if pager.Setup() == nil {
			defer pager.Complete()
		}
	}

	return printTopDiff(items, topDump.Data, field, resultNum, reverse)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// collectTopData collects top data
func collectTopData(field string) (topItems, error) {
	if strings.HasPrefix(strings.ToLower(field), "cpu") {
		return collectTopCPUInfo(field)
	}

	return collectTopInfo(field)
}

// collectTopInfo collect field value from all instances
func collectTopInfo(field string) (topItems, error) {
	var section string
	var items topItems

	for _, id := range CORE.GetInstanceIDList() {
		state, err := CORE.GetInstanceState(id, false)

		if !state.IsWorks() || err != nil {
			continue
		}

		info, err := CORE.GetInstanceInfo(id, time.Second, false)

		if err != nil {
			continue
		}

		switch field {
		case "keys":
			items = append(items, topItem{id, float64(info.Keyspace.Keys()), false})
			continue
		case "expires":
			items = append(items, topItem{id, float64(info.Keyspace.Expires()), false})
			continue
		}

		if section == "" {
			section = findInfoSection(info, field)

			if section == "" {
				return nil, fmt.Errorf("Unknown info field \"%s\"", field)
			}
		}

		str := strings.Trim(info.Get(section, field), "%")
		value, err := strconv.ParseFloat(str, 64)

		if err != nil {
			return nil, fmt.Errorf("Field \"%s\" has an unsupported type", field)
		}

		items = append(items, topItem{id, value, strings.Contains(str, ".")})
	}

	return items, nil
}

// collectTopCPUInfo calculate CPU usage info for all instances
func collectTopCPUInfo(field string) (topItems, error) {
	var items topItems

	u1 := make(map[int][]float64)
	u2 := make(map[int][]float64)

	for i := 0; i < 2; i++ {
	INFOLOOP:
		for _, id := range CORE.GetInstanceIDList() {
			state, err := CORE.GetInstanceState(id, false)

			if !state.IsWorks() || err != nil {
				continue INFOLOOP
			}

			info, err := CORE.GetInstanceInfo(id, 3*time.Second, false)

			if err != nil {
				continue INFOLOOP
			}

			switch i {
			case 0:
				u1[id] = extractCPUUsageInfo(info)

			case 1:
				u2[id] = extractCPUUsageInfo(info)
			}
		}

		time.Sleep(5 * time.Second)
	}

	var usage []float64

	for id := range u1 {
		if len(u2[id]) != 4 {
			continue
		}

		usage = calculateInstanceCPUUsage(u1[id], u2[id], 5)

		switch field {
		case "cpu_sys":
			items = append(items, topItem{id, usage[0], true})
		case "cpu_user":
			items = append(items, topItem{id, usage[1], true})
		case "cpu_sys_children":
			items = append(items, topItem{id, usage[2], true})
		case "cpu_user_children":
			items = append(items, topItem{id, usage[3], true})
		case "cpu":
			items = append(items, topItem{id, usage[0] + usage[1], true})
		case "cpu_children":
			items = append(items, topItem{id, usage[2] + usage[3], true})
		}
	}

	return items, nil
}

// collectTopDump collect data for dump
func collectTopDump() *topDump {
	dump := &topDump{}

	for _, id := range CORE.GetInstanceIDList() {
		state, err := CORE.GetInstanceState(id, false)

		if !state.IsWorks() || err != nil {
			continue
		}

		info, err := CORE.GetInstanceInfo(id, 3*time.Second, false)

		if err != nil {
			continue
		}

		dump.Data = append(dump.Data, &topDumpItem{id, info.Flatten()})
	}

	return dump
}

// printTopInfo print items from top
func printTopInfo(items topItems, resultNum int, diff bool) {
	var t *table.Table

	if !useRawOutput {
		t = table.NewTable("#", "VALUE", "ID", "DESCRIPTION")
		t.SetSizes(0, 6, 3)
		t.SetAlignments(table.ALIGN_RIGHT, table.ALIGN_RIGHT, table.ALIGN_RIGHT)
	}

	for i, item := range items {
		value := fmtutil.PrettyNum(items[i].Value)

		// Format float values
		value = strutil.Exclude(value, ".00")

		if diff && items[i].Value > 0 {
			value = "+" + value
		}

		if !useRawOutput {
			meta, err := CORE.GetInstanceMeta(item.ID)

			if err != nil {
				t.Add(fmt.Sprintf("{s}%d{!}", i+1), value, item.ID, "{s-}--------{!}")
			} else {
				t.Add(fmt.Sprintf("{s}%d{!}", i+1), value, item.ID, meta.Desc)
			}

		} else {
			if items[i].IsFloat {
				fmtc.Printf("%d %f\n", item.ID, items[i].Value)
			} else {
				fmtc.Printf("%d %.0f\n", item.ID, items[i].Value)
			}
		}

		if i == resultNum-1 {
			break
		}
	}

	if !useRawOutput {
		t.Render()
	}
}

// printTopDiff prints top data diff
func printTopDiff(curTop topItems, dumpData []*topDumpItem, field string, resultNum int, reverse bool) int {
	diff, err := diffTopData(field, curTop, dumpData)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	if len(diff) == 0 {
		terminal.Warn("There is no data to compare")
		return EC_WARN
	}

	if reverse {
		sort.Sort(sort.Reverse(diff))
	} else {
		sort.Sort(diff)
	}

	printTopInfo(diff, resultNum, true)

	return EC_OK
}

// diffTopData compares top data
func diffTopData(field string, curTop topItems, dumpData []*topDumpItem) (topItems, error) {
	result := make(topItems, 0)

	switch field {
	case "keys", "expires":
		field += "_total"
	}

	for _, c := range curTop {
		v := findDumpInfo(dumpData, c.ID, field)

		if v == "" {
			continue
		}

		v = strings.Trim(v, "%")
		vf, err := strconv.ParseFloat(v, 64)

		diff := c.Value - vf

		if diff == 0 {
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("Field \"%s\" has an unsupported type", field)
		}

		result = append(result, topItem{c.ID, diff, strings.Contains(v, ".")})
	}

	return result, nil
}

// findDumpInfo tries to find info in dumped data
func findDumpInfo(dumpData []*topDumpItem, id int, field string) string {
	for _, item := range dumpData {
		if item.ID != id {
			continue
		}

		for _, infoField := range item.Info {
			if infoField[0] == field {
				return infoField[1]
			}
		}
	}

	return ""
}

// findInfoSection try to find section for given field
func findInfoSection(info *REDIS.Info, field string) string {
	for sectionName, sectionInfo := range info.Sections {
		for _, sectionField := range sectionInfo.Fields {
			if sectionField == field {
				return sectionName
			}
		}
	}

	return ""
}

// parseTopCommandArguments parse top command arguments
func parseTopCommandArguments(args CommandArgs) (string, int, bool) {
	var err error

	var field = "used_memory"
	var resultNum = 10
	var reverse = false

	switch len(args) {
	case 0:
		return field, resultNum, reverse
	case 1:
		field = args.Get(0)
	case 2:
		field = args.Get(0)
		resultNum, err = args.GetI(1)

		if err != nil {
			resultNum = 10
		}
	}

	if strings.HasPrefix(field, "^") {
		field = strings.TrimLeft(field, "^")
		reverse = true
	}

	if field == "-" {
		field = "used_memory"
	}

	return field, mathutil.Between(resultNum, 1, 9999), reverse
}

// writeTopDump writes top dump to file
func writeTopDump(filename string, data *topDump) error {
	if !strings.HasSuffix(filename, ".gz") {
		return fmt.Errorf("Output must have .gz extension")
	}

	if fsutil.IsExist(filename) {
		return fmt.Errorf("Dump %s already exists", filename)
	}

	return jsonutil.WriteGz(filename, data)
}

// readTopDump reads top dump from the file
func readTopDump(filename string) (*topDump, error) {
	if !strings.HasSuffix(filename, ".gz") {
		return nil, fmt.Errorf("Dump must have .gz extension")
	}

	dump := &topDump{}
	err := jsonutil.ReadGz(filename, dump)

	if err != nil {
		return nil, err
	}

	return dump, nil
}
