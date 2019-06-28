/*
 *  Glue - Robust Go and Javascript Socket Library
 *  Copyright (C) 2015  Roland Singer <roland.singer[at]desertbit.com>
 * 
 *  SPDX-License-Identifier: MIT
 */

// Package log holds the log backend used by the socket library.
// Use the logrus L value to adapt the log formatting
// or log levels if required...
package log

import (
	"github.com/sirupsen/logrus"
)

var (
	// L is the public logrus value used internally by glue.
	L = logrus.New()
)

func init() {
	// Set the default log options.
	L.Formatter = new(logrus.TextFormatter)
	L.Level = logrus.DebugLevel
}
