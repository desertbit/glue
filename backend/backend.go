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

// Package backend provides various socket implementations.
package backend

const (
	httpBaseSocketURL = "/glue/"

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
