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
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/desertbit/glue/log"
	"github.com/desertbit/glue/utils"
)

//#################//
//### Constants ###//
//#################//

const (
	// The channel buffer size for received data.
	readChanBuffer = 7
)

//####################//
//### Channel type ###//
//####################//

// A Channel is a separate communication channel.
type Channel struct {
	s           *Socket
	readHandler *handler

	name     string
	readChan chan string
}

func newChannel(s *Socket, name string) *Channel {
	return &Channel{
		s:           s,
		readHandler: newHandler(),
		name:        name,
		readChan:    make(chan string, readChanBuffer),
	}
}

// Socket returns the channel's socket.
func (c *Channel) Socket() *Socket {
	return c.s
}

// Write data to the channel.
func (c *Channel) Write(data string) {
	// Prepend the socket command and send the channel name and data.
	c.s.write(cmdChannelData + utils.MarshalValues(c.name, data))
}

// Read the next message from the channel. This method is blocking.
// One variadic argument sets a timeout duration.
// If no timeout is specified, this method will block forever.
// ErrSocketClosed is returned, if the socket connection is closed.
// ErrReadTimeout is returned, if the timeout is reached.
func (c *Channel) Read(timeout ...time.Duration) (string, error) {
	timeoutChan := make(chan (struct{}))

	// Create a timeout timer if a timeout is specified.
	if len(timeout) > 0 && timeout[0] > 0 {
		timer := time.AfterFunc(timeout[0], func() {
			// Trigger the timeout by closing the channel.
			close(timeoutChan)
		})

		// Always stop the timer on defer.
		defer timer.Stop()
	}

	select {
	case data := <-c.readChan:
		return data, nil
	case <-c.s.isClosedChan:
		// The connection was closed.
		// Return an error.
		return "", ErrSocketClosed
	case <-timeoutChan:
		// The timeout was reached.
		// Return an error.
		return "", ErrReadTimeout
	}
}

// OnRead sets the function which is triggered if new data is received on the channel.
// If this event function based method of reading data from the socket is used,
// then don't use the socket Read method.
// Either use the OnRead or the Read approach.
func (c *Channel) OnRead(f OnReadFunc) {
	// Create a new read handler for this channel.
	// Previous handlers are stopped first.
	handlerStopped := c.readHandler.New()

	// Start the handler goroutine.
	go func() {
		for {
			select {
			case data := <-c.readChan:
				func() {
					// Recover panics and log the error.
					defer func() {
						if e := recover(); e != nil {
							log.L.Errorf("glue: panic while calling onRead function: %v\n%s", e, debug.Stack())
						}
					}()

					// Trigger the on read event function.
					f(data)
				}()
			case <-c.s.isClosedChan:
				// Release this goroutine if the socket is closed.
				return
			case <-handlerStopped:
				// Release this goroutine.
				return
			}
		}
	}()
}

// DiscardRead ignores and discars the data received from this channel.
// Call this method during initialization, if you don't read any data from
// this channel. If received data is not discarded, then the read buffer will block as soon
// as it is full, which will also block the keep-alive mechanism of the socket. The result
// would be a closed socket...
func (c *Channel) DiscardRead() {
	// Create a new read handler for this channel.
	// Previous handlers are stopped first.
	handlerStopped := c.readHandler.New()

	// Start the handler goroutine.
	go func() {
		for {
			select {
			case <-c.readChan:
				// Don't do anything.
				// Just discard the data.
			case <-c.s.isClosedChan:
				// Release this goroutine if the socket is closed.
				return
			case <-handlerStopped:
				// Release this goroutine.
				return
			}
		}
	}()
}

func (c *Channel) triggerRead(data string) {
	// Send the data to the read channel.
	c.readChan <- data
}

//#####################//
//### Channels type ###//
//#####################//

type channels struct {
	m     map[string]*Channel
	mutex sync.Mutex
}

func newChannels() *channels {
	return &channels{
		m: make(map[string]*Channel),
	}
}

func (cs *channels) get(name string) *Channel {
	// Lock the mutex.
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	return cs.m[name]
}

func (cs *channels) triggerReadForChannel(name, data string) error {
	// Get the channel.
	c := cs.get(name)
	if c == nil {
		return fmt.Errorf("received data for channel '%s': channel does not exists", name)
	}

	// Trigger the read.
	c.triggerRead(data)

	return nil
}

//#################################//
//### Additional Socket Methods ###//
//#################################//

// Channel returns the corresponding channel value specified by the name.
// If no channel value exists for the given name, a new channel is created.
// Multiple calls to Channel with the same name, will always return the same
// channel value pointer.
func (s *Socket) Channel(name string) *Channel {
	// Get the socket channel pointer.
	cs := s.channels

	// Lock the mutex.
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	// Get the channel if it exists.
	c, ok := cs.m[name]
	if ok {
		return c
	}

	// Create and add the new channel to the socket channels map.
	c = newChannel(s, name)
	cs.m[name] = c

	return c
}
