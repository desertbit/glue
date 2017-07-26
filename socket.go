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

package glue

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/blang/semver"
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
	// This project follows the Semantic Versioning (http://semver.org/).
	Version = "1.9.1"
)

// Private
// #######
const (
	// The constant length of the random socket ID.
	socketIDLength = 20

	// Send pings to the peer with this period.
	pingPeriod = 30 * time.Second

	// Kill the socket after this timeout.
	pingResponseTimeout = 7 * time.Second

	// The main channel name.
	mainChannelName = "m"

	// Socket commands. Must be two character long.
	// ############################################
	cmdLen               = 2
	cmdInit              = "in"
	cmdPing              = "pi"
	cmdPong              = "po"
	cmdClose             = "cl"
	cmdInvalid           = "iv"
	cmdDontAutoReconnect = "dr"
	cmdChannelData       = "cd"
)

//#################//
//### Variables ###//
//#################//

// Public errors:
var (
	ErrSocketClosed = errors.New("the socket connection is closed")
	ErrReadTimeout  = errors.New("the read timeout was reached")
)

// Private
var (
	serverVersion semver.Version
)

//####################//
//### Public Types ###//
//####################//

// ClosedChan is a channel which doesn't block as soon as the socket is closed.
type ClosedChan <-chan struct{}

// OnCloseFunc is an event function.
type OnCloseFunc func()

// OnReadFunc is an event function.
type OnReadFunc func(data string)

//#####################//
//### Private Types ###//
//#####################//

type initData struct {
	SocketID string `json:"socketID"`
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
	server *Server
	bs     backend.BackendSocket

	id            string // Unique socket ID.
	isInitialized bool

	channels    *channels
	mainChannel *Channel

	writeChan    chan string
	readChan     chan string
	isClosedChan ClosedChan

	pingTimer         *time.Timer
	pingTimeout       *time.Timer
	sendPingMutex     sync.Mutex
	pingRequestActive bool
}

// newSocket creates a new socket and initializes it.
func newSocket(server *Server, bs backend.BackendSocket) *Socket {
	// Create a new socket value.
	s := &Socket{
		server: server,
		bs:     bs,

		id:       utils.RandomString(socketIDLength),
		channels: newChannels(),

		writeChan:    bs.WriteChan(),
		readChan:     bs.ReadChan(),
		isClosedChan: bs.ClosedChan(),

		pingTimer:   time.NewTimer(pingPeriod),
		pingTimeout: time.NewTimer(pingResponseTimeout),
	}

	// Create the main channel.
	s.mainChannel = s.Channel(mainChannelName)

	// Call the on close method as soon as the socket closes.
	go func() {
		<-s.isClosedChan
		s.onClose()
	}()

	// Stop the timeout again. It will be started by the ping timer.
	s.pingTimeout.Stop()

	// Add the new socket to the active sockets map.
	// If the ID is already present, then generate a new one.
	func() {
		// Lock the mutex.
		s.server.socketsMutex.Lock()
		defer s.server.socketsMutex.Unlock()

		// Be sure that the ID is unique.
		for {
			if _, ok := s.server.sockets[s.id]; !ok {
				break
			}

			s.id = utils.RandomString(socketIDLength)
		}

		// Add the socket to the map.
		s.server.sockets[s.id] = s
	}()

	// Start the loops and handlers in new goroutines.
	go s.pingTimeoutHandler()
	go s.readLoop()
	go s.pingLoop()

	return s
}

// ID returns the socket's unique ID.
// This is a cryptographically secure pseudorandom number.
func (s *Socket) ID() string {
	return s.id
}

// IsInitialized returns a boolean indicating if a socket is initialized
// and ready to be used. This flag is set to true after the OnNewSocket function
// has returned for this socket.
func (s *Socket) IsInitialized() bool {
	return s.isInitialized
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
// This method can be called multiple times to bind multiple functions.
func (s *Socket) OnClose(f OnCloseFunc) {
	go func() {
		// Recover panics and log the error.
		defer func() {
			if e := recover(); e != nil {
				log.L.Errorf("glue: panic while calling onClose function: %v\n%s", e, debug.Stack())
			}
		}()

		<-s.isClosedChan
		f()
	}()
}

// ClosedChan returns a channel which is non-blocking (closed)
// as soon as the socket is closed.
func (s *Socket) ClosedChan() ClosedChan {
	return s.isClosedChan
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
	// Remove the socket again from the active sockets map.
	func() {
		// Lock the mutex.
		s.server.socketsMutex.Lock()
		defer s.server.socketsMutex.Unlock()

		delete(s.server.sockets, s.id)
	}()

	// Clear the write channel to release blocked goroutines.
	// The pingLoop might be blocked...
	for i := 0; i < len(s.writeChan); i++ {
		select {
		case <-s.writeChan:
		default:
			break
		}
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

//###############//
//### Private ###//
//###############//

func init() {
	var err error

	// Parses the server version string and returns a validated Version.
	serverVersion, err = semver.Make(Version)
	if err != nil {
		log.L.Fatalf("failed to parse glue server protocol version: %v", err)
	}
}

func initSocket(s *Socket, dataJSON string) {
	// Handle the socket initialization in an anonymous function
	// to handle the error in a clean and simple way.
	dontAutoReconnect, err := func() (bool, error) {
		// Handle received initialization data:
		// ####################################

		// Unmarshal the data JSON.
		var cData clientInitData
		err := json.Unmarshal([]byte(dataJSON), &cData)
		if err != nil {
			return false, fmt.Errorf("json unmarshal init data: %v", err)
		}

		// Parses the client version string and returns a validated Version.
		clientVersion, err := semver.Make(cData.Version)
		if err != nil {
			return false, fmt.Errorf("invalid client protocol version: %v", err)
		}

		// Check if the client protocol version is supported.
		if clientVersion.Major != serverVersion.Major ||
			clientVersion.Minor > serverVersion.Minor ||
			(clientVersion.Minor == serverVersion.Minor && clientVersion.Patch > serverVersion.Patch) {
			// The client should not automatically reconnect. Return true...
			return true, fmt.Errorf("client socket protocol version is not supported: %s", cData.Version)
		}

		// Send initialization data:
		// #########################

		// Create the new initialization data value.
		data := initData{
			SocketID: s.ID(),
		}

		// Marshal the data to a JSON string.
		dataJSON, err := json.Marshal(&data)
		if err != nil {
			return false, fmt.Errorf("json marshal init data: %v", err)
		}

		// Send the init data to the client.
		s.write(cmdInit + string(dataJSON))

		return false, nil
	}()

	// Handle the error.
	if err != nil {
		if dontAutoReconnect {
			// Tell the client to not automatically reconnect.
			s.write(cmdDontAutoReconnect)

			// Pause to be sure that the previous socket command gets send to the client.
			time.Sleep(time.Second)
		}

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
		s.server.onNewSocket(s)
	}()

	// Update the initialized flag.
	s.isInitialized = true
}
