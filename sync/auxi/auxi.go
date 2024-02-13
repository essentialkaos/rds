package auxi

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"github.com/essentialkaos/ek/v12/strutil"

	API "github.com/essentialkaos/rds/api"
	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// GetCoreCompatibility returns RDS core versions compatibility
func GetCoreCompatibility(version string) API.CoreCompatibility {
	_, coreVer := parseVersionString(version)

	if coreVer == "" {
		return API.CORE_COMPAT_ERROR
	}

	switch {
	case coreVer == CORE.VERSION:
		return API.CORE_COMPAT_OK
	case coreVer[0:1] == CORE.VERSION[0:1]:
		return API.CORE_COMPAT_PARTIAL
	default:
		return API.CORE_COMPAT_ERROR
	}
}

// parseVersionString parse and return app version and core version
func parseVersionString(version string) (string, string) {
	return strutil.ReadField(version, 0, false, '/'),
		strutil.ReadField(version, 1, false, '/')
}
