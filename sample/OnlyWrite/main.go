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

package main

import (
	"log"
	"net/http"
	"runtime"

	"github.com/desertbit/glue"
)

const (
	ListenAddress = ":8888"
)

func main() {
	// Set the maximum number of CPUs that can be executing simultaneously.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Release the glue library on defer.
	// This will block new incoming connections
	// and close all current active sockets.
	defer glue.Release()

	// Set the glue event function.
	glue.OnNewSocket(onNewSocket)

	// Set the http file server.
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("public"))))
	http.Handle("/dist/", http.StripPrefix("/dist/", http.FileServer(http.Dir("../../client/dist"))))

	// Start the http server.
	err := http.ListenAndServe(ListenAddress, nil)
	if err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}

func onNewSocket(s *glue.Socket) {
	// Set a function which is triggered as soon as the socket is closed.
	s.OnClose(func() {
		log.Printf("socket closed with remote address: %s", s.RemoteAddr())
	})

	// Discard all reads.
	// If received data is not discarded, then the read buffer will block as soon
	// as it is full, which will also block the keep-alive mechanism of the socket.
	// The result would be a closed socket...
	s.DiscardRead()

	// Send a welcome string to the client.
	s.Write("Hello Client")
}
