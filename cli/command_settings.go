package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"strings"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/strutil"
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
	for _, section := range CORE.Config.Sections() {
		fmtutil.Separator(true)
		fmtc.Printf(" ▾ {*}%s{!}\n", strings.ToUpper(section))
		fmtutil.Separator(true)

		for _, prop := range CORE.Config.Props(section) {
			hidden := isPrivateSettingsProp(prop)
			printSettingsProperty(prop, CORE.Config.GetS(section+":"+prop), hidden)
		}
	}

	fmtutil.Separator(true)

	return EC_OK
}

// printSpecificSettings prints specific settings
func printSpecificSettings(args CommandArgs) int {
	for i := range args {
		propName := args.Get(i)
		if !CORE.Config.HasProp(propName) {
			terminal.Error("Unknown settings property \"%s\"", propName)
			return EC_ERROR
		}
	}

	var curSection string

	for i := range args {
		propName := args.Get(i)
		section := strutil.ReadField(propName, 0, false, ":")
		prop := strutil.ReadField(propName, 1, false, ":")

		if curSection != section {
			fmtutil.Separator(true)
			fmtc.Printf(" ▾ {*}%s{!}\n", strings.ToUpper(section))
			fmtutil.Separator(true)
			curSection = section
		}

		hidden := isPrivateSettingsProp(prop)
		printSettingsProperty(prop, CORE.Config.GetS(propName), hidden)
	}

	fmtutil.Separator(true)

	return EC_OK
}

// printSettingsProperty prints formatted settings property
func printSettingsProperty(name, value string, hidden bool) {
	fmtc.Printf(" %-28s {s}|{!} ", name)

	switch {
	case hidden:
		fmtc.Println("{s-}[HIDDEN]{!}")

	case value == "true":
		fmtc.Println("Yes")

	case value == "false":
		fmtc.Println("No")

	case value == "":
		fmtc.Println("{s-}-empty-{!}")

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

	default:
		fmtc.Println(value)
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
