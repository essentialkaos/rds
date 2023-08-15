package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/log"
	"github.com/essentialkaos/ek/v12/pluralize"
	"github.com/essentialkaos/ek/v12/spinner"
	"github.com/essentialkaos/ek/v12/strutil"
	"github.com/essentialkaos/ek/v12/terminal"

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
		terminal.Error(err.Error())
		return EC_ERROR
	}

	ok, err := terminal.ReadAnswer(
		pluralize.P(
			"Do you want to modify meta for %d %s?",
			len(idList), "instance", "instances",
		), "Y",
	)

	if !ok || err != nil {
		return EC_ERROR
	}

	fmtc.NewLine()

	info, err := readEditInfo(false, true, true, true)

	if err != nil {
		if err == terminal.ErrKillSignal {
			return EC_OK
		}

		terminal.Error(err.Error())
		return EC_ERROR
	}

	hasErrors := false

	for _, id := range idList {
		if !CORE.IsInstanceExist(id) {
			continue
		}

		spinner.Show("Updating meta for instance %d", id)

		meta, err := CORE.GetInstanceMeta(id)

		if err != nil {
			spinner.Done(false)
			terminal.Error(err.Error())
			fmtc.NewLine()
			hasErrors = true
			continue
		}

		if info.InstancePassword != "" {
			auth, err := CORE.NewInstanceAuth(info.InstancePassword)

			if err != nil {
				spinner.Done(false)
				terminal.Error(err.Error())
				fmtc.NewLine()
				hasErrors = true
				continue
			}

			meta.Auth.Pepper = auth.Pepper
			meta.Auth.Hash = auth.Hash
		}

		if info.Owner != "" {
			meta.Auth.User = info.Owner
		}

		if info.ReplicationType != "" {
			meta.Preferencies.ReplicationType = CORE.ReplicationType(info.ReplicationType)
		}

		err = CORE.UpdateInstance(meta)

		if err != nil {
			spinner.Done(false)
			terminal.Error(err.Error())
			fmtc.NewLine()
			hasErrors = true
			continue
		}

		log.Info("(%s) Updated info for instance with ID %d", CORE.User.RealName, id)

		err = SC.PropagateCommand(API.COMMAND_EDIT, meta.ID, meta.UUID)

		if err != nil {
			spinner.Done(false)
			terminal.Error(err.Error())
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
