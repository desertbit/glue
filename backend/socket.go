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
	OnClose(f func())

	WriteChan() chan string
	ReadChan() chan string
}
