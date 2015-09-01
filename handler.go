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
