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
