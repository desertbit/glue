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

// Package ajaxsocket provides the ajax socket implementation.
package ajaxsocket

import (
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/desertbit/glue/log"
	"github.com/desertbit/glue/utils"
)

//#################//
//### Constants ###//
//#################//

const (
	ajaxPollTimeout     = 35 * time.Second
	ajaxUIDLength       = 10
	ajaxPollTokenLength = 7

	// Ajax poll data commands:
	ajaxPollCmdTimeout = "t"
	ajaxPollCmdClosed  = "c"

	// Ajax protocol commands:
	ajaxSocketDataDelimiter = "&"
	ajaxSocketDataKeyLength = 1
	ajaxSocketDataKeyInit   = "i"
	ajaxSocketDataKeyPush   = "u"
	ajaxSocketDataKeyPoll   = "o"
)

//########################//
//### Ajax Server type ###//
//##################äääää#//

type Server struct {
	sockets      map[string]*Socket
	socketsMutex sync.Mutex

	onNewSocketConnection func(*Socket)
}

func NewServer(onNewSocketConnectionFunc func(*Socket)) *Server {
	return &Server{
		sockets:               make(map[string]*Socket),
		onNewSocketConnection: onNewSocketConnectionFunc,
	}
}

func (s *Server) HandleRequest(w http.ResponseWriter, req *http.Request) {
	// Get the remote address and user agent.
	remoteAddr, _ := utils.RemoteAddress(req)
	userAgent := req.Header.Get("User-Agent")

	// Get the request body data.
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.L.WithFields(logrus.Fields{
			"remoteAddress": remoteAddr,
			"userAgent":     userAgent,
		}).Warningf("failed to read ajax request body: %v", err)

		http.Error(w, "Internal Server Error", 500)
		return
	}

	// Check for bad requests.
	if req.Method != "POST" {
		log.L.WithFields(logrus.Fields{
			"remoteAddress": remoteAddr,
			"userAgent":     userAgent,
		}).Warningf("client accessed the ajax interface with an invalid http method: %s", req.Method)

		http.Error(w, "Bad Request", 400)
		return
	}

	// Get the request body as string.
	data := string(body)

	// Get the head of the body data delimited by an delimiter.
	var head string
	i := strings.Index(data, ajaxSocketDataDelimiter)
	if i < 0 {
		// There is no delimiter. The complete data is the head.
		head = data
		data = ""
	} else {
		// Extract the head.
		head = data[:i]
		data = data[i+1:]
	}

	// Validate the head length.
	if len(head) < ajaxSocketDataKeyLength {
		log.L.WithFields(logrus.Fields{
			"remoteAddress": remoteAddr,
			"userAgent":     userAgent,
		}).Warningf("ajax: head data is too short: '%s'", head)

		http.Error(w, "Bad Request", 400)
		return
	}

	// The head is split into key and value.
	key := head[:ajaxSocketDataKeyLength]
	value := head[ajaxSocketDataKeyLength:]

	// Handle the specific request.
	switch key {
	case ajaxSocketDataKeyInit:
		s.initAjaxRequest(remoteAddr, userAgent, w)
	case ajaxSocketDataKeyPoll:
		s.pollAjaxRequest(value, remoteAddr, userAgent, data, w)
	case ajaxSocketDataKeyPush:
		s.pushAjaxRequest(value, remoteAddr, userAgent, data, w)
	default:
		log.L.WithFields(logrus.Fields{
			"remoteAddress": remoteAddr,
			"userAgent":     userAgent,
			"key":           key,
			"value":         value,
		}).Warningf("ajax: invalid request.")

		http.Error(w, "Bad Request", 400)
		return
	}
}

func (s *Server) initAjaxRequest(remoteAddr, userAgent string, w http.ResponseWriter) {
	var uid string

	// Create a new ajax socket value.
	a := newSocket(s)
	a.remoteAddr = remoteAddr
	a.userAgent = userAgent

	func() {
		// Lock the mutex
		s.socketsMutex.Lock()
		defer s.socketsMutex.Unlock()

		// Obtain a new unique ID.
		for {
			// Generate it.
			uid = utils.RandomString(ajaxUIDLength)

			// Check if the new UID is already used.
			// This is very unlikely, but we have to check this!
			_, ok := s.sockets[uid]
			if !ok {
				// Break the loop if the UID is unique.
				break
			}
		}

		// Set the UID.
		a.uid = uid

		// Add the new ajax socket to the map.
		s.sockets[uid] = a
	}()

	// Create a new poll token.
	a.pollToken = utils.RandomString(ajaxPollTokenLength)

	// Tell the client the UID and poll token.
	io.WriteString(w, uid+ajaxSocketDataDelimiter+a.pollToken)

	// Trigger the event that a new socket connection was made.
	s.onNewSocketConnection(a)
}

func (s *Server) pushAjaxRequest(uid, remoteAddr, userAgent, data string, w http.ResponseWriter) {
	// Obtain the ajax socket with the uid.
	a := func() *Socket {
		// Lock the mutex.
		s.socketsMutex.Lock()
		defer s.socketsMutex.Unlock()

		// Obtain the ajax socket with the uid-
		a, ok := s.sockets[uid]
		if !ok {
			return nil
		}
		return a
	}()

	if a == nil {
		log.L.WithFields(logrus.Fields{
			"remoteAddress": remoteAddr,
			"userAgent":     userAgent,
			"uid":           uid,
		}).Warningf("ajax: client requested an invalid ajax socket: uid is invalid!")

		http.Error(w, "Bad Request", 400)
		return
	}

	// The user agents have to match.
	if a.userAgent != userAgent {
		log.L.WithFields(logrus.Fields{
			"remoteAddress":   remoteAddr,
			"userAgent":       userAgent,
			"uid":             uid,
			"clientUserAgent": userAgent,
			"socketUserAgent": a.userAgent,
		}).Warningf("ajax: client push request: user agents do not match!")

		http.Error(w, "Bad Request", 400)
		return
	}

	// Check if the push request was called with no data.
	if len(data) == 0 {
		log.L.WithFields(logrus.Fields{
			"remoteAddress": remoteAddr,
			"userAgent":     userAgent,
			"uid":           uid,
		}).Warningf("ajax: client push request with no data!")

		http.Error(w, "Bad Request", 400)
		return
	}

	// Update the remote address. The client might be behind a proxy.
	a.remoteAddr = remoteAddr

	// Write the received data to the read channel.
	a.readChan <- data
}

func (s *Server) pollAjaxRequest(uid, remoteAddr, userAgent, data string, w http.ResponseWriter) {
	// Obtain the ajax socket with the uid.
	a := func() *Socket {
		// Lock the mutex.
		s.socketsMutex.Lock()
		defer s.socketsMutex.Unlock()

		// Obtain the ajax socket with the uid-
		a, ok := s.sockets[uid]
		if !ok {
			return nil
		}
		return a
	}()

	if a == nil {
		log.L.WithFields(logrus.Fields{
			"remoteAddress": remoteAddr,
			"userAgent":     userAgent,
			"uid":           uid,
		}).Warningf("ajax: client requested an invalid ajax socket: uid is invalid!")

		http.Error(w, "Bad Request", 400)
		return
	}

	// The user agents have to match.
	if a.userAgent != userAgent {
		log.L.WithFields(logrus.Fields{
			"remoteAddress":   remoteAddr,
			"userAgent":       userAgent,
			"uid":             uid,
			"clientUserAgent": userAgent,
			"socketUserAgent": a.userAgent,
		}).Warningf("ajax: client poll request: user agents do not match!")

		http.Error(w, "Bad Request", 400)
		return
	}

	// Check if the poll tokens matches.
	// The poll token is the data value.
	if a.pollToken != data {
		log.L.WithFields(logrus.Fields{
			"remoteAddress":   remoteAddr,
			"userAgent":       userAgent,
			"uid":             uid,
			"clientPollToken": data,
			"socketPollToken": a.pollToken,
		}).Warningf("ajax: client poll request: poll tokens do not match!")

		http.Error(w, "Bad Request", 400)
		return
	}

	// Create a new poll token.
	a.pollToken = utils.RandomString(ajaxPollTokenLength)

	// Create a timeout timer for the poll.
	timeout := time.NewTimer(ajaxPollTimeout)

	defer func() {
		// Stop the timeout timer.
		timeout.Stop()
	}()

	// Send messages as soon as there are some available.
	select {
	case data := <-a.writeChan:
		// Send the new poll token and message data to the client.
		io.WriteString(w, a.pollToken+ajaxSocketDataDelimiter+data)
	case <-timeout.C:
		// Tell the client that this ajax connection has reached the timeout.
		io.WriteString(w, ajaxPollCmdTimeout)
	case <-a.closer.IsClosedChan:
		// Tell the client that this ajax connection is closed.
		io.WriteString(w, ajaxPollCmdClosed)
	}
}
