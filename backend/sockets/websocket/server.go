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

package websocket

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/desertbit/glue/log"
	"github.com/desertbit/glue/utils"
	"github.com/gorilla/websocket"
)

//#############################//
//### WebSocket Server type ###//
//##################äää#####ää#//

type Server struct {
	// Websocket upgrader
	upgrader websocket.Upgrader

	onNewSocketConnection func(*Socket)
}

func NewServer(onNewSocketConnectionFunc func(*Socket)) *Server {
	return &Server{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Don't check the origin. This is done by the backend server package.
			CheckOrigin: func(r *http.Request) bool { return true },
		},

		onNewSocketConnection: onNewSocketConnectionFunc,
	}
}

func (s *Server) HandleRequest(rw http.ResponseWriter, req *http.Request) {
	// Get the remote address and user agent.
	remoteAddr, requestRemoteAddrMethodUsed := utils.RemoteAddress(req)
	userAgent := req.Header.Get("User-Agent")

	// This has to be a GET request.
	if req.Method != "GET" {
		log.L.WithFields(logrus.Fields{
			"remoteAddress": remoteAddr,
			"userAgent":     userAgent,
			"method":        req.Method,
		}).Warning("client accessed websocket handler with an invalid request method")

		http.Error(rw, "Method not allowed", 405)
		return
	}

	// Upgrade to a websocket.
	ws, err := s.upgrader.Upgrade(rw, req, nil)
	if err != nil {
		log.L.WithFields(logrus.Fields{
			"remoteAddress": remoteAddr,
			"userAgent":     userAgent,
		}).Warningf("failed to upgrade to websocket layer: %v", err)

		http.Error(rw, "Bad Request", 400)
		return
	}

	// Create a new websocket value.
	w := newSocket(ws)

	// Set the user agent.
	w.userAgent = userAgent

	// Set the remote address get function.
	if requestRemoteAddrMethodUsed {
		// Obtain the remote address from the websocket directly.
		w.remoteAddrFunc = func() string {
			return utils.RemovePortFromRemoteAddr(w.ws.RemoteAddr().String())
		}
	} else {
		// Obtain the remote address from the current string.
		// It was obtained using the request Headers. So don't use the
		// websocket RemoteAddr() method, because it does not return
		// the clients IP address.
		w.remoteAddrFunc = func() string {
			return remoteAddr
		}
	}

	// Start the handlers in new goroutines.
	go w.writeLoop()
	go w.readLoop()

	// Trigger the event that a new socket connection was made.
	s.onNewSocketConnection(w)
}
