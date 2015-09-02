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

// Package backend provides the server backend with various socket implementations.
package backend

import (
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/desertbit/glue/backend/sockets/ajaxsocket"
	"github.com/desertbit/glue/backend/sockets/websocket"
	"github.com/desertbit/glue/log"
	"github.com/desertbit/glue/utils"
)

//#################//
//### Constants ###//
//#################//

const (
	httpURLAjaxSocketSuffix = "ajax"
	httpURLWebSocketSuffix  = "ws"
)

//######################//
//### Backend Server ###//
//######################//

type Server struct {
	onNewSocketConnection func(BackendSocket)

	// An Integer holding the length of characters which should be stripped
	// from the ServerHTTP URL path.
	httpURLStripLength int

	// Socket Servers
	webSocketServer  *websocket.Server
	ajaxSocketServer *ajaxsocket.Server
}

func NewServer(httpURLStripLength int) *Server {
	// Create a new backend server.
	s := &Server{
		// Set a dummy function.
		// This prevents panics, if new sockets are created,
		// but no function was set.
		onNewSocketConnection: func(BackendSocket) {},

		httpURLStripLength: httpURLStripLength,
	}

	// Create the websocket server and pass the function which handles new incoming socket connections.
	s.webSocketServer = websocket.NewServer(func(ws *websocket.Socket) {
		s.triggerOnNewSocketConnection(ws)
	})

	// Create the ajax server and pass the function which handles new incoming socket connections.
	s.ajaxSocketServer = ajaxsocket.NewServer(func(as *ajaxsocket.Socket) {
		s.triggerOnNewSocketConnection(as)
	})

	return s
}

// OnNewSocketConnection sets the event function which is
// triggered if a new socket connection was made.
func (s *Server) OnNewSocketConnection(f func(BackendSocket)) {
	s.onNewSocketConnection = f
}

// ServeHTTP implements the HTTP Handler interface of the http package.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get the URL path.
	path := r.URL.Path

	// Call this in an inline function to handle errors.
	err := func() error {
		// Strip the base URL.
		if len(path) < s.httpURLStripLength {
			return fmt.Errorf("invalid request")
		}
		path = path[s.httpURLStripLength:]

		// Route the HTTP request in a very simple way by comparing the strings.
		if path == httpURLWebSocketSuffix {
			// Handle the websocket request.
			s.webSocketServer.HandleRequest(w, r)
		} else if path == httpURLAjaxSocketSuffix {
			// Handle the ajax request.
			s.ajaxSocketServer.HandleRequest(w, r)
		} else {
			return fmt.Errorf("invalid request")
		}

		return nil
	}()

	// Handle the error.
	if err != nil {
		// Get the remote address and user agent.
		remoteAddr, _ := utils.RemoteAddress(r)
		userAgent := r.Header.Get("User-Agent")

		// Log the invalid request.
		log.L.WithFields(logrus.Fields{
			"remoteAddress": remoteAddr,
			"userAgent":     userAgent,
			"url":           r.URL.Path,
		}).Warningf("handle HTTP request: %v", err)

		http.Error(w, "Bad Request", 400)
	}
}

//################################//
//### Backend Server - Private ###//
//################################//

func (s *Server) triggerOnNewSocketConnection(bs BackendSocket) {
	// Trigger the on new socket connection event in a new goroutine
	// to not block any socket functions. Otherwise this might block HTTP handlers.
	go s.onNewSocketConnection(bs)
}
