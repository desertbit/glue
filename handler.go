/*
 *  Glue - Robust Go and Javascript Socket Library
 *  Copyright (C) 2015  Roland Singer <roland.singer[at]desertbit.com>
 * 
 *  SPDX-License-Identifier: MIT
 */

package glue

import "sync"

//###############//
//### Handler ###//
//###############//

type handler struct {
	stopChan       chan struct{}
	stopChanClosed bool

	mutex sync.Mutex
}

func newHandler() *handler {
	return &handler{
		stopChanClosed: true,
	}
}

// New creates a new handler and stopps the previous handler if present.
// A stop channel is returned, which is closed as soon as the handler is stopped.
func (h *handler) New() chan struct{} {
	// Lock the mutex.
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Signal the stop request by closing the channel if open.
	if !h.stopChanClosed {
		close(h.stopChan)
	}

	// Create a new stop channel.
	h.stopChan = make(chan struct{})

	// Update the flag.
	h.stopChanClosed = false

	return h.stopChan
}

// Stop the handler if present.
func (h *handler) Stop() {
	// Lock the mutex.
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Signal the stop request by closing the channel if open.
	if !h.stopChanClosed {
		close(h.stopChan)
		h.stopChanClosed = true
	}
}
