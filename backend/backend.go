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

// Backend socket implementations.
package backend

const (
	httpBaseSocketUrl = "/glue/"

	// Channel buffer sizes:
	readChanSize  = 5
	writeChanSize = 10
)

var (
	onNewSocketConnection func(BackendSocket)
)

func init() {
	// Set a dummy function.
	// This prevents panics, if new sockets are created,
	// but no function was set.
	onNewSocketConnection = func(BackendSocket) {}
}

//############################//
//### Backend Socket Types ###//
//############################//

type SocketType int

const (
	// The available socket types.
	TypeAjaxSocket SocketType = 1 << iota
	TypeWebSocket  SocketType = 1 << iota
)

//################################//
//### Backend Socket Interface ###//
//################################//

type OnCloseFunc func()

type BackendSocket interface {
	Type() SocketType
	RemoteAddr() string
	UserAgent() string

	Close()
	IsClosed() bool
	OnClose(OnCloseFunc)

	WriteChan() chan string
	ReadChan() chan string
}

//##############//
//### Public ###//
//##############//

// OnNewSocketConnection sets the event function which is
// triggered if a new socket connection was made.
func OnNewSocketConnection(f func(BackendSocket)) {
	onNewSocketConnection = f
}

//###############//
//### Private ###//
//###############//

func triggerOnNewSocketConnection(bs BackendSocket) {
	// Trigger the on new socket connection event in a new goroutine
	// to not block any socket functions. Otherwise this might block HTTP handlers.
	go onNewSocketConnection(bs)
}
