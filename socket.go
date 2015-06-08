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

// Robust Go and Javascript Socket Library.
// This library is thread-safe.
package glue

import (
	"sync"
	"time"

	"github.com/desertbit/glue/backend"
)

const (
	// Send pings to the peer with this period.
	pingPeriod = 30 * time.Second
	// Kill the socket after this timeout.
	pingResponseTimeout = 7 * time.Second

	// Socket commands. Only one character.
	cmdLen     = 1
	cmdPing    = "i"
	cmdPong    = "o"
	cmdData    = "d"
	cmdClose   = "c"
	cmdInvalid = "z"
)

var (
	block       bool
	onNewSocket = func(*Socket) {} // Initialize with dummy function to remove nil check.
)

//###################//
//### Extra Types ###//
//###################//

type OnNewSocketFunc func(s *Socket)
type OnCloseFunc func()
type OnReadFunc func(data string)

//###################//
//### Socket Type ###//
//###################//

type Socket struct {
	bs backend.BackendSocket

	writeChan chan string
	readChan  chan string

	onRead      OnReadFunc
	onCloseFunc OnCloseFunc

	pingTimer         *time.Timer
	pingTimeout       *time.Timer
	sendPingMutex     sync.Mutex
	pingRequestActive bool

	isClosedChan chan struct{}
}

// newSocket creates a new socket and initializes it.
func newSocket(bs backend.BackendSocket) *Socket {
	// Create a new socket value.
	s := &Socket{
		bs:        bs,
		writeChan: bs.WriteChan(),
		readChan:  bs.ReadChan(),

		// Set a dummy function to not always
		// check if the method is not set.
		onRead: func(string) {},

		pingTimer:   time.NewTimer(pingPeriod),
		pingTimeout: time.NewTimer(pingResponseTimeout),

		isClosedChan: make(chan struct{}),
	}

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
	// Prepend the data command and write to the client.
	s.write(cmdData + data)
}

// OnRead sets the function which is triggered if new data is received.
func (s *Socket) OnRead(f OnReadFunc) {
	// Set the on read function.
	s.onRead = f
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
	// Stop all goroutines for this socket by closing the is closed channel.
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

	// Trigger the on close event if defined.
	if s.onCloseFunc != nil {
		s.onCloseFunc()
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
			case cmdData:
				// Trigger the on read event function.
				s.onRead(data)
			default:
				// Send an invalid command response.
				s.write(cmdInvalid)
			}
		case <-s.isClosedChan:
			// Just exit the loop
			return
		}
	}
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
func OnNewSocket(f OnNewSocketFunc) {
	onNewSocket = f
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
	s := newSocket(bs)

	// Trigger the on new socket event function.
	onNewSocket(s)
}
