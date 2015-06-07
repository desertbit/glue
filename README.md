# Glue - Robust Go and Javascript Socket Library

Glue is a real-time bidirectional socket library. It is a **clean**, **robust** and **effecient** alternative to socket.io. This library is designed to connect webbrowsers with a go-backend in a simple way. It automatically detects supported socket layers and chooses the most suitable one.


## Socket layers

Currently two socket layers are supported:

- **WebSockets** - This is the primary option. They are used if the webbrowser supports WebSockets defined after [RFC 6455](https://tools.ietf.org/html/rfc6455).
- **AjaxSockets** - This socket layer is used as a fallback mode.


## Support

Feel free to contribute to this project. Please check the [TODO](TODO.md) file for more information.


## Install

### Client

The client javascript Glue library is located in **[client/dist/glue.js](client/dist/glue.js)**.
It requires jQuery.

### Server

Get the source and start hacking.

`go get github.com/desertbit/glue`


## Exsample

This socket library is very straightforward to use.
Check the sample directory for another exsample.


### Client

```html
<script>
	// Create and connect to the server.
	// Optional pass options.
	var socket = glue();

    socket.onMessage(function(data) {
        console.log("onMessage: " + data);

        // Echo the message back to the server.
        socket.send("echo: " + data);
    });


    socket.on("connected", function() {
        console.log("connected");
    });

    socket.on("connecting", function() {
        console.log("connecting");
    });

    socket.on("disconnected", function() {
        console.log("disconnected");
    });

    socket.on("reconnecting", function() {
        console.log("reconnecting");
    });

    socket.on("error", function(e, msg) {
        console.log("error: " + msg);
    });

    socket.on("connect_timeout", function() {
        console.log("connect_timeout");
    });

    socket.on("timeout", function() {
        console.log("timeout");
    });

    socket.on("discard_send_buffer", function(e, buf) {
        console.log("discard_send_buffer: ");
        for (var i = 0; i < buf.length; i++) {
        	console.log("  i: " + buf[i]);
        }
    });
</script>
```

### Server

```go
	package main

	import (
		"log"
		"net/http"

		"github.com/desertbit/glue"
	)

	const (
		ListenAddress = ":8888"
	)

	func main() {
		// Set the glue event function.
		glue.OnNewSocket(onNewSocket)

		// Start the http server.
		err := http.ListenAndServe(ListenAddress, nil)
		if err != nil {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}

	func onNewSocket(s *glue.Socket) {
		s.OnRead(func(data string) {
			log.Println("received: ", data)
		})

		s.Write("Hello World")
	}
```