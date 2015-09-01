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

// Emit a close function only once, also if called multiple times.
// This implementation is thread-safe.
package closer

import (
	"sync"
)

type Closer struct {
	// Channel which is closed if the closer is closed.
	IsClosedChan chan struct{}

	f        func()
	isClosed bool
	mutex    sync.Mutex
}

// New creates a new closer.
// The passed function is emitted only once, as soon close is called.
func New(f func()) *Closer {
	return &Closer{
		IsClosedChan: make(chan struct{}),
		f:            f,
	}
}

// Close calls the function and sets the IsClosed boolean.
func (c *Closer) Close() {
	// Lock the mutex
	c.mutex.Lock()

	// Just return if already closed.
	if c.isClosed {
		// Unlock the mutex again
		c.mutex.Unlock()
		return
	}

	// Update the flag
	c.isClosed = true

	// Unlock the mutex again
	c.mutex.Unlock()

	// Close the channel.
	close(c.IsClosedChan)

	// Emit the function.
	c.f()
}

// IsClosed returns a boolean whenever this closer is already closed.
func (c *Closer) IsClosed() bool {
	return c.isClosed
}
