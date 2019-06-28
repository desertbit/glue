/*
 *  Glue - Robust Go and Javascript Socket Library
 *  Copyright (C) 2015  Roland Singer <roland.singer[at]desertbit.com>
 * 
 *  SPDX-License-Identifier: MIT
 */

package main

import (
	"log"
	"net/http"

	"github.com/desertbit/glue"
)

func main() {
	// Set the http file server.
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("public"))))
	http.Handle("/dist/", http.StripPrefix("/dist/", http.FileServer(http.Dir("../../client/dist"))))

	// Create a new glue server.
	server := glue.NewServer(glue.Options{
		HTTPListenAddress: ":8080",
	})

	// Release the glue server on defer.
	// This will block new incoming connections
	// and close all current active sockets.
	defer server.Release()

	// Set the glue event function to handle new incoming socket connections.
	server.OnNewSocket(onNewSocket)

	// Run the glue server.
	err := server.Run()
	if err != nil {
		log.Fatalf("Glue Run: %v", err)
	}
}

func onNewSocket(s *glue.Socket) {
	// We won't read any data from the socket itself.
	// Discard received data!
	s.DiscardRead()

	// Set a function which is triggered as soon as the socket is closed.
	s.OnClose(func() {
		log.Printf("socket closed with remote address: %s", s.RemoteAddr())
	})

	// Create a channel.
	c := s.Channel("golang")

	// Set the channel on read event function.
	c.OnRead(func(data string) {
		// Echo the received data back to the client.
		c.Write("channel golang: " + data)
	})

	// Write to the channel.
	c.Write("Hello Gophers!")
}
