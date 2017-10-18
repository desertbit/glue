/*
 *  Glue - Robust Go and Javascript Socket Library
 *  Copyright (C) 2015  Roland Singer <roland.singer[at]desertbit.com>
 *
 *  This program is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU General Public License as published by
 *  the Free Software Foundation, either version 3 of the License, or
 *  (at your option) any later version.
 *
 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU General Public License for more details.
 *
 *  You should have received a copy of the GNU General Public License
 *  along with this program.  If not, see <http://www.gnu.org/licenses/>.
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
