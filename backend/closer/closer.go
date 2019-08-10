/*
 *  Glue - Robust Go and Javascript Socket Library
 *  Copyright (C) 2015  Roland Singer <roland.singer[at]desertbit.com>
 * 
 *  SPDX-License-Identifier: MIT
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

	f     func()
	mutex sync.Mutex
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
	defer c.mutex.Unlock()

	// Just return if already closed.
	if c.IsClosed() {
		return
	}

	// Close the channel.
	close(c.IsClosedChan)

	// Emit the function.
	c.f()
}

// IsClosed returns a boolean whenever this closer is already closed.
func (c *Closer) IsClosed() bool {
	select {
	case <-c.IsClosedChan:
		return true
	default:
		return false
	}
}
