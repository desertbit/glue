/*
 *  Glue - Robust Go and Javascript Socket Library
 *  Copyright (C) 2015  Roland Singer <roland.singer[at]desertbit.com>
 * 
 *  SPDX-License-Identifier: MIT
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
