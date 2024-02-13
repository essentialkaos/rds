package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/sliceutil"
	"github.com/essentialkaos/ek/v12/system"
	"github.com/essentialkaos/ek/v12/terminal"

	API "github.com/essentialkaos/rds/api"
	CORE "github.com/essentialkaos/rds/core"
	SC "github.com/essentialkaos/rds/sync/client"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// EditCommand is "edit" command handler
func EditCommand(args CommandArgs) int {
	err := args.Check(false)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	id, _, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	state, err := CORE.GetInstanceState(id, true)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	err = showInstanceBasicInfoCard(id, state)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	ok, err := terminal.ReadAnswer(
		"Do you want to modify meta for this instance?", "Y",
	)

	if !ok || err != nil {
		return EC_CANCEL
	}

	meta, err := CORE.GetInstanceMeta(id)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	fmtc.NewLine()

	info, err := readEditInfo(true, true, true, true)

	if err != nil {
		if err == terminal.ErrKillSignal {
			return EC_OK
		}

		terminal.Error(err)
		return EC_ERROR
	}

	var changes []string

	// It's safe to modify this metadata, because GetInstanceMeta returns
	// copy of metadata
	if info.InstancePassword != "" {
		auth, err := CORE.NewInstanceAuth(info.InstancePassword)

		if err != nil {
			terminal.Error(err)
			return EC_ERROR
		}

		meta.Auth.Pepper = auth.Pepper
		meta.Auth.Hash = auth.Hash

		changes = append(changes, "password updated")
	}

	if info.Owner != "" {
		changes = append(changes, fmt.Sprintf("owner changed %q → %q", meta.Auth.User, info.Owner))
		meta.Auth.User = info.Owner
	}

	if info.Desc != "" {
		changes = append(changes, fmt.Sprintf("description changed %q → %q", meta.Desc, info.Desc))
		meta.Desc = info.Desc
	}

	if info.ReplicationType != "" {
		changes = append(changes, fmt.Sprintf(
			"replication type changed %q → %q",
			meta.Preferencies.ReplicationType,
			info.ReplicationType),
		)
		meta.Preferencies.ReplicationType = CORE.ReplicationType(info.ReplicationType)
	}

	err = CORE.UpdateInstance(meta)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	for _, c := range changes {
		logger.Info(id, "Instance meta updated: %s", c)
	}

	fmtc.Printf("{g}Done. Data for instance with ID %d successfully updated.{!}\n", id)

	err = SC.PropagateCommand(API.COMMAND_EDIT, meta.ID, meta.UUID)

	if err != nil {
		terminal.Error(err)
		return EC_ERROR
	}

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Read user input for edit command
func readEditInfo(readDesc, readPass, readOwner, readReplType bool) (*instanceBasicInfo, error) {
	var err error

	info := &instanceBasicInfo{}

	if readDesc {
		info.Desc, err = terminal.Read("Please enter a new description (or leave blank to keep existing)", false)

		if err != nil {
			return nil, err
		}

		fmtc.NewLine()
	}

	if readPass {
		for {
			info.InstancePassword, err = terminal.ReadPassword("Please enter a new password (or leave blank to keep existing)", false)

			if err != nil {
				return nil, err
			}

			if info.InstancePassword != "" && len(info.InstancePassword) < CORE.Config.GetI(CORE.MAIN_MIN_PASS_LENGTH) {
				terminal.Error("\nPassword can't be less than %s symbols.\n", CORE.Config.GetS(CORE.MAIN_MIN_PASS_LENGTH))
				continue
			}

			break
		}

		fmtc.NewLine()
	}

	if readOwner {
		for {
			info.Owner, err = terminal.Read("Please enter a new owner name (or leave blank to keep existing)", false)

			if err != nil {
				return nil, err
			}

			if info.Owner == "" || system.IsUserExist(info.Owner) {
				break
			} else {
				terminal.Error("\nUser %s doesn't exist on this system\n", info.Owner)
				continue
			}
		}

		fmtc.NewLine()
	}

	if readReplType {
		supportedReplTypes := []string{string(CORE.REPL_TYPE_REPLICA), string(CORE.REPL_TYPE_STANDBY)}

		for {
			info.ReplicationType, err = terminal.Read("Please enter a new replication type (or leave blank to keep existing)", false)

			if err != nil {
				return nil, err
			}

			if info.ReplicationType == "" || sliceutil.Contains(supportedReplTypes, info.ReplicationType) {
				break
			} else {
				terminal.Error(
					"\nUnsupported replication type. Only \"%s\" and \"%s\" is supported.\n",
					CORE.REPL_TYPE_REPLICA, CORE.REPL_TYPE_STANDBY,
				)
				continue
			}
		}

		fmtc.NewLine()
	}

	return info, nil
}
