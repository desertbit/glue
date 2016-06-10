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

package ajaxsocket

import (
	"github.com/desertbit/glue/backend/closer"
	"github.com/desertbit/glue/backend/global"
)

//########################//
//### Ajax Socket type ###//
//########################//

type Socket struct {
	uid        string
	pollToken  string
	userAgent  string
	remoteAddr string

	closer *closer.Closer

	writeChan chan string
	readChan  chan string
}

// Create a new ajax socket.
func newSocket(s *Server) *Socket {
	a := &Socket{
		writeChan: make(chan string, global.WriteChanSize),
		readChan:  make(chan string, global.ReadChanSize),
	}

	// Set the closer function.
	a.closer = closer.New(func() {
		// Remove the ajax socket from the map.
		if len(a.uid) > 0 {
			func() {
				s.socketsMutex.Lock()
				defer s.socketsMutex.Unlock()

				delete(s.sockets, a.uid)
			}()
		}
	})

	return a
}

//##############################################//
//### Ajax Socket - Interface implementation ###//
//##############################################//

func (s *Socket) Type() global.SocketType {
	return global.TypeAjaxSocket
}

func (s *Socket) RemoteAddr() string {
	return s.remoteAddr
}

func (s *Socket) UserAgent() string {
	return s.userAgent
}

func (s *Socket) Close() {
	s.closer.Close()
}

func (s *Socket) IsClosed() bool {
	return s.closer.IsClosed()
}

func (s *Socket) ClosedChan() <-chan struct{} {
	return s.closer.IsClosedChan
}

func (s *Socket) WriteChan() chan string {
	return s.writeChan
}

func (s *Socket) ReadChan() chan string {
	return s.readChan
}
