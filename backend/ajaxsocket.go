/*
 *  Glue - Robust Go and Javascript Socket Library
 *  Copyright DesertBit
 *  Author: Roland Singer
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package backend

import (
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/desertbit/glue/backend/closer"
	"github.com/desertbit/glue/log"
	"github.com/desertbit/glue/utils"

	"github.com/Sirupsen/logrus"
)

const (
	// HTTP upgrader url.
	httpAjaxSocketUrl = httpBaseSocketUrl + "ajax"

	ajaxPollTimeout     = 35 * time.Second
	ajaxUIDLength       = 10
	ajaxPollTokenLength = 7

	// Ajax protocol commands.
	ajaxSocketDataDelimiter = "&"
	ajaxSocketDataKeyLength = 1
	ajaxSocketDataKeyInit   = "i"
	ajaxSocketDataKeyPush   = "u"
	ajaxSocketDataKeyPoll   = "o"
)

var (
	ajaxSockets map[string]*ajaxSocket = make(map[string]*ajaxSocket)
	ajaxMutex   sync.Mutex
)

func init() {
	// Create the ajax socket handler.
	http.HandleFunc(httpAjaxSocketUrl, handleAjaxSocket)
}

//#######################//
//### AjaxSocket type ###//
//#######################//

type ajaxSocket struct {
	uid       string
	pollToken string

	userAgent  string
	remoteAddr string

	closer *closer.Closer

	writeChannel chan string
	stopPoll     chan struct{}

	onClose OnCloseFunc
	onRead  OnReadFunc
}

// Create a new ajax socket.
func newAjaxSocket() *ajaxSocket {
	a := &ajaxSocket{
		writeChannel: make(chan string, 3),
		stopPoll:     make(chan struct{}),

		// Set a dummy function to not always
		// check if the method is not set.
		onRead: func(string) {},
	}

	// Set the closer function.
	a.closer = closer.New(func() {
		// Stop the polling goroutine if present.
		close(a.stopPoll)

		// Remove the ajax socket from the map.
		if len(a.uid) > 0 {
			ajaxMutex.Lock()
			delete(ajaxSockets, a.uid)
			ajaxMutex.Unlock()
		}

		// Trigger the onClose function if defined.
		if a.onClose != nil {
			a.onClose()
		}
	})

	return a
}

//############################################//
//### AjaxSocket - Interface implementation ###//
//############################################//

func (a *ajaxSocket) Type() SocketType {
	return TypeAjaxSocket
}

func (a *ajaxSocket) RemoteAddr() string {
	return a.remoteAddr
}

func (a *ajaxSocket) UserAgent() string {
	return a.userAgent
}

func (a *ajaxSocket) Close() {
	a.closer.Close()
}

func (a *ajaxSocket) OnClose(f OnCloseFunc) {
	a.onClose = f
}

func (a *ajaxSocket) IsClosed() bool {
	return a.closer.IsClosed()
}

func (a *ajaxSocket) Write(data string) {
	// Write the data to the channel.
	a.writeChannel <- data
}

func (a *ajaxSocket) OnRead(f OnReadFunc) {
	a.onRead = f
}

//############################//
//### AjaxSocket - Private ###//
//############################//

func handleAjaxSocket(w http.ResponseWriter, req *http.Request) {
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
		initAjaxRequest(remoteAddr, userAgent, w)
	case ajaxSocketDataKeyPoll:
		pollAjaxRequest(value, remoteAddr, userAgent, data, w)
	case ajaxSocketDataKeyPush:
		pushAjaxRequest(value, remoteAddr, userAgent, data, w)
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

func initAjaxRequest(remoteAddr, userAgent string, w http.ResponseWriter) {
	var uid string

	// Create a new ajax socket value.
	a := newAjaxSocket()
	a.remoteAddr = remoteAddr
	a.userAgent = userAgent

	func() {
		// Lock the mutex
		ajaxMutex.Lock()
		defer ajaxMutex.Unlock()

		// Obtain a new unique ID.
		for {
			// Generate it.
			uid = utils.RandomString(ajaxUIDLength)

			// Check if the new UID is already used.
			// This is very unlikely, but we have to check this!
			_, ok := ajaxSockets[uid]
			if !ok {
				// Break the loop if the UID is unique.
				break
			}
		}

		// Set the UID.
		a.uid = uid

		// Add the new ajax socket to the map.
		ajaxSockets[uid] = a
	}()

	// Create a new poll token.
	a.pollToken = utils.RandomString(ajaxPollTokenLength)

	// Tell the client the UID and poll token.
	io.WriteString(w, uid+ajaxSocketDataDelimiter+a.pollToken)

	// Trigger the event that a new socket connection was made.
	onNewSocketConnection(a)
}

func pushAjaxRequest(uid, remoteAddr, userAgent, data string, w http.ResponseWriter) {
	// Obtain the ajax socket with the uid.
	a := func() *ajaxSocket {
		// Lock the mutex.
		ajaxMutex.Lock()
		defer ajaxMutex.Unlock()

		// Obtain the ajax socket with the uid-
		a, ok := ajaxSockets[uid]
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

	// Trigger the onRead event function.
	a.onRead(data)
}

func pollAjaxRequest(uid, remoteAddr, userAgent, data string, w http.ResponseWriter) {
	// Obtain the ajax socket with the uid.
	a := func() *ajaxSocket {
		// Lock the mutex.
		ajaxMutex.Lock()
		defer ajaxMutex.Unlock()

		// Obtain the ajax socket with the uid-
		a, ok := ajaxSockets[uid]
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
	case data := <-a.writeChannel:
		// Send the new poll token and message data to the client.
		io.WriteString(w, a.pollToken+ajaxSocketDataDelimiter+data)
	case <-timeout.C:
		// Do nothing on timeout
		// Just release this goroutine.
		return
	case <-a.stopPoll:
		// Do nothing on timeout
		// Just release this go routine
		return
	}
}
