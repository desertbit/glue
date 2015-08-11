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

// Package glue - Robust Go and Javascript Socket Library.
// This library is thread-safe.
package glue

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/desertbit/glue/backend"
	"github.com/desertbit/glue/log"
	"github.com/desertbit/glue/utils"
)

//#################//
//### Constants ###//
//#################//

// Public
// ######
const (
	// Version holds the Glue Socket Protocol Version as string.
	Version = "1.2.0"
)

// Private
// #######
const (
	// Send pings to the peer with this period.
	pingPeriod = 30 * time.Second

	// Kill the socket after this timeout.
	pingResponseTimeout = 7 * time.Second

	// The main channel name.
	mainChannelName = "m"

	// Socket commands. Must be two character long.
	// ############################################
	cmdLen         = 2
	cmdInit        = "in"
	cmdPing        = "pi"
	cmdPong        = "po"
	cmdClose       = "cl"
	cmdInvalid     = "iv"
	cmdChannelData = "cd"
)

//#################//
//### Variables ###//
//#################//

// Public errors:
var (
	ErrSocketClosed = errors.New("the socket connection is closed")
	ErrReadTimeout  = errors.New("the read timeout was reached")
)

var (
	// Private
	// #######

	block       bool
	onNewSocket = func(*Socket) {} // Initialize with dummy function to remove nil check.

	sockets      = make(map[string]*Socket) // A map holding all active current sockets.
	socketsMutex sync.Mutex
)

//####################//
//### Public Types ###//
//####################//

// OnNewSocketFunc is an event function.
type OnNewSocketFunc func(s *Socket)

// OnCloseFunc is an event function.
type OnCloseFunc func()

// OnReadFunc is an event function.
type OnReadFunc func(data string)

//#####################//
//### Private Types ###//
//#####################//

type initData struct {
	// Hint: Currently not used.
	// Placeholder for encryption.
}

type clientInitData struct {
	Version string `json:"version"`
}

//###################//
//### Socket Type ###//
//###################//

// A Socket represents a single socket connections to a client.
type Socket struct {
	// A Value is a placeholder for custom data.
	// Use this to attach socket specific data.
	Value interface{}

	// Private
	// #######

	id string // Unique socket ID.
	bs backend.BackendSocket

	channels    *channels
	mainChannel *Channel

	writeChan    chan string
	readChan     chan string
	isClosedChan chan struct{}

	onCloseFunc OnCloseFunc

	pingTimer         *time.Timer
	pingTimeout       *time.Timer
	sendPingMutex     sync.Mutex
	pingRequestActive bool
}

// newSocket creates a new socket and initializes it.
func newSocket(bs backend.BackendSocket) *Socket {
	// Create a new socket value.
	s := &Socket{
		id:       utils.UUID(),
		bs:       bs,
		channels: newChannels(),

		writeChan:    bs.WriteChan(),
		readChan:     bs.ReadChan(),
		isClosedChan: make(chan struct{}),

		pingTimer:   time.NewTimer(pingPeriod),
		pingTimeout: time.NewTimer(pingResponseTimeout),
	}

	// Create the main channel.
	s.mainChannel = s.Channel(mainChannelName)

	// Set the event functions.
	bs.OnClose(s.onClose)

	// Stop the timeout again. It will be started by the ping timer.
	s.pingTimeout.Stop()

	// Start the loops and handlers in new goroutines.
	go s.pingTimeoutHandler()
	go s.readLoop()
	go s.pingLoop()

	return s
}

// ID returns the socket's unique ID.
func (s *Socket) ID() string {
	return s.id
}

// RemoteAddr returns the remote address of the client.
func (s *Socket) RemoteAddr() string {
	return s.bs.RemoteAddr()
}

// UserAgent returns the user agent of the client.
func (s *Socket) UserAgent() string {
	return s.bs.UserAgent()
}

// Close the socket connection.
func (s *Socket) Close() {
	s.bs.Close()
}

// IsClosed returns a boolean whenever the connection is closed.
func (s *Socket) IsClosed() bool {
	return s.bs.IsClosed()
}

// OnClose sets the functions which is triggered if the socket connection is closed.
func (s *Socket) OnClose(f OnCloseFunc) {
	s.onCloseFunc = f
}

// Write data to the client.
func (s *Socket) Write(data string) {
	// Write to the main channel.
	s.mainChannel.Write(data)
}

// Read the next message from the socket. This method is blocking.
// One variadic argument sets a timeout duration.
// If no timeout is specified, this method will block forever.
// ErrSocketClosed is returned, if the socket connection is closed.
// ErrReadTimeout is returned, if the timeout is reached.
func (s *Socket) Read(timeout ...time.Duration) (string, error) {
	return s.mainChannel.Read(timeout...)
}

// OnRead sets the function which is triggered if new data is received.
// If this event function based method of reading data from the socket is used,
// then don't use the socket Read method.
// Either use the OnRead or the Read approach.
func (s *Socket) OnRead(f OnReadFunc) {
	s.mainChannel.OnRead(f)
}

// DiscardRead ignores and discars the data received from the client.
// Call this method during initialization, if you don't read any data from
// the socket. If received data is not discarded, then the read buffer will block as soon
// as it is full, which will also block the keep-alive mechanism of the socket. The result
// would be a closed socket...
func (s *Socket) DiscardRead() {
	s.mainChannel.DiscardRead()
}

//##############################//
//### Private Socket methods ###//
//##############################//

func (s *Socket) write(rawData string) {
	// Write to the stream and check if the buffer is full.
	select {
	case <-s.isClosedChan:
		// Just return because the socket is closed.
		return
	case s.writeChan <- rawData:
	default:
		// The buffer if full. No data was send.
		// Send a ping. If no pong is received within
		// the timeout, the socket is closed.
		s.sendPing()

		// Now write the current data to the socket.
		// This will block if the buffer is still full.
		s.writeChan <- rawData
	}
}

func (s *Socket) onClose() {
	// Stop all goroutines for this socket by closing the isClosed channel.
	close(s.isClosedChan)

	// Clear the write channel to release blocked goroutines.
	// The pingLoop might be blocked...
	for i := 0; i < len(s.writeChan); i++ {
		select {
		case <-s.writeChan:
		default:
			break
		}
	}

	// Remove the socket again from the active sockets map.
	func() {
		// Lock the mutex.
		socketsMutex.Lock()
		defer socketsMutex.Unlock()

		delete(sockets, s.id)
	}()

	// Trigger the on close event if defined.
	if s.onCloseFunc != nil {
		func() {
			// Recover panics and log the error.
			defer func() {
				if e := recover(); e != nil {
					log.L.Errorf("glue: panic while calling onClose function: %v\n%s", e, debug.Stack())
				}
			}()

			s.onCloseFunc()
		}()
	}
}

func (s *Socket) resetPingTimeout() {
	// Lock the mutex.
	s.sendPingMutex.Lock()
	defer s.sendPingMutex.Unlock()

	// Stop the timeout timer.
	s.pingTimeout.Stop()

	// Update the flag.
	s.pingRequestActive = false

	// Reset the ping timer again to request
	// a pong repsonse during the next timeout.
	s.pingTimer.Reset(pingPeriod)
}

// SendPing sends a ping to the client. If no pong response is
// received within the timeout, the socket will be closed.
// Multiple calls to this method won't send multiple ping requests,
// if a ping request is already active.
func (s *Socket) sendPing() {
	// Lock the mutex.
	s.sendPingMutex.Lock()

	// Check if a ping request is already active.
	if s.pingRequestActive {
		// Unlock the mutex again.
		s.sendPingMutex.Unlock()
		return
	}

	// Update the flag and unlock the mutex again.
	s.pingRequestActive = true
	s.sendPingMutex.Unlock()

	// Start the timeout timer. This will close
	// the socket if no pong response is received
	// within the timeout.
	// Do this before the write. The write channel might block
	// if the buffers are full.
	s.pingTimeout.Reset(pingResponseTimeout)

	// Send a ping request by writing to the stream.
	s.writeChan <- cmdPing
}

// Close the socket during a ping response timeout.
func (s *Socket) pingTimeoutHandler() {
	defer func() {
		s.pingTimeout.Stop()
	}()

	select {
	case <-s.pingTimeout.C:
		// Close the socket due to the timeout.
		s.bs.Close()
	case <-s.isClosedChan:
		// Just release this goroutine.
	}
}

func (s *Socket) pingLoop() {
	defer func() {
		// Stop the timeout timer.
		s.pingTimeout.Stop()

		// Stop the ping timer.
		s.pingTimer.Stop()
	}()

	for {
		select {
		case <-s.pingTimer.C:
			// Send a ping. If no pong is received within
			// the timeout, the socket is closed.
			s.sendPing()

		case <-s.isClosedChan:
			// Just exit the loop.
			return
		}
	}
}

func (s *Socket) readLoop() {
	// Wait for data received from the read channel.
	for {
		select {
		case data := <-s.readChan:
			// Reset the ping timeout.
			s.resetPingTimeout()

			// Get the command. The command is always prepended to the data message.
			cmd := data[:cmdLen]
			data = data[cmdLen:]

			// Handle the received data and log error messages.
			if err := s.handleRead(cmd, data); err != nil {
				log.L.WithFields(logrus.Fields{
					"remoteAddress": s.RemoteAddr(),
					"userAgent":     s.UserAgent(),
					"cmd":           cmd,
				}).Warningf("glue: handle received data: %v", err)
			}
		case <-s.isClosedChan:
			// Just exit the loop
			return
		}
	}
}

func (s *Socket) handleRead(cmd, data string) error {
	// Perform the command request.
	switch cmd {
	case cmdPing:
		// Send a pong reply.
		s.write(cmdPong)

	case cmdPong:
		// Don't do anything, The ping timer was already reset.

	case cmdClose:
		// Close the socket.
		s.bs.Close()

	case cmdInit:
		// Handle the initialization.
		initSocket(s, data)

	case cmdChannelData:
		// Unmarshal the channel name and data string.
		name, data, err := utils.UnmarshalValues(data)
		if err != nil {
			return err
		}

		// Push the data to the corresponding channel.
		if err = s.channels.triggerReadForChannel(name, data); err != nil {
			return err
		}
	default:
		// Send an invalid command response.
		s.write(cmdInvalid)

		// Return an error.
		return fmt.Errorf("received invalid socket command")
	}

	return nil
}

//##############//
//### Public ###//
//##############//

// Block new incomming connections.
func Block(b bool) {
	block = b
}

// OnNewSocket sets the event function which is
// triggered if a new socket connection was made.
// The event function must not block! As soon as the event function
// returns, the socket is added to the active sockets map.
func OnNewSocket(f OnNewSocketFunc) {
	onNewSocket = f
}

// Sockets returns a list of all current connected sockets.
// Hint: Sockets are added to the active sockets list after the OnNewSocket
// event function is called.
func Sockets() []*Socket {
	// Lock the mutex.
	socketsMutex.Lock()
	defer socketsMutex.Unlock()

	// Create the slice.
	list := make([]*Socket, len(sockets))

	// Add all sockets from the map.
	i := 0
	for _, s := range sockets {
		list[i] = s
		i++
	}

	return list
}

// Release this package. This will block all new incomming socket connections
// and close all current connected sockets.
func Release() {
	// Block all new incomming socket connections.
	Block(true)

	// Wait for a little moment, so new incomming sockets are added
	// to the sockets active list.
	time.Sleep(200 * time.Millisecond)

	// Close all current connected sockets.
	sockets := Sockets()
	for _, s := range sockets {
		s.Close()
	}
}

//###############//
//### Private ###//
//###############//

func init() {
	// Set the event function.
	backend.OnNewSocketConnection(onNewSocketConnection)
}

func onNewSocketConnection(bs backend.BackendSocket) {
	// Close the socket if incomming connections should be blocked.
	if block {
		bs.Close()
		return
	}

	// Create a new socket value.
	// The goroutines are started automatically.
	newSocket(bs)
}

func initSocket(s *Socket, dataJSON string) {
	// Handle the socket initialization in an anonymous function
	// to handle the error in a clean and simple way.
	err := func() error {
		// Handle received initialization data:
		// ####################################

		// Unmarshal the data JSON.
		var cData clientInitData
		err := json.Unmarshal([]byte(dataJSON), &cData)
		if err != nil {
			return fmt.Errorf("json unmarshal init data: %v", err)
		}

		// The client and server protocol versions have to match.
		if cData.Version != Version {
			return fmt.Errorf("client and server socket protocol version does not match: %s", cData.Version)
		}

		// Send initialization data:
		// #########################

		// Create the new initialization data value.
		data := initData{}

		// Marshal the data to a JSON string.
		dataJSON, err := json.Marshal(&data)
		if err != nil {
			return fmt.Errorf("json marshal init data: %v", err)
		}

		// Send the init data to the client.
		s.write(cmdInit + string(dataJSON))

		return nil
	}()

	// Handle the error.
	if err != nil {
		// Close the socket.
		s.Close()

		// Log the error.
		log.L.WithFields(logrus.Fields{
			"remoteAddress": s.RemoteAddr(),
			"userAgent":     s.UserAgent(),
		}).Warningf("glue: init socket: %v", err)

		return
	}

	// Trigger the on new socket event function.
	func() {
		// Recover panics and log the error.
		defer func() {
			if e := recover(); e != nil {
				// Close the socket and log the error message.
				s.Close()
				log.L.Errorf("glue: panic while calling on new socket function: %v\n%s", e, debug.Stack())
			}
		}()

		// Trigger the event function.
		onNewSocket(s)
	}()

	// Just return if the socket was closed by the triggered event function.
	if s.IsClosed() {
		return
	}

	// Add the new socket to the active sockets map.
	func() {
		// Lock the mutex.
		socketsMutex.Lock()
		defer socketsMutex.Unlock()

		// Add the socket to the map.
		sockets[s.id] = s
	}()
}
