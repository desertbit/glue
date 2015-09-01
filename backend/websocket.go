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

import (
	"io"
	"net/http"
	"time"

	"github.com/desertbit/glue/backend/closer"
	"github.com/desertbit/glue/log"
	"github.com/desertbit/glue/utils"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
)

const (
	// HTTP upgrader url.
	httpWebSocketURL = httpBaseSocketURL + "ws"

	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next message from the peer.
	readWait = 60 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 0
)

var (
	// Websocket upgrader
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func init() {
	// Create the websocket handler
	http.HandleFunc(httpWebSocketURL, handleWebSocket)
}

//######################//
//### WebSocket type ###//
//######################//

type webSocket struct {
	ws *websocket.Conn

	closer  *closer.Closer
	onClose OnCloseFunc

	writeChan chan string
	readChan  chan string

	userAgent      string
	remoteAddrFunc func() string
}

// Create a new websocket type.
func newWebSocket(ws *websocket.Conn) *webSocket {
	w := &webSocket{
		ws:        ws,
		writeChan: make(chan string, writeChanSize),
		readChan:  make(chan string, readChanSize),
	}

	// Set the closer function.
	w.closer = closer.New(func() {
		// Send a close message to the client.
		// Ignore errors.
		w.write(websocket.CloseMessage, []byte{})

		// Close the socket.
		w.ws.Close()

		// Trigger the onClose function if defined.
		if w.onClose != nil {
			w.onClose()
		}
	})

	return w
}

//############################################//
//### WebSocket - Interface implementation ###//
//############################################//

func (w *webSocket) Type() SocketType {
	return TypeWebSocket
}

func (w *webSocket) RemoteAddr() string {
	return w.remoteAddrFunc()
}

func (w *webSocket) UserAgent() string {
	return w.userAgent
}

func (w *webSocket) Close() {
	w.closer.Close()
}

func (w *webSocket) OnClose(f OnCloseFunc) {
	w.onClose = f
}

func (w *webSocket) IsClosed() bool {
	return w.closer.IsClosed()
}

func (w *webSocket) WriteChan() chan string {
	return w.writeChan
}

func (w *webSocket) ReadChan() chan string {
	return w.readChan
}

//###########################//
//### WebSocket - Private ###//
//###########################//

// write writes a message with the given message type and payload.
func (w *webSocket) write(mt int, payload []byte) error {
	w.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return w.ws.WriteMessage(mt, payload)
}

// readLoop reads messages from the websocket
func (w *webSocket) readLoop() {
	defer func() {
		// Close the socket on defer.
		w.Close()
	}()

	// Set the limits.
	w.ws.SetReadLimit(maxMessageSize)

	// Set the pong handler.
	w.ws.SetPongHandler(func(string) error {
		// Reset the read deadline.
		w.ws.SetReadDeadline(time.Now().Add(readWait))
		return nil
	})

	for {
		// Reset the read deadline.
		w.ws.SetReadDeadline(time.Now().Add(readWait))

		// Read from the websocket.
		_, data, err := w.ws.ReadMessage()
		if err != nil {
			// Only log errors if this is not EOF and
			// if the socket was not closed already.
			if err != io.EOF && !w.IsClosed() {
				log.L.WithFields(logrus.Fields{
					"remoteAddress": w.RemoteAddr(),
					"userAgent":     w.UserAgent(),
				}).Warningf("failed to read data from websocket: %v", err)
			}
			return
		}

		// Write the received data to the read channel.
		w.readChan <- string(data)
	}
}

func (w *webSocket) writeLoop() {
	for {
		select {
		case data := <-w.writeChan:
			// Write the data to the websocket.
			err := w.write(websocket.TextMessage, []byte(data))
			if err != nil {
				log.L.WithFields(logrus.Fields{
					"remoteAddress": w.RemoteAddr(),
					"userAgent":     w.UserAgent(),
				}).Warningf("failed to write to websocket: %v", err)

				// Close the websocket on error.
				w.Close()
				return
			}

		case <-w.closer.IsClosedChan:
			// Just release this loop.
			return
		}
	}
}

//####################//
//### HTTP Handler ###//
//####################//

func handleWebSocket(rw http.ResponseWriter, req *http.Request) {
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
	ws, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		log.L.WithFields(logrus.Fields{
			"remoteAddress": remoteAddr,
			"userAgent":     userAgent,
		}).Warningf("failed to upgrade to websocket layer: %v", err)

		http.Error(rw, "Bad Request", 400)
		return
	}

	// Create a new websocket value.
	w := newWebSocket(ws)

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
	triggerOnNewSocketConnection(w)
}
