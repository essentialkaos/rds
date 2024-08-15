package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bufio"
	"os"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fmtutil"
	"github.com/essentialkaos/ek/v13/fsutil"
	"github.com/essentialkaos/ek/v13/mathutil"
	"github.com/essentialkaos/ek/v13/path"
	"github.com/essentialkaos/ek/v13/strutil"
	"github.com/essentialkaos/ek/v13/terminal"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	LOG_SOURCE_CLI  = "cli"
	LOG_SOURCE_SYNC = "sync"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// LogCommand is "log" command handler
func LogCommand(args CommandArgs) int {
	if len(args) == 0 {
		terminal.Error("You must define instance ID or log source (cli/sync) for this command")
		return EC_ERROR
	}

	var logFile string
	var isRedisLog bool

	source := args.Get(0)

	switch source {
	case LOG_SOURCE_CLI:
		logFile = path.Join(CORE.Config.GetS(CORE.PATH_LOG_DIR), CORE.LOG_FILE_CLI)
	case LOG_SOURCE_SYNC:
		logFile = path.Join(CORE.Config.GetS(CORE.PATH_LOG_DIR), CORE.LOG_FILE_SYNC)
	default:
		id, _, err := CORE.ParseIDDBPair(source)

		if err != nil {
			terminal.Error("Unknown log source %q", source)
			return EC_ERROR
		}

		logFile = CORE.GetInstanceLogFilePath(id)
		isRedisLog = true
	}

	err := fsutil.ValidatePerms("FR", logFile)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	fmtutil.SeparatorColorTag = "{s-}"
	fmtutil.SeparatorSymbol = "-"

	err = readLogFile(logFile, isRedisLog)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// readLogFile reads and formats given log file
func readLogFile(logFile string, isRedisLog bool) error {
	fd, err := os.OpenFile(logFile, os.O_RDONLY, 0)

	if err != nil {
		return err
	}

	defer fd.Close()

	fs := fsutil.GetSize(logFile)

	if fs > 0 {
		fd.Seek(-1*mathutil.Min(fs, 4096), 2)
	}

	lastPrint := time.Now()
	r := bufio.NewReader(fd)

	for {
		line, err := r.ReadString('\n')

		if err != nil {
			time.Sleep(250 * time.Millisecond)
			continue
		}

		line = strings.TrimRight(line, "\r\n")

		if time.Since(lastPrint) > time.Minute {
			fmtutil.Separator(true)
		}

		if isRedisLog {
			printRedisLogLine(line)
		} else {
			printRDSLogLine(line)
		}

		lastPrint = time.Now()
	}
}

// printRDSLogLine formats and prints line from RDS log
func printRDSLogLine(line string) {
	var target, colorTag string

	dateIndex := strings.Index(line, "]")

	if dateIndex == -1 {
		return
	}

	date := line[:dateIndex+1]
	line = line[dateIndex+2:]

	if strings.HasPrefix(line, "(") {
		targetIndex := strings.Index(line, ")")

		if targetIndex == -1 {
			return
		}

		target = line[:targetIndex+1]
		line = line[targetIndex+2:]
	}

	switch {
	case strings.Contains(line, "[CRITICAL] "):
		colorTag = "{r*}"
	case strings.Contains(line, "[ERROR] "):
		colorTag = "{r}"
	case strings.Contains(line, "[WARN] "):
		colorTag = "{y}"
	}

	if target != "" {
		fmtc.Printf("{s-}%s{!} {s}%s{!} "+colorTag+"%s{!}\n", date, target, line)
	} else {
		fmtc.Printf("{s-}%s{!} "+colorTag+"%s{!}\n", date, line)
	}
}

// printRedisLogLine formats and prints line from Redis log
func printRedisLogLine(line string) {
	role := strutil.ReadField(line, 0, false, ' ')
	role = strutil.ReadField(role, 1, false, ':')

	day := strutil.ReadField(line, 1, false, ' ')
	month := strutil.ReadField(line, 2, false, ' ')
	year := strutil.ReadField(line, 3, false, ' ')
	hms := strutil.ReadField(line, 4, false, ' ')

	switch strings.ToLower(month) {
	case "jan":
		month = "1"
	case "feb":
		month = "2"
	case "mar":
		month = "3"
	case "apr":
		month = "4"
	case "may":
		month = "5"
	case "jun":
		month = "6"
	case "jul":
		month = "7"
	case "aug":
		month = "8"
	case "sep":
		month = "9"
	case "oct":
		month = "10"
	case "nov":
		month = "11"
	case "dec":
		month = "12"
	}

	var sepIndex int
	var colorTag string

	for i := 0; i < 4; i++ {
		switch i {
		case 0:
			sepIndex = strings.Index(line, " # ") // Warning
			colorTag = "{y}"
		case 1:
			sepIndex = strings.Index(line, " * ") // Notice
			colorTag = ""
		case 2:
			sepIndex = strings.Index(line, " - ") // Verbose
			colorTag = "{s-}"
		case 3:
			sepIndex = strings.Index(line, " . ") // Debug
			colorTag = "{s-}"
		}

		if sepIndex > 0 {
			break
		}
	}

	if sepIndex == -1 {
		return
	}

	fmtc.Printf(
		"{s-}[ %s/%s/%s %s | %s ]{!} "+colorTag+"%s{!}\n",
		year, month, day, hms, role, line[sepIndex+3:],
	)
}
