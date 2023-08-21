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
	"github.com/essentialkaos/ek/v12/terminal"

	API "github.com/essentialkaos/rds/api"
	CORE "github.com/essentialkaos/rds/core"
	SC "github.com/essentialkaos/rds/sync/client"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// TagAddCommand is "tag-add" command handler
func TagAddCommand(args CommandArgs) int {
	err := args.Check(false)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	if !args.Has(1) {
		terminal.Warn("You must define tag")
		return EC_ERROR
	}

	id, _, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	tag := args.Get(1)
	meta, err := CORE.GetInstanceMeta(id)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	err = CORE.AddTag(id, tag)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	err = SC.PropagateCommand(API.COMMAND_EDIT, meta.ID, meta.UUID)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	// We use renderTags function from listing
	fmtc.Printf("Tag "+renderTags(tag)+" added to instance %d\n", id)
	log.Info("(%s) Added tag \"%s\" to instance %d", CORE.User.RealName, tag, id)

	return EC_OK
}

// TagRemoveCommand is "tag-remove" command handler
func TagRemoveCommand(args CommandArgs) int {
	err := args.Check(false)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	if !args.Has(1) {
		terminal.Warn("You must define tag")
		return EC_ERROR
	}

	id, _, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	tag := args.Get(1)
	meta, err := CORE.GetInstanceMeta(id)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	tagName, _ := CORE.ParseTag(tag)

	err = CORE.RemoveTag(id, tagName)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	err = SC.PropagateCommand(API.COMMAND_EDIT, meta.ID, meta.UUID)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	fmtc.Printf("{g}Tag \"%s\" removed from instance %d{!}\n", tagName, id)

	log.Info("(%s) Removed tag \"%s\" from instance %d", CORE.User.RealName, tagName, id)

	return EC_OK
}
