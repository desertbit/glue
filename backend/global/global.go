/*
 *  Glue - Robust Go and Javascript Socket Library
 *  Copyright (C) 2015  Roland Singer <roland.singer[at]desertbit.com>
 * 
 *  SPDX-License-Identifier: MIT
 */

// Package global provides global types and constants for the backend packages.
package global

const (
	// Channel buffer sizes:
	ReadChanSize  = 5
	WriteChanSize = 10
)

//############################//
//### Backend Socket Types ###//
//############################//

type SocketType int

const (
	// The available socket types.
	TypeAjaxSocket SocketType = 1 << iota
	TypeWebSocket  SocketType = 1 << iota
)
