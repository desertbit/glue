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

// Package glue - Robust Go and Javascript Socket Library.
// This library is thread-safe.
package glue

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/desertbit/glue/backend"
)

//####################//
//### Public Types ###//
//####################//

// OnNewSocketFunc is an event function.
type OnNewSocketFunc func(s *Socket)

//###################//
//### Server Type ###//
//###################//

// A Server represents a glue server which handles incoming socket connections.
type Server struct {
	bs      *backend.Server
	options *Options

	block       bool
	onNewSocket OnNewSocketFunc

	sockets      map[string]*Socket // A map holding all active current sockets.
	socketsMutex sync.Mutex
}

// NewServer creates a new glue server instance.
// One variadic arguments specifies the server options.
func NewServer(o ...Options) *Server {
	// Get or create the options.
	var options *Options
	if len(o) > 0 {
		options = &o[0]
	} else {
		options = &Options{}
	}

	// Set the default option values for unset values.
	options.SetDefaults()

	// Create a new backend server.
	bs := backend.NewServer(len(options.HTTPHandleURL), options.EnableCORS, options.CheckOrigin)

	// Create a new server value.
	s := &Server{
		bs:          bs,
		options:     options,
		onNewSocket: func(*Socket) {}, // Initialize with dummy function to remove nil check.
		sockets:     make(map[string]*Socket),
	}

	// Set the backend server event function.
	bs.OnNewSocketConnection(s.handleOnNewSocketConnection)

	return s
}

// Block new incomming connections.
func (s *Server) Block(b bool) {
	s.block = b
}

// OnNewSocket sets the event function which is
// triggered if a new socket connection was made.
// The event function must not block! As soon as the event function
// returns, the socket is added to the active sockets map.
func (s *Server) OnNewSocket(f OnNewSocketFunc) {
	s.onNewSocket = f
}

// GetSocket obtains a socket by its ID.
// Returns nil if not found.
func (s *Server) GetSocket(id string) *Socket {
	// Lock the mutex.
	s.socketsMutex.Lock()
	defer s.socketsMutex.Unlock()

	// Obtain the socket.
	socket, ok := s.sockets[id]
	if !ok {
		return nil
	}

	return socket
}

// Sockets returns a list of all current connected sockets.
// Hint: Sockets are added to the active sockets list before the OnNewSocket
// event function is called.
// Use the IsInitialized flag to determind if a socket is not ready yet...
func (s *Server) Sockets() []*Socket {
	// Lock the mutex.
	s.socketsMutex.Lock()
	defer s.socketsMutex.Unlock()

	// Create the slice.
	list := make([]*Socket, len(s.sockets))

	// Add all sockets from the map.
	i := 0
	for _, s := range s.sockets {
		list[i] = s
		i++
	}

	return list
}

// Release this package. This will block all new incomming socket connections
// and close all current connected sockets.
func (s *Server) Release() {
	// Block all new incomming socket connections.
	s.Block(true)

	// Wait for a little moment, so new incomming sockets are added
	// to the sockets active list.
	time.Sleep(200 * time.Millisecond)

	// Close all current connected sockets.
	sockets := s.Sockets()
	for _, s := range sockets {
		s.Close()
	}
}

// Run starts the server and listens for incoming socket connections.
// This is a blocking method.
func (s *Server) Run() error {
	// Skip if set to none.
	if s.options.HTTPSocketType != HTTPSocketTypeNone {
		// Set the base glue HTTP handler.
		http.Handle(s.options.HTTPHandleURL, s)

		// Start the http server.
		if s.options.HTTPSocketType == HTTPSocketTypeUnix {
			// Listen on the unix socket.
			l, err := net.Listen("unix", s.options.HTTPListenAddress)
			if err != nil {
				return fmt.Errorf("Listen: %v", err)
			}

			// Start the http server.
			err = http.Serve(l, nil)
			if err != nil {
				return fmt.Errorf("Serve: %v", err)
			}
		} else if s.options.HTTPSocketType == HTTPSocketTypeTCP {
			// Start the http server.
			err := http.ListenAndServe(s.options.HTTPListenAddress, nil)
			if err != nil {
				return fmt.Errorf("ListenAndServe: %v", err)
			}
		} else {
			return fmt.Errorf("invalid socket options type: %v", s.options.HTTPSocketType)
		}
	} else {
		// HINT: This is only a placeholder until the internal glue TCP server is implemented.
		w := make(chan struct{})
		<-w
	}

	return nil
}

// ServeHTTP implements the HTTP Handler interface of the http package.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.bs.ServeHTTP(w, r)
}

//########################//
//### Server - Private ###//
//########################//

func (s *Server) handleOnNewSocketConnection(bs backend.BackendSocket) {
	// Close the socket if incomming connections should be blocked.
	if s.block {
		bs.Close()
		return
	}

	// Create a new socket value.
	// The goroutines are started automatically.
	newSocket(s, bs)
}
