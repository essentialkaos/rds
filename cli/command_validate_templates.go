package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/terminal"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// ValidateTemplatesCommand is "validate-templates" command handler
func ValidateTemplatesCommand(args CommandArgs) int {
	errs := CORE.ValidateTemplates()

	if len(errs) == 0 {
		fmtc.Println("{g}Redis and Sentinel configuration templates have no problems{!}")
		return EC_OK
	}

	terminal.Error("Templates validation errors:\n")

	for _, err := range errs {
		terminal.Error("- %v", err)
	}

	return EC_ERROR
}
