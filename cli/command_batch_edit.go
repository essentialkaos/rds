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
	"strconv"
	"strings"

	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/pluralize"
	"github.com/essentialkaos/ek/v13/spinner"
	"github.com/essentialkaos/ek/v13/strutil"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/terminal/input"

	API "github.com/essentialkaos/rds/api"
	CORE "github.com/essentialkaos/rds/core"
	SC "github.com/essentialkaos/rds/sync/client"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// BatchEditCommand is "batch-edit" command handler
func BatchEditCommand(args CommandArgs) int {
	if len(args) == 0 {
		terminal.Error("You must define instance ID's for this command")
		return EC_ERROR
	}

	idList, err := parseIDList(args)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	ok, err := input.ReadAnswer(
		pluralize.P(
			"Do you want to modify meta for %d %s?",
			len(idList), "instance", "instances",
		), "Y",
	)

	if err != nil || !ok {
		return EC_CANCEL
	}

	info, err := readEditInfo(false, true, true, true)

	if err != nil {
		if err == input.ErrKillSignal {
			return EC_OK
		}

		terminal.Error(err)
		return EC_ERROR
	}

	hasErrors := false

	for _, id := range idList {
		if !CORE.IsInstanceExist(id) {
			continue
		}

		spinner.Show("Updating meta for instance {*}%d{!}", id)

		meta, err := CORE.GetInstanceMeta(id)

		if err != nil {
			spinner.Done(false)
			terminal.Error(err)
			fmtc.NewLine()
			hasErrors = true
			continue
		}

		var changes []string

		if info.InstancePassword != "" {
			auth, err := CORE.NewInstanceAuth(info.InstancePassword)

			if err != nil {
				spinner.Done(false)
				terminal.Error(err)
				fmtc.NewLine()
				hasErrors = true
				continue
			}

			meta.Auth.Pepper = auth.Pepper
			meta.Auth.Hash = auth.Hash

			changes = append(changes, "password updated")
		}

		if info.Owner != "" {
			changes = append(changes, fmt.Sprintf("owner changed %q → %q", meta.Auth.User, info.Owner))
			meta.Auth.User = info.Owner
		}

		if info.ReplicationType != "" {
			changes = append(changes, fmt.Sprintf("description changed %q → %q", meta.Desc, info.Desc))
			meta.Preferencies.ReplicationType = CORE.ReplicationType(info.ReplicationType)
		}

		err = CORE.UpdateInstance(meta)

		if err != nil {
			spinner.Done(false)
			terminal.Error(err)
			fmtc.NewLine()
			hasErrors = true
			continue
		}

		for _, c := range changes {
			logger.Info(id, "Instance meta updated (batch): %s", c)
		}

		err = SC.PropagateCommand(API.COMMAND_EDIT, meta.ID, meta.UUID)

		if err != nil {
			spinner.Done(false)
			terminal.Error(err)
			fmtc.NewLine()
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

// parseIDList parse id's list
func parseIDList(ids []string) ([]int, error) {
	ids = strutil.Fields(strings.Join(ids, " "))

	var result []int

	for _, id := range ids {
		if strings.Contains(id, "-") {
			start, end := parseIDRange(id)

			if start == -1 || end == -1 {
				return nil, fmt.Errorf("Can't parse ID's range %s", id)
			}

			idList, err := fillIDList(start, end)

			if err != nil {
				return nil, err
			}

			result = append(result, idList...)
		} else {
			idInt, err := strconv.Atoi(id)

			if err != nil {
				return nil, fmt.Errorf("Can't parse ID %s", id)
			}

			result = append(result, idInt)
		}
	}

	return result, nil
}

// parseIDRange parse id range and return range start and end
func parseIDRange(id string) (int, int) {
	idSlice := strings.Split(id, "-")

	if len(idSlice) != 2 {
		return -1, -1
	}

	start, err := strconv.Atoi(idSlice[0])

	if err != nil {
		return -1, -1
	}

	end, err := strconv.Atoi(idSlice[1])

	if err != nil {
		return -1, -1
	}

	return start, end
}

// fillIDList fill int slice
func fillIDList(start, end int) ([]int, error) {
	if start > end {
		return nil, errors.New("Range start can't be greater than range end")
	}

	if start == end {
		return []int{start}, nil
	}

	var result []int

	for i := 0; i <= end-start; i++ {
		result = append(result, start+i)
	}

	return result, nil
}
