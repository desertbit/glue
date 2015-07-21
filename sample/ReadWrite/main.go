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

	// Run the read loop in a new goroutine.
	go readLoop(s)

	// Send a welcome string to the client.
	s.Write("Hello Client")
}

func readLoop(s *glue.Socket) {
	for {
		// Wait for available data.
		// Optional: pass a timeout duration to read.
		data, err := s.Read()
		if err != nil {
			// Just return and release this goroutine if the socket was closed.
			if err == glue.ErrSocketClosed {
				return
			}

			log.Printf("read error: %v", err)
			continue
		}

		// Echo the received data back to the client.
		s.Write(data)
	}
}
