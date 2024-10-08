package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"strings"

	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fmtutil"
	"github.com/essentialkaos/ek/v13/fmtutil/table"
	"github.com/essentialkaos/ek/v13/options"
	"github.com/essentialkaos/ek/v13/strutil"
	"github.com/essentialkaos/ek/v13/system"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/terminal/input"

	API "github.com/essentialkaos/rds/api"
	CORE "github.com/essentialkaos/rds/core"
	SC "github.com/essentialkaos/rds/sync/client"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type instanceBasicInfo struct {
	Owner                  string
	Desc                   string
	InstancePassword       string
	ServicePassword        string
	ReplicationType        string
	CustomInstancePassword bool
	CustomServicePassword  bool
}

// ////////////////////////////////////////////////////////////////////////////////// //

// CreateCommand is "create" command handler
func CreateCommand(args CommandArgs) int {
	var err error

	if CORE.GetAvailableInstanceID() == -1 {
		terminal.Warn("No available ID for usage")
		return EC_WARN
	}

	if !checkVirtualIP() {
		return EC_WARN
	}

	if !isSystemConfigured() {
		return EC_WARN
	}

	if !isEnoughMemoryToCreate() {
		return EC_ERROR
	}

	tags, err := parseTagsOption(options.GetS(OPT_TAGS))

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	info, err := readBasicInstanceInfo()

	if err != nil {
		if err == input.ErrKillSignal {
			return EC_OK
		}

		terminal.Error(err)

		return EC_ERROR
	}

	if !info.CustomInstancePassword {
		info.InstancePassword = CORE.GenPassword()
	}

	if options.GetB(OPT_SECURE) && !info.CustomServicePassword {
		info.ServicePassword = CORE.GenPassword()
	}

	meta, err := CORE.NewInstanceMeta(
		info.InstancePassword,
		info.ServicePassword,
	)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	meta.Desc = info.Desc
	meta.Preferencies.ReplicationType = CORE.ReplicationType(info.ReplicationType)
	meta.Preferencies.IsSaveDisabled = options.GetB(OPT_DISABLE_SAVES)
	meta.Tags = tags

	err = CORE.CreateInstance(meta)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	fmtc.Println("{*}Done, a new Redis instance has been successfully created. Just for you.{!}")

	if len(tags) == 0 {
		logger.Info(meta.ID, "Instance created")
	} else {
		logger.Info(meta.ID, "Instance created (tags: %s)", strings.Join(tags, ","))
	}

	showInstanceInfo(meta, info, tags)

	err = SC.PropagateCommand(API.COMMAND_CREATE, meta.ID, meta.UUID)

	if err != nil {
		terminal.Error(err)
	}

	err = CORE.SaveStates(CORE.GetStatesFilePath())

	if err != nil {
		terminal.Error(err)
	}

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// readBasicInstanceInfo reads user input for creating instance
func readBasicInstanceInfo() (*instanceBasicInfo, error) {
	var err error

	info := &instanceBasicInfo{}

	for {
		info.Desc, err = input.Read("Please enter the description for your instance", true)

		if err != nil {
			return nil, err
		}

		info.Desc = strings.TrimSpace(info.Desc)

		if len(info.Desc) < CORE.MIN_DESC_LENGTH {
			terminal.Warn("\nDescription must at least %d symbols long\n", CORE.MIN_DESC_LENGTH)
			continue
		}

		if len(info.Desc) > CORE.MAX_DESC_LENGTH {
			terminal.Warn("\nDescription must be less than %d symbols long\n", CORE.MAX_DESC_LENGTH)
			continue
		}

		break
	}

	fmtc.NewLine()

	for {
		info.InstancePassword, err = input.ReadPassword("Please enter the password for instance (or leave blank for autogenerated password)", false)

		if err != nil {
			return nil, err
		}

		if info.InstancePassword != "" && len(info.InstancePassword) < CORE.Config.GetI(CORE.MAIN_MIN_PASS_LENGTH) {
			terminal.Warn("\nPassword can't be less than %s symbols\n", CORE.Config.GetS(CORE.MAIN_MIN_PASS_LENGTH))
			continue
		}

		info.CustomInstancePassword = info.InstancePassword != ""

		break
	}

	fmtc.NewLine()

	if options.GetB(OPT_SECURE) {
		for {
			info.ServicePassword, err = input.ReadPassword("Please enter the password for service user (or leave blank for autogenerated password)", false)

			if err != nil {
				return nil, err
			}

			if info.ServicePassword != "" && len(info.ServicePassword) < CORE.Config.GetI(CORE.MAIN_MIN_PASS_LENGTH) {
				terminal.Warn("\nPassword can't be less than %s symbols\n", CORE.Config.GetS(CORE.MAIN_MIN_PASS_LENGTH))
				continue
			}

			info.CustomServicePassword = info.ServicePassword != ""

			break
		}

		fmtc.NewLine()
	}

	for {
		var role string

		if CORE.Config.GetS(CORE.REPLICATION_DEFAULT_ROLE) == string(CORE.REPL_TYPE_REPLICA) {
			role, err = input.Read("Please select a type of replication for instance ({*}[R - replica]{!*} / S - standby)", false)
		} else {
			role, err = input.Read("Please select a type of replication for instance (R - replica / {*}[S - standby]{!*})", false)
		}

		if err != nil {
			return nil, err
		}

		switch role {
		case "":
			info.ReplicationType = CORE.Config.GetS(CORE.REPLICATION_DEFAULT_ROLE)
		case "1", "S", "s":
			info.ReplicationType = string(CORE.REPL_TYPE_STANDBY)
		case "2", "R", "r":
			info.ReplicationType = string(CORE.REPL_TYPE_REPLICA)
		default:
			terminal.Warn("\nYou entered a wrong value\n")
			continue
		}

		break
	}

	fmtc.NewLine()

	return info, err
}

// Show basic instance info
func showInstanceInfo(meta *CORE.InstanceMeta, info *instanceBasicInfo, tags []string) {
	t := table.NewTable().SetSizes(17, MAX_DESC_LENGTH)

	fmtc.NewLine()

	t.Border()
	fmtc.Println(" ▾ {*}INSTANCE INFO{!}")
	t.Border()

	t.Print("ID", meta.ID)
	t.Print("Port", CORE.GetInstancePort(meta.ID))
	t.Print("Replication Type", meta.Preferencies.ReplicationType)
	t.Print(
		"Description",
		strutil.Ellipsis(meta.Desc, MAX_DESC_LENGTH)+" "+renderTags(tags...),
	)

	if !info.CustomInstancePassword {
		t.Print(
			"Instance Password",
			fmtutil.ColorizePassword(info.InstancePassword, "{b}", "{g}", "{y}"),
		)
	}

	if info.CustomServicePassword {
		t.Print(
			"Service Password",
			fmtutil.ColorizePassword(info.ServicePassword, "{b}", "{g}", "{y}"),
		)
	}

	t.Border()
	fmtc.NewLine()

	fmtc.Println("{y}▲ Please save your passwords in a safe place!{!}")
}

// parseTagsOption parses tags option
func parseTagsOption(tags string) ([]string, error) {
	if tags == "" {
		return nil, nil
	}

	tagsSlice := strings.Split(tags, ",")

	if len(tagsSlice) > CORE.MAX_TAGS {
		return nil, fmt.Errorf("Max number of tags (%d) reached", CORE.MAX_TAGS)
	}

	for _, tag := range tagsSlice {
		if !CORE.IsValidTag(tag) {
			return nil, fmt.Errorf("Tag %s has the wrong format", tag)
		}
	}

	return tagsSlice, nil
}

// getMemUsage returns memory usage in percentage
func getMemUsage() int {
	memUsage, err := system.GetMemUsage()

	if err != nil {
		return 0
	}

	return int((float64(memUsage.MemUsed) / float64(memUsage.MemTotal)) * 100)
}
