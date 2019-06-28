/*
 *  Glue - Robust Go and Javascript Socket Library
 *  Copyright (C) 2015  Roland Singer <roland.singer[at]desertbit.com>
 * 
 *  SPDX-License-Identifier: MIT
 */

package backend

import "github.com/desertbit/glue/backend/global"

//################################//
//### Backend Socket Interface ###//
//################################//

type BackendSocket interface {
	Type() global.SocketType
	RemoteAddr() string
	UserAgent() string

	Close()
	IsClosed() bool
	ClosedChan() <-chan struct{}

	WriteChan() chan string
	ReadChan() chan string
}
