/*
 *  Glue - Robust Go and Javascript Socket Library
 *  Copyright DesertBit
 *  Author: Roland Singer
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

// Package log holds the log backend used by the socket library.
// Use the logrus L value to adapt the log formatting
// or log levels if required...
package log

import (
	"github.com/Sirupsen/logrus"
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
