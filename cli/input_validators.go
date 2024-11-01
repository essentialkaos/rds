package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"

	"github.com/essentialkaos/ek/v13/system"
	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type inputValidatorDesc struct{}
type inputValidatorPassword struct{}
type inputValidatorOwner struct{}

type inputValidatorRole struct {
	Default string
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Validate validates instance description input
func (v inputValidatorDesc) Validate(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	if len(input) < CORE.MIN_DESC_LENGTH {
		return input, fmt.Errorf("Description must at least %d symbols long", CORE.MIN_DESC_LENGTH)
	}

	if len(input) > CORE.MAX_DESC_LENGTH {
		return input, fmt.Errorf("Description must be less than %d symbols long", CORE.MAX_DESC_LENGTH)
	}

	return input, nil
}

// Validate validates instance password input
func (v inputValidatorPassword) Validate(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	if len(input) < CORE.Config.GetI(CORE.MAIN_MIN_PASS_LENGTH) {
		return input, fmt.Errorf(
			"Password can't be less than %d symbols",
			CORE.Config.GetI(CORE.MAIN_MIN_PASS_LENGTH),
		)
	}

	return input, nil
}

// Validate validates instance role input
func (v inputValidatorRole) Validate(input string) (string, error) {
	if input == "" {
		return v.Default, nil
	}

	switch input {
	case "1", "S", "s":
		return string(CORE.REPL_TYPE_STANDBY), nil
	case "2", "R", "r":
		return string(CORE.REPL_TYPE_REPLICA), nil
	}

	return input, fmt.Errorf("Unsupported value, please enter R of S")
}

// Validate validates instance owner user input
func (v inputValidatorOwner) Validate(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	if !system.IsUserExist(input) {
		return input, fmt.Errorf("User %q doesn't exist on this system", input)
	}

	return input, nil
}
