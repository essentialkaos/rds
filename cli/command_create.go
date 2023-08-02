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
	"github.com/essentialkaos/ek/v12/fmtutil/table"
	"github.com/essentialkaos/ek/v12/log"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/passwd"
	"github.com/essentialkaos/ek/v12/system"
	"github.com/essentialkaos/ek/v12/terminal"

	API "github.com/essentialkaos/rds/api"
	CORE "github.com/essentialkaos/rds/core"
	SC "github.com/essentialkaos/rds/sync/client"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type instanceBasicInfo struct {
	Owner           string
	Password        string
	ReplicationType string
	Auth            string
	Desc            string
}

// ////////////////////////////////////////////////////////////////////////////////// //

// CreateCommand is "create" command handler
func CreateCommand(args CommandArgs) int {
	var err error

	if CORE.GetAvailableInstanceID() == -1 {
		terminal.Warn("No available ID for usage")
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
		terminal.Error(err.Error())
		return EC_ERROR
	}

	desc, instancePass, redisPass, replType, err := readInitInfo()

	if err != nil {
		if err == terminal.ErrKillSignal {
			return EC_OK
		}

		terminal.Error(err.Error())

		return EC_ERROR
	}

	if instancePass == "" {
		instancePass = passwd.GenPassword(
			CORE.Config.GetI(CORE.MAIN_DEFAULT_PASS_LENGTH),
			passwd.STRENGTH_MEDIUM,
		)
	}

	pepper := passwd.GenPassword(32, passwd.STRENGTH_MEDIUM)
	hash, err := passwd.Encrypt(instancePass, pepper)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	if options.GetB(OPT_SECURE) && redisPass == "" {
		redisPass = passwd.GenPassword(
			CORE.Config.GetI(CORE.MAIN_DEFAULT_PASS_LENGTH),
			passwd.STRENGTH_MEDIUM,
		)
	}

	meta := CORE.NewInstanceMeta()

	meta.Desc = desc
	meta.ReplicationType = replType
	meta.Tags = tags

	meta.AuthInfo.User = CORE.User.RealName
	meta.AuthInfo.Hash = hash
	meta.AuthInfo.Pepper = pepper

	if redisPass != "" {
		meta.Preferencies.Password = redisPass
		meta.Preferencies.IsSecure = true
	}

	meta.Preferencies.IsSaveDisabled = options.GetB(OPT_DISABLE_SAVES)

	err = CORE.CreateInstance(meta)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	fmtc.Println("{*}Done, a new Redis instance has been successfully created. Just for you.{!}")

	if len(tags) == 0 {
		log.Info("(%s) Created instance with ID %d", CORE.User.RealName, meta.ID)
	} else {
		log.Info(
			"(%s) Created instance with ID %d (tags: %s)",
			CORE.User.RealName, meta.ID, strings.Join(tags, ","),
		)
	}

	showInstanceInfo(meta, instancePass, tags)

	err = SC.PropagateCommand(API.COMMAND_CREATE, meta.ID, meta.UUID)

	if err != nil {
		terminal.Error(err.Error())
	}

	err = CORE.SaveStates(CORE.GetStatesFilePath())

	if err != nil {
		terminal.Error(err.Error())
	}

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// readInitInfo reads user input for init command
func readInitInfo() (string, string, string, CORE.ReplicationType, error) {
	var err error
	var replType CORE.ReplicationType
	var desc, instancePass, redisPass string

	for {
		desc, err = terminal.Read("Please enter the description for your instance", true)

		if err != nil {
			return desc, instancePass, redisPass, replType, err
		}

		desc = strings.TrimSpace(desc)

		if len(desc) < CORE.MIN_DESC_LENGTH {
			terminal.Warn("\nDescription must at least %d symbols long\n", CORE.MIN_DESC_LENGTH)
			continue
		}

		if len(desc) > CORE.MAX_DESC_LENGTH {
			terminal.Warn("\nDescription must be less than %d symbols long\n", CORE.MAX_DESC_LENGTH)
			continue
		}

		break
	}

	fmtc.NewLine()

	for {
		instancePass, err = terminal.ReadPassword("Please enter the password for instance (or leave blank for autogenerated password)", false)

		if err != nil {
			return desc, instancePass, redisPass, replType, err
		}

		if instancePass != "" && len(instancePass) < CORE.Config.GetI(CORE.MAIN_MIN_PASS_LENGTH) {
			terminal.Warn("\nPassword can't be less than %s symbols\n", CORE.Config.GetS(CORE.MAIN_MIN_PASS_LENGTH))
			continue
		}

		break
	}

	fmtc.NewLine()

	if options.GetB(OPT_SECURE) {
		for {
			redisPass, err = terminal.ReadPassword("Please enter the password for Redis auth (or leave blank for autogenerated password)", false)

			if err != nil {
				return desc, instancePass, redisPass, replType, err
			}

			if redisPass != "" && len(redisPass) < CORE.Config.GetI(CORE.MAIN_MIN_PASS_LENGTH) {
				terminal.Warn("\nAuth password can't be less than %s symbols\n", CORE.Config.GetS(CORE.MAIN_MIN_PASS_LENGTH))
				continue
			}

			break
		}

		fmtc.NewLine()
	}

	for {
		var role string

		if CORE.Config.GetS(CORE.REPLICATION_DEFAULT_ROLE) == string(CORE.REPL_TYPE_REPLICA) {
			role, err = terminal.Read("Please select a type of replication for instance ({*}[R - replica]{!*} / S - standby)", false)
		} else {
			role, err = terminal.Read("Please select a type of replication for instance (R - replica / {*}[S - standby]{!*})", false)
		}

		if err != nil {
			return desc, instancePass, redisPass, replType, err
		}

		switch role {
		case "":
			replType = CORE.ReplicationType(CORE.Config.GetS(CORE.REPLICATION_DEFAULT_ROLE))
		case "1", "S", "s":
			replType = CORE.REPL_TYPE_STANDBY
		case "2", "R", "r":
			replType = CORE.REPL_TYPE_REPLICA
		default:
			terminal.Warn("\nYou entered a wrong value\n")
			continue
		}

		break
	}

	fmtc.NewLine()

	return desc, instancePass, redisPass, replType, err
}

// Show basic instance info
func showInstanceInfo(meta *CORE.InstanceMeta, password string, tags []string) {
	var desc string

	if len(meta.Desc) > MAX_DESC_LENGTH {
		desc = meta.Desc[:MAX_DESC_LENGTH-1] + "…"
	} else {
		desc = meta.Desc
	}

	t := table.NewTable().SetSizes(16, MAX_DESC_LENGTH)

	coloredPassword := fmtutil.ColorizePassword(password, "", "{c}", "{m}")
	coloredPrefix := fmtutil.ColorizePassword(meta.Preferencies.Prefix+"_", "", "{c}", "{m}")

	fmtc.NewLine()

	t.Separator()
	fmtc.Println(" ▾ {*}INSTANCE INFO{!}")
	t.Separator()

	t.Print("ID", meta.ID)
	t.Print("Port", CORE.GetInstancePort(meta.ID))
	t.Print("Replication Type", meta.ReplicationType)
	t.Print("Description", desc+" "+renderTags(tags...))

	if meta.Preferencies.Password != "" {
		coloredRedisPass := fmtutil.ColorizePassword(meta.Preferencies.Password, "", "{c}", "{m}")
		t.Print("Auth Password", coloredRedisPass)
	}

	t.Print("Password", coloredPassword)
	t.Print("Command Prefix", coloredPrefix)

	t.Separator()

	fmtc.NewLine()

	fmtc.Println("{y}Please save your password and command prefix in a safe place{!}")
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
