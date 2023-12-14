package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"

	"github.com/essentialkaos/ek/v12/log"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type Logger struct{}

// ////////////////////////////////////////////////////////////////////////////////// //

// Error writes debug message to CLI log
func (l *Logger) Debug(id int, f string, a ...any) error {
	return log.Debug(l.getPrefix(id)+f, a...)
}

// Error writes info message to CLI log
func (l *Logger) Info(id int, f string, a ...any) error {
	return log.Info(l.getPrefix(id)+f, a...)
}

// Error writes warning message to CLI log
func (l *Logger) Warn(id int, f string, a ...any) error {
	return log.Warn(l.getPrefix(id)+f, a...)
}

// Error writes error message to CLI log
func (l *Logger) Error(id int, f string, a ...any) error {
	return log.Error(l.getPrefix(id)+f, a...)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// getPrefix returns log message prefix
func (l *Logger) getPrefix(id int) string {
	if id <= 0 {
		return fmt.Sprintf("(---|%s) ", CORE.User.RealName)
	}

	return fmt.Sprintf("(%3d|%s) ", id, CORE.User.RealName)
}
