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
