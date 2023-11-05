package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/knf"
	"github.com/essentialkaos/ek/v12/path"
	"github.com/essentialkaos/ek/v12/system"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Preferences struct contains user-specific preferences
type Preferences struct {
	DisableTips     bool
	EnablePowerline bool
	SimpleUI        bool
}

// ////////////////////////////////////////////////////////////////////////////////// //

// GetPreferences reads preferences defined by current user
func GetPreferences() Preferences {
	currentUser, err := system.CurrentUser()

	if err != nil {
		return Preferences{}
	}

	return readUserPreferences(currentUser.RealName)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// readUserPreferences reads user preferences
func readUserPreferences(user string) Preferences {
	prefsFile := path.Join("/home", user, ".config", "rds", "preferences.knf")

	if !fsutil.CheckPerms("FRS", prefsFile) {
		return Preferences{}
	}

	prefsCfg, err := knf.Read(prefsFile)

	if err != nil {
		return Preferences{}
	}

	return Preferences{
		DisableTips:     prefsCfg.GetB("cli:disable-tips", false),
		EnablePowerline: prefsCfg.GetB("cli:enable-powerline", false),
		SimpleUI:        prefsCfg.GetB("cli:simple-ui", false),
	}
}
