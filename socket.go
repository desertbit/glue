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
	"time"

	"github.com/desertbit/glue/backend"
)

const (
	// Send pings to the peer with this period.
	pingPeriod = 30 * time.Second

	// Stream buffer sizes.
	readStreamBufferSize  = 3
	writeStreamBufferSize = 5

	// Socket commands. Only one character.
	cmdLen     = 1
	cmdPing    = "i"
	cmdPong    = "o"
	cmdData    = "d"
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

	onRead      OnReadFunc
	onCloseFunc OnCloseFunc

	pingCount int
	pingTimer *time.Timer

	readStream    chan string
	writeStream   chan string
	stopReadLoop  chan struct{}
	stopWriteLoop chan struct{}
}

// newSocket creates a new socket and initializes it.
func newSocket(bs backend.BackendSocket) *Socket {
	// Create a new socket value.
	s := &Socket{
		bs: bs,

		// Set a dummy function to not always
		// check if the method is not set.
		onRead: func(string) {},

		pingCount: 0,
		pingTimer: time.NewTimer(pingPeriod),

		readStream:    make(chan string, readStreamBufferSize),
		writeStream:   make(chan string, writeStreamBufferSize),
		stopReadLoop:  make(chan struct{}),
		stopWriteLoop: make(chan struct{}),
	}

	// Set the event functions.
	bs.OnClose(s.onClose)

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
	// Write to the stream.
	s.writeStream <- rawData
}

func (s *Socket) onClose() {
	// Stop the write and read loop by triggering the quit triggers.
	close(s.stopReadLoop)
	close(s.stopWriteLoop)

	// Trigger the on close event if defined.
	if s.onCloseFunc != nil {
		s.onCloseFunc()
	}
}

func (s *Socket) writeLoop() {
	defer func() {
		// Stop the ping timer.
		s.pingTimer.Stop()
	}()

	for {
		select {
		case data := <-s.writeStream:
			// Send the data to the client.
			s.bs.Write(data)

		case <-s.pingTimer.C:
			// Check if the client didn't respond since the last ping request.
			if s.pingCount >= 1 {
				// Close the socket.
				s.bs.Close()
				return
			}

			// Increment the ping count.
			s.pingCount += 1

			// Send a ping request.
			s.bs.Write(cmdPing)

			// Reset the timer again
			s.pingTimer.Reset(pingPeriod)

		case <-s.stopWriteLoop:
			// Just exit the loop
			return
		}
	}
}

func (s *Socket) readLoop() {
	//  Our on read function.
	onRead := func(data string) {
		// Write to the read stream.
		s.readStream <- data
	}

	// Set the on read function to the backend socket.
	s.bs.OnRead(onRead)

	// Wait for data received from the read stream.
	for {
		select {
		case data := <-s.readStream:
			// Reset the timer again
			s.pingTimer.Reset(pingPeriod)

			// Reset the ping count
			s.pingCount = 0

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
			case cmdData:
				// Trigger the on read event function.
				s.onRead(data)
			default:
				// Send an invalid command response.
				s.write(cmdInvalid)
			}
		case <-s.stopReadLoop:
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

	// Start the loops in new goroutines.
	go s.readLoop()
	go s.writeLoop()

	// Trigger the on new socket event function.
	onNewSocket(s)
}
