# Glue - Robust Go and Javascript Socket Library

[![GoDoc](https://godoc.org/github.com/desertbit/glue?status.svg)](https://godoc.org/github.com/desertbit/glue)
[![Go Report Card](https://goreportcard.com/badge/github.com/desertbit/glue)](https://goreportcard.com/report/github.com/desertbit/glue)
[![Join the chat at https://gitter.im/desertbit/glue](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/desertbit/glue?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

Glue is a real-time bidirectional socket library. It is a **clean**, **robust** and **efficient** alternative to [socket.io](http://socket.io/). This library is designed to connect webbrowsers with a go-backend in a simple way. It automatically detects supported socket layers and chooses the most suitable one. This library handles automatic reconnections on disconnections and handles caching to bridge those disconnections. The server implementation is **thread-safe** and **stable**. The API is **fixed** and there won't be any breaking API changes.

## Socket layers
Currently two socket layers are supported:
- **WebSockets** - This is the primary option. They are used if the webbrowser supports WebSockets defined by [RFC 6455](https://tools.ietf.org/html/rfc6455).
- **AjaxSockets** - This socket layer is used as a fallback mode.

## Support
Feel free to contribute to this project. Please check the [TODO](TODO.md) file for more information.

## Install
### Client
The client javascript Glue library is located in **[client/dist/glue.js](client/dist/glue.js)**.

You can use bower to install the client library:

`bower install --save glue-socket`

### Server
Get the source and start hacking.

`go get github.com/desertbit/glue`

Import it with:

```go
import "github.com/desertbit/glue"
```

## Documentation
### Client - Javascript Library
A simple call to glue() without any options will establish a socket connection to the same host. A glue socket object is returned.

```js
// Create and connect to the server.
// Optional pass a host string and options.
var socket = glue();
```

Optional Javascript options which can be passed to Glue:

```js
var host = "https://foo.bar";

var opts = {
    // The base URL is appended to the host string. This value has to match with the server value.
    baseURL: "/glue/",

    // Force a socket type.
    // Values: false, "WebSocket", "AjaxSocket"
    forceSocketType: false,

    // Kill the connect attempt after the timeout.
    connectTimeout:  10000,

    // If the connection is idle, ping the server to check if the connection is stil alive.
    pingInterval:           35000,
    // Reconnect if the server did not response with a pong within the timeout.
    pingReconnectTimeout:   5000,

    // Whenever to automatically reconnect if the connection was lost.
    reconnect:          true,
    reconnectDelay:     1000,
    reconnectDelayMax:  5000,
    // To disable set to 0 (endless).
    reconnectAttempts:  10,

    // Reset the send buffer after the timeout.
    resetSendBufferTimeout: 10000
};

// Create and connect to the server.
// Optional pass a host string and options.
var socket = glue(host, opts);
```

The glue socket object has following public methods:

```js
// version returns the glue socket protocol version.
socket.version();

// type returns the current used socket type as string.
// Either "WebSocket" or "AjaxSocket".
socket.type();

// state returns the current socket state as string.
// Following states are available:
//  - "disconnected"
//  - "connecting"
//  - "reconnecting"
//  - "connected"
socket.state();

// socketID returns the socket's ID.
// This is a cryptographically secure pseudorandom number.
socket.socketID();

// send a data string to the server.
// One optional discard callback can be passed.
// It is called if the data could not be send to the server.
// The data is passed as first argument to the discard callback.
// returns:
//  1 if immediately send,
//  0 if added to the send queue and
//  -1 if discarded.
socket.send(data, discardCallback);

// onMessage sets the function which is triggered as soon as a message is received.
socket.onMessage(f);

// on binds event functions to events.
// This function is equivalent to jQuery's on method syntax.
// Following events are available:
//  - "connected"
//  - "connecting"
//  - "disconnected"
//  - "reconnecting"
//  - "error"
//  - "connect_timeout"
//  - "timeout"
//  - "discard_send_buffer"
socket.on();

// Reconnect to the server.
// This is ignored if the socket is not disconnected.
// It will reconnect automatically if required.
socket.reconnect();

// close the socket connection.
socket.close();

// channel returns the given channel object specified by name
// to communicate in a separate channel than the default one.
socket.channel(name);
```

A channel object has following public methods:

```js
// onMessage sets the function which is triggered as soon as a message is received.
c.onMessage(f);

// send a data string to the channel.
// One optional discard callback can be passed.
// It is called if the data could not be send to the server.
// The data is passed as first argument to the discard callback.
// returns:
//  1 if immediately send,
//  0 if added to the send queue and
//  -1 if discarded.
c.send(data, discardCallback);
```

### Server - Go Library
Check the Documentation at [GoDoc.org](https://godoc.org/github.com/desertbit/glue).

#### Use a custom HTTP multiplexer
If you choose to use a custom HTTP multiplexer, then it is possible to deactivate the automatic HTTP handler registration of glue.

```go
// Create a new glue server without configuring and starting the HTTP server.
server := glue.NewServer(glue.Options{
    HTTPSocketType: HTTPSocketTypeNone,
})

//...
```

The glue server implements the ServeHTTP method of the HTTP Handler interface of the http package. Use this to register the glue HTTP handler with a custom multiplexer. Be aware, that the URL of the custom HTTP handler has to match with the glue HTTPHandleURL options string.

#### Reading data
Data has to be read from the socket and each channel. If you don't require to read data from the socket or a channel, then discard received data with the DiscardRead() method. If received data is not discarded, then the read buffer will block as soon as it is full, which will also block the keep-alive mechanism of the socket. The result would be a closed socket...

```go
// ...

// Discard received data from the main socket channel.
// Hint: Channels have to be discarded separately.
s.DiscardRead()

// ...

// Create a channel.
c := s.Channel("golang")

// Discard received data from a channel.
c.DiscardRead()
```

#### Bind custom values to a socket
The socket.Value interface is a placeholder for custom data.

```go
type CustomValues struct {
    Foo string
    Bar int
}

// ...

s.Value = &CustomValues{
    Foo: "Hello World",
    Bar: 900,
}

// ...

v, ok := s.Value.(*CustomValues)
if !ok {
    // Handle error
    return
}
```

### Channels
Channels are separate communication channels from the client to the server of a single socket connections. Multiple separate communication channels can be created:

Server:

```go
// ...

// Create a channel.
c := s.Channel("golang")

// Set the channel on read event function.
c.OnRead(func(data string) {
    // ...
})

// Write to the channel.
c.Write("Hello Gophers!")
```

Client:

```js
var c = socket.channel("golang");

c.onMessage(function(data) {
    console.log(data);
});

c.send("Hello World");
```

### Broadcasting Messages

With Glue it is easy to broadcast messages to multiple clients. The Glue Server keeps track of all active connected client sessions.
You can make use of the server **Sockets**, **GetSocket** or **OnNewSocket** methods to implement broadcasting.


## Example
This socket library is very straightforward to use. Check the [sample directory](sample) for more examples.

### Client

```html
<script>
    // Create and connect to the server.
    // Optional pass a host string and options.
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

    socket.on("discard_send_buffer", function() {
        console.log("some data could not be send and was discarded.");
    });
</script>
```

### Server
Read data from the socket with a read event function. Check the sample directory for other ways of reading data from the socket.

```go
import (
    "log"
    "net/http"

    "github.com/desertbit/glue"
)

func main() {
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
```

## Similar Go Projects
- [go-socket.io](https://github.com/googollee/go-socket.io) - socket.io library for golang, a realtime application framework.
