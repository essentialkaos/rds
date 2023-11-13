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
	"time"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/fmtutil/table"
	"github.com/essentialkaos/ek/v12/strutil"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
	REDIS "github.com/essentialkaos/rds/redis"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type instanceMemoryMetric struct {
	Field string
	Value any
}

// ////////////////////////////////////////////////////////////////////////////////// //

// MemoryCommand is "memory" command handler
func MemoryCommand(args CommandArgs) int {
	err := args.Check(true)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	id, _, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	metrics, err := getInstanceMemoryUsage(id)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	if useRawOutput {
		printRawMemoryUsage(metrics)
	} else {
		printMemoryUsage(metrics)
	}

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// printMemoryUsage prints memory metrics
func printMemoryUsage(metrics []instanceMemoryMetric) {
	if len(metrics) == 0 {
		terminal.Warn("No data to show")
		return
	}

	t := table.NewTable("METRIC", "VALUE")

	for _, m := range metrics {
		switch {
		case strings.HasSuffix(m.Field, ".percentage"):
			t.Add(m.Field, fmtutil.PrettyNum(m.Value)+"%")

		case strutil.HasPrefixAny(m.Field, "allocator.", "clients.", "replication.", "aof.", "lua.", "cluster."),
			strutil.HasSuffixAny(m.Field, ".bytes", ".allocated", ".bytes-per-key"):
			t.Add(m.Field, fmtc.Sprintf("%s {s}(%s){!}",
				fmtutil.PrettySize(m.Value), fmtutil.PrettyNum(m.Value),
			))

		default:
			t.Add(m.Field, fmtutil.PrettyNum(m.Value))
		}
	}

	t.Render()
}

// printRawMemoryUsage prints raw memory metrics
func printRawMemoryUsage(metrics []instanceMemoryMetric) {
	for _, m := range metrics {
		fmt.Printf("%s %v\n", m.Field, m.Value)
	}
}

// getInstanceMemoryUsage executes MEMORY STATS command and returns results as
// a slice with metrics
func getInstanceMemoryUsage(id int) ([]instanceMemoryMetric, error) {
	resp, err := CORE.ExecCommand(id,
		&REDIS.Request{
			Command: []string{"MEMORY", "STATS"},
			Timeout: 3 * time.Second,
		},
	)

	if err != nil {
		return nil, err
	}

	items, err := resp.Array()

	if err != nil {
		return nil, err
	}

	return memoryStatsToMetrics("", items)
}

func memoryStatsToMetrics(prefix string, data []*REDIS.Resp) ([]instanceMemoryMetric, error) {
	var err error
	var value any
	var result []instanceMemoryMetric

	if len(data)%2 != 0 {
		return nil, fmt.Errorf("Wrong number of items in MEMORY STATS command response")
	}

	for i := 0; i < len(data); i += 2 {
		field, _ := data[i].Str()

		if data[i+1].HasType(REDIS.ARRAY) {
			valueArray, err := data[i+1].Array()

			if err == nil {
				metrics, err := memoryStatsToMetrics(field+".", valueArray)

				if err == nil {
					result = append(result, metrics...)
					continue
				}
			}
		} else {
			if data[i+1].HasType(REDIS.INT) {
				value, err = data[i+1].Int()
			} else {
				value, err = data[i+1].Float64()
			}
		}

		if err != nil {
			return nil, fmt.Errorf("Can't parse value for field %s: %v", field, err)
		}

		result = append(result, instanceMemoryMetric{prefix + field, value})
	}

	return result, nil
}
