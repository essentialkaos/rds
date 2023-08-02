package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/log"
	"github.com/essentialkaos/ek/v12/passwd"
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
		terminal.Error(err.Error())
		return EC_ERROR
	}

	id, _, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	state, err := CORE.GetInstanceState(id, true)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	err = showInstanceBasicInfoCard(id, state)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	ok, err := terminal.ReadAnswer(
		"Do you want to modify meta for this instance?", "Y",
	)

	if !ok || err != nil {
		return EC_ERROR
	}

	meta, err := CORE.GetInstanceMeta(id)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	fmtc.NewLine()

	desc, pass, user, replType, err := readEditInfo(true, true, true, true)

	if err != nil {
		if err == terminal.ErrKillSignal {
			return EC_OK
		}

		terminal.Error(err.Error())
		return EC_ERROR
	}

	// It's safe to modify this metadata, because GetInstanceMeta returns
	// copy of metadata
	if pass != "" {
		pepper := passwd.GenPassword(32, passwd.STRENGTH_MEDIUM)
		hash, err := passwd.Encrypt(pass, pepper)

		if err != nil {
			terminal.Error(err.Error())
			return EC_ERROR
		}

		meta.AuthInfo.Pepper = pepper
		meta.AuthInfo.Hash = hash
	}

	if user != "" {
		meta.AuthInfo.User = user
	}

	if desc != "" {
		meta.Desc = desc
	}

	if replType != "" {
		meta.ReplicationType = CORE.ReplicationType(replType)
	}

	err = CORE.UpdateInstance(meta)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	log.Info("(%s) Updated info for instance with ID %d", CORE.User.RealName, id)
	fmtc.Printf("{g}Done. Data for instance with ID %d successfully updated.{!}\n", id)

	err = SC.PropagateCommand(API.COMMAND_EDIT, meta.ID, meta.UUID)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Read user input for edit command
func readEditInfo(readDesc, readPass, readOwner, readReplType bool) (string, string, string, string, error) {
	var desc, pass, user, replType string
	var err error

	if readDesc {
		desc, err = terminal.Read("Please enter a new description (or leave blank to keep existing)", false)

		if err != nil {
			return "", "", "", "", err
		}

		fmtc.NewLine()
	}

	if readPass {
		for {
			pass, err = terminal.ReadPassword("Please enter a new password (or leave blank to keep existing)", false)

			if err != nil {
				return "", "", "", "", err
			}

			if pass != "" && len(pass) < CORE.Config.GetI(CORE.MAIN_MIN_PASS_LENGTH) {
				terminal.Error("\nPassword can't be less than %s symbols.\n", CORE.Config.GetS(CORE.MAIN_MIN_PASS_LENGTH))
				continue
			}

			break
		}

		fmtc.NewLine()
	}

	if readOwner {
		for {
			user, err = terminal.Read("Please enter a new owner name (or leave blank to keep existing)", false)

			if err != nil {
				return "", "", "", "", err
			}

			if user == "" || system.IsUserExist(user) {
				break
			} else {
				terminal.Error("\nUser %s doesn't exist on this system\n", user)
				continue
			}
		}

		fmtc.NewLine()
	}

	if readReplType {
		supportedReplTypes := []string{string(CORE.REPL_TYPE_REPLICA), string(CORE.REPL_TYPE_STANDBY)}

		for {
			replType, err = terminal.Read("Please enter a new replication type (or leave blank to keep existing)", false)

			if err != nil {
				return "", "", "", "", err
			}

			if replType == "" || sliceutil.Contains(supportedReplTypes, replType) {
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

	return desc, pass, user, replType, nil
}
