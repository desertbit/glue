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
