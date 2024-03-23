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

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/fmtutil/barcode"
	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/pager"
	"github.com/essentialkaos/ek/v12/system"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// StopAllCommand is "settings" command handler
func SettingsCommand(args CommandArgs) int {
	var ec int

	if len(args) == 0 {
		ec = printAllSettings()
	} else {
		ec = printSpecificSettings(args)
	}

	return ec
}

// ////////////////////////////////////////////////////////////////////////////////// //

// printAllSettings prints all settings
func printAllSettings() int {
	if (options.GetB(OPT_PAGER) || prefs.AutoPaging) && !useRawOutput {
		if pager.Setup() == nil {
			defer pager.Complete()
		}
	}

	for _, section := range CORE.Config.Sections() {
		printSettingsSection(section)
	}

	fmtutil.Separator(true)

	return EC_OK
}

// printSpecificSettings prints specific settings
func printSpecificSettings(args CommandArgs) int {
	for i := range args {
		section := args.Get(i)
		if !CORE.Config.HasSection(section) {
			terminal.Error("Unknown settings section \"%s\"", section)
			return EC_ERROR
		}
	}

	for i := range args {
		printSettingsSection(args.Get(i))
	}

	fmtutil.Separator(true)

	return EC_OK
}

// printSettingsSection prints configuration section settings
func printSettingsSection(section string) {
	fmtutil.Separator(true)
	fmtc.Printf(" ▾ {*}%s{!}\n", strings.ToUpper(section))
	fmtutil.Separator(true)

	for _, prop := range CORE.Config.Props(section) {
		hidden := isPrivateSettingsProp(prop)
		printSettingsProperty(prop, CORE.Config.GetS(section+":"+prop), hidden)
	}
}

// printSettingsProperty prints formatted settings property
func printSettingsProperty(name, value string, hidden bool) {
	fmtc.Printf(" %-28s {s}|{!} ", name)

	switch {
	case value == "true":
		fmt.Println("Yes")

	case value == "false":
		fmt.Println("No")

	case value == "":
		fmtc.Println("{s-}[empty]{!}")

	case name == "auth-token" && hidden:
		fmt.Println(barcode.Dots([]byte(value)))

	case strings.HasPrefix(value, "/"):
		if fsutil.IsExist(value) {
			fmtc.Printf("%s {g}✔ {!}\n", value)
		} else {
			fmtc.Printf("%s {r}✖ {!}\n", value)
		}

	case name == "user":
		if system.IsUserExist(value) {
			fmtc.Printf("%s {g}✔ {!}\n", value)
		} else {
			fmtc.Printf("%s {r}✖ {!}\n", value)
		}

	case name == "virtual-ip" && value != "":
		switch CORE.GetKeepalivedState() {
		case CORE.KEEPALIVED_STATE_UNKNOWN:
			fmtc.Println(value)
		case CORE.KEEPALIVED_STATE_MASTER:
			fmtc.Printf("%s {g}(master){!}\n", value)
		case CORE.KEEPALIVED_STATE_BACKUP:
			fmtc.Printf("%s {s}(backup){!}\n", value)
		}

	case hidden:
		fmtc.Println("{s-}[hidden]{!}")

	default:
		fmt.Println(value)
	}
}

// isPrivateSettingsProp returns true if property must be hidden due to security
func isPrivateSettingsProp(prop string) bool {
	if options.GetB(OPT_PRIVATE) {
		return false
	}

	if prop == "auth-token" || strings.Contains(prop, "-pass-length") {
		return true
	}

	return false
}
