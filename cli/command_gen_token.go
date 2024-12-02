package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"

	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fmtutil"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// GenTokenCommand is "gen-token" command handler
func GenTokenCommand(args CommandArgs) int {
	token := CORE.GenerateToken()

	if !useRawOutput {
		fmtutil.Separator(true)
		fmtc.Printfn("\n  {*}Token:{!} %s\n", token)
		fmtutil.Separator(true)
	} else {
		fmt.Println(token)
	}

	return EC_OK
}
