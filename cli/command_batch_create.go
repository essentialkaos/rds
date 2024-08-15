package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/essentialkaos/ek/v13/csv"
	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fmtutil/table"
	"github.com/essentialkaos/ek/v13/fsutil"
	"github.com/essentialkaos/ek/v13/options"
	"github.com/essentialkaos/ek/v13/spinner"
	"github.com/essentialkaos/ek/v13/strutil"
	"github.com/essentialkaos/ek/v13/system"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/terminal/input"

	API "github.com/essentialkaos/rds/api"
	CORE "github.com/essentialkaos/rds/core"
	SC "github.com/essentialkaos/rds/sync/client"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// BatchCreateCommand is "batch-create" command handler
func BatchCreateCommand(args CommandArgs) int {
	var err error

	if len(args) == 0 {
		terminal.Error("You must define path to CSV file")
		return EC_ERROR
	}

	if !isSystemConfigured() {
		return EC_WARN
	}

	if !isEnoughMemoryToCreate() {
		return EC_ERROR
	}

	infoList, err := readInstanceList(args.Get(0))

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	showInstanceList(infoList)

	ok, err := input.ReadAnswer("Create these instances?", "N")

	if !ok || err != nil {
		return EC_CANCEL
	}

	fmtc.NewLine()

	var hasErrors bool

	for _, info := range infoList {
		id := CORE.GetAvailableInstanceID()

		spinner.Show("Creating instance {*}%s{!}", info.Desc)

		if id == -1 {
			spinner.Done(false)
			hasErrors = true
			continue
		}

		meta, err := CORE.NewInstanceMeta(info.InstancePassword, info.ServicePassword)

		if err != nil {
			spinner.Done(false)
			hasErrors = true
			continue
		}

		meta.Desc = info.Desc
		meta.Auth.User = info.Owner
		meta.Preferencies.ReplicationType = CORE.ReplicationType(info.ReplicationType)
		meta.Preferencies.IsSaveDisabled = options.GetB(OPT_DISABLE_SAVES)

		err = CORE.CreateInstance(meta)

		if err != nil {
			spinner.Done(false)
			hasErrors = true
			continue
		}

		logger.Info(meta.ID, "Instance created (batch)")

		err = SC.PropagateCommand(API.COMMAND_CREATE, meta.ID, meta.UUID)

		if err != nil {
			spinner.Done(false)
			hasErrors = true
			continue
		}

		spinner.Done(true)
	}

	if hasErrors {
		return EC_ERROR
	}

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// readInstanceList read instances info from csv file
func readInstanceList(file string) ([]*instanceBasicInfo, error) {
	var result []*instanceBasicInfo

	err := fsutil.ValidatePerms("FRS", file)

	if err != nil {
		return nil, err
	}

	fd, err := os.OpenFile(file, os.O_RDONLY, 0)

	if err != nil {
		return nil, err
	}

	defer fd.Close()

	r := csv.NewReader(fd)

	for {
		row, err := r.Read()

		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		err = validateInstanceListRow(row)

		if err != nil {
			return nil, fmt.Errorf("Can't parse row %d: %v", r.Line(), err)
		}

		result = append(result, &instanceBasicInfo{
			Owner:            row.Get(0),
			InstancePassword: row.Get(1),
			ReplicationType:  row.Get(2),
			ServicePassword:  row.Get(3),
			Desc:             row.Get(4),
		})
	}

	return result, nil
}

// validateInstanceListRow validate CSV record values
func validateInstanceListRow(row csv.Row) error {
	if row.Size() != 5 {
		return errors.New("Not enough records (at least 5 columns required)")
	}

	owner := row.Get(0)
	instPass := row.Get(1)
	replType := row.Get(2)
	srvPass := row.Get(3)
	desc := row.Get(4)

	switch {
	case strutil.Exclude(owner, " ") == "":
		return errors.New("Column 1 must contain valid user name")

	case strutil.Exclude(instPass, " ") == "":
		return errors.New("Column 2 must contain valid password")

	case strutil.Exclude(replType, " ") == "":
		return errors.New("Column 3 must contain valid replication type")

	case strutil.Exclude(desc, " ") == "":
		return errors.New("Column 5 must contain valid description")

	case len(instPass) < CORE.Config.GetI(CORE.MAIN_MIN_PASS_LENGTH):
		return fmt.Errorf("Password can't be less than %s symbols long", CORE.Config.GetS(CORE.MAIN_MIN_PASS_LENGTH))

	case srvPass != "" && len(srvPass) < CORE.Config.GetI(CORE.MAIN_MIN_PASS_LENGTH):
		return fmt.Errorf("Auth password can't be less than %s symbols long", CORE.Config.GetS(CORE.MAIN_MIN_PASS_LENGTH))

	case replType != "" && replType != string(CORE.REPL_TYPE_REPLICA) && replType != string(CORE.REPL_TYPE_STANDBY):
		return errors.New("Column 3 must contain valid replication type (\"replica\" or \"standby\")")

	case len(desc) < CORE.MIN_DESC_LENGTH:
		return fmt.Errorf("Description must at least %d symbols long", CORE.MIN_DESC_LENGTH)

	case len(desc) > CORE.MAX_DESC_LENGTH:
		return fmt.Errorf("Description must be less than %d symbols long", CORE.MAX_DESC_LENGTH)

	case strings.Contains(owner, " "):
		return errors.New("Owner name can't contain spaces")

	case strings.Contains(instPass, " "):
		return errors.New("Password can't contain spaces")

	case strings.Contains(srvPass, " "):
		return errors.New("Auth password can't contain spaces")

	case !system.IsUserExist(owner):
		return fmt.Errorf("The user with name %s doesn't exist on the system", owner)
	}

	return nil
}

// showInstanceList show table with instances info
func showInstanceList(infoList []*instanceBasicInfo) {
	t := table.NewTable(
		"OWNER", "PASSWORD", "REPLICATION TYPE",
		"AUTH PASSWORD", "DESCRIPTION",
	)

	t.SetAlignments(table.ALIGN_RIGHT, table.ALIGN_RIGHT, table.ALIGN_RIGHT, table.ALIGN_RIGHT)

	for _, info := range infoList {
		if !options.GetB(OPT_PRIVATE) {
			t.Add(
				info.Owner, "{s-}[hidden]{!}", info.ReplicationType,
				strutil.Q(info.ServicePassword, "{s-}—{!}"), info.Desc,
			)
		} else {
			t.Add(
				info.Owner, info.InstancePassword, info.ReplicationType,
				strutil.Q(info.ServicePassword, "{s-}—{!}"), info.Desc,
			)
		}
	}

	t.Render()
	fmtc.NewLine()
}
