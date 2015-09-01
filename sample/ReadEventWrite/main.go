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

	// Set a function which is triggered during each received message.
	s.OnRead(func(data string) {
		// Echo the received data back to the client.
		s.Write(data)
	})

	// Send a welcome string to the client.
	s.Write("Hello Client")
}
