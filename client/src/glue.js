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

var glue = function(host, options) {
    // Turn on strict mode.
    'use strict';

    // Include the dependencies.
    @@include('./emitter.js')
    @@include('./websocket.js')
    @@include('./ajaxsocket.js')



    /*
     * Constants
     */

    var Version         = "1.9.1",
        MainChannelName = "m";

    var SocketTypes = {
        WebSocket:  "WebSocket",
        AjaxSocket: "AjaxSocket"
    };

    var Commands = {
        Len: 	            2,
        Init:               'in',
        Ping:               'pi',
        Pong:               'po',
        Close: 	            'cl',
        Invalid:            'iv',
        DontAutoReconnect:  'dr',
        ChannelData:        'cd'
    };

    var States = {
        Disconnected:   "disconnected",
        Connecting:     "connecting",
        Reconnecting:   "reconnecting",
        Connected:      "connected"
    };

    var DefaultOptions = {
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



    /*
     * Variables
     */

    var emitter                 = new Emitter,
        bs                      = false,
        mainChannel,
        initialConnectedOnce    = false,    // If at least one successful connection was made.
        bsNewFunc,                          // Function to create a new backend socket.
        currentSocketType,
        currentState            = States.Disconnected,
        reconnectCount          = 0,
        autoReconnectDisabled   = false,
        connectTimeout          = false,
        pingTimeout             = false,
        pingReconnectTimeout    = false,
        sendBuffer              = [],
        resetSendBufferTimeout  = false,
        resetSendBufferTimedOut = false,
        isReady                 = false,    // If true, the socket is initialized and ready.
        beforeReadySendBuffer   = [],       // Buffer to hold requests for the server while the socket is not ready yet.
        socketID               = "";


    /*
     * Include the dependencies
     */

    // Exported helper methods for the dependencies.
    var closeSocket, send, sendBuffered;

    @@include('./utils.js')
    @@include('./channel.js')



    /*
     * Methods
     */

    // Function variables.
    var reconnect, triggerEvent;

    // Sends the data to the server if a socket connection exists, otherwise it is discarded.
    // If the socket is not ready yet, the data is buffered until the socket is ready.
    send = function(data) {
        if (!bs) {
            return;
        }

        // If the socket is not ready yet, buffer the data.
        if (!isReady) {
            beforeReadySendBuffer.push(data);
            return;
        }

        // Send the data.
        bs.send(data);
    };

    // Hint: the isReady flag has to be true before calling this function!
    var sendBeforeReadyBufferedData = function() {
        // Skip if empty.
        if (beforeReadySendBuffer.length === 0) {
            return;
        }

        // Send the buffered data.
        for (var i = 0; i < beforeReadySendBuffer.length; i++) {
            send(beforeReadySendBuffer[i]);
        }

        // Clear the buffer.
        beforeReadySendBuffer = [];
    };

    var stopResetSendBufferTimeout = function() {
        // Reset the flag.
        resetSendBufferTimedOut = false;

        // Stop the timeout timer if present.
        if (resetSendBufferTimeout !== false) {
            clearTimeout(resetSendBufferTimeout);
            resetSendBufferTimeout = false;
        }
    };

    var startResetSendBufferTimeout = function() {
        // Skip if already running or if already timed out.
        if (resetSendBufferTimeout !== false || resetSendBufferTimedOut) {
            return;
        }

        // Start the timeout.
        resetSendBufferTimeout = setTimeout(function() {
            // Update the flags.
            resetSendBufferTimeout = false;
            resetSendBufferTimedOut = true;

            // Return if already empty.
            if (sendBuffer.length === 0) {
                return;
            }

            // Call the discard callbacks if defined.
            var buf;
            for (var i = 0; i < sendBuffer.length; i++) {
                buf = sendBuffer[i];
                if (buf.discardCallback && utils.isFunction(buf.discardCallback)) {
                    try {
                        buf.discardCallback(buf.data);
                    }
                    catch (err) {
                       console.log("glue: failed to call discard callback: " + err.message);
                    }
                }
            }

            // Trigger the event if any buffered send data is discarded.
            triggerEvent("discard_send_buffer");

            // Reset the buffer.
            sendBuffer = [];
        }, options.resetSendBufferTimeout);
    };

    var sendDataFromSendBuffer = function() {
        // Stop the reset send buffer tiemout.
        stopResetSendBufferTimeout();

        // Skip if empty.
        if (sendBuffer.length === 0) {
            return;
        }

        // Send data, which could not be send...
        var buf;
        for (var i = 0; i < sendBuffer.length; i++) {
            buf = sendBuffer[i];
            send(buf.cmd + buf.data);
        }

        // Clear the buffer again.
        sendBuffer = [];
    };

    // Send data to the server.
    // This is a helper method which handles buffering,
    // if the socket is currently not connected.
    // One optional discard callback can be passed.
    // It is called if the data could not be send to the server.
    // The data is passed as first argument to the discard callback.
    // returns:
    //  1 if immediately send,
    //  0 if added to the send queue and
    //  -1 if discarded.
    sendBuffered = function(cmd, data, discardCallback) {
        // Be sure, that the data value is an empty
        // string if not passed to this method.
        if (!data) {
            data = "";
        }

        // Add the data to the send buffer if disconnected.
        // They will be buffered for a short timeout to bridge short connection errors.
        if (!bs || currentState !== States.Connected) {
            // If already timed out, then call the discard callback and return.
            if (resetSendBufferTimedOut) {
                if (discardCallback && utils.isFunction(discardCallback)) {
                    discardCallback(data);
                }

                return -1;
            }

            // Reset the send buffer after a specific timeout.
            startResetSendBufferTimeout();

            // Append to the buffer.
            sendBuffer.push({
                cmd:                cmd,
                data:               data,
                discardCallback:    discardCallback
            });

            return 0;
        }

        // Send the data with the command to the server.
        send(cmd + data);

        return 1;
    };

    var stopConnectTimeout = function() {
        // Stop the timeout timer if present.
        if (connectTimeout !== false) {
            clearTimeout(connectTimeout);
            connectTimeout = false;
        }
    };

    var resetConnectTimeout = function() {
        // Stop the timeout.
        stopConnectTimeout();

        // Start the timeout.
        connectTimeout = setTimeout(function() {
            // Update the flag.
            connectTimeout = false;

            // Trigger the event.
            triggerEvent("connect_timeout");

            // Reconnect to the server.
            reconnect();
        }, options.connectTimeout);
    };

    var stopPingTimeout = function() {
        // Stop the timeout timer if present.
        if (pingTimeout !== false) {
            clearTimeout(pingTimeout);
            pingTimeout = false;
        }

        // Stop the reconnect timeout.
        if (pingReconnectTimeout !== false) {
            clearTimeout(pingReconnectTimeout);
            pingReconnectTimeout = false;
        }
    };

    var resetPingTimeout = function() {
        // Stop the timeout.
        stopPingTimeout();

        // Start the timeout.
        pingTimeout = setTimeout(function() {
            // Update the flag.
            pingTimeout = false;

            // Request a Pong response to check if the connection is still alive.
            send(Commands.Ping);

            // Start the reconnect timeout.
            pingReconnectTimeout = setTimeout(function() {
                // Update the flag.
                pingReconnectTimeout = false;

                // Trigger the event.
                triggerEvent("timeout");

                // Reconnect to the server.
                reconnect();
            }, options.pingReconnectTimeout);
        }, options.pingInterval);
    };

    var newBackendSocket = function() {
        // If at least one successfull connection was made,
        // then create a new socket using the last create socket function.
        // Otherwise determind which socket layer to use.
        if (initialConnectedOnce) {
            bs = bsNewFunc();
            return;
        }

        // Fallback to the ajax socket layer if there was no successful initial
        // connection and more than one reconnection attempt was made.
        if (reconnectCount > 1) {
            bsNewFunc = newAjaxSocket;
            bs = bsNewFunc();
            currentSocketType = SocketTypes.AjaxSocket;
            return;
        }

        // Choose the socket layer depending on the browser support.
        if ((!options.forceSocketType && window.WebSocket) ||
            options.forceSocketType === SocketTypes.WebSocket)
        {
            bsNewFunc = newWebSocket;
            currentSocketType = SocketTypes.WebSocket;
        }
        else
        {
            bsNewFunc = newAjaxSocket;
            currentSocketType = SocketTypes.AjaxSocket;
        }

        // Create the new socket.
        bs = bsNewFunc();
    };

    var initSocket = function(data) {
        // Parse the data JSON string to an object.
        data = JSON.parse(data);

        // Validate.
        // Close the socket and log the error on invalid data.
        if (!data.socketID) {
            closeSocket();
            console.log("glue: socket initialization failed: invalid initialization data received");
            return;
        }

        // Set the socket ID.
        socketID = data.socketID;

        // The socket initialization is done.
        // ##################################

        // Set the ready flag.
        isReady = true;

        // First send all data messages which were
        // buffered because the socket was not ready.
        sendBeforeReadyBufferedData();

        // Now set the state and trigger the event.
        currentState = States.Connected;
        triggerEvent("connected");

        // Send the queued data from the send buffer if present.
        // Do this after the next tick to be sure, that
        // the connected event gets fired first.
        setTimeout(sendDataFromSendBuffer, 0);
    };

    var connectSocket = function() {
        // Set a new backend socket.
        newBackendSocket();

        // Set the backend socket events.
        bs.onOpen = function() {
            // Stop the connect timeout.
            stopConnectTimeout();

            // Reset the reconnect count.
            reconnectCount = 0;

            // Set the flag.
            initialConnectedOnce = true;

            // Reset or start the ping timeout.
            resetPingTimeout();

            // Prepare the init data to be send to the server.
            var data = {
                version: Version
            };

            // Marshal the data object to a JSON string.
            data = JSON.stringify(data);

            // Send the init data to the server with the init command.
            // Hint: the backend socket is used directly instead of the send function,
            // because the socket is not ready yet and this part belongs to the
            // initialization process.
            bs.send(Commands.Init + data);
        };

        bs.onClose = function() {
            // Reconnect the socket.
            reconnect();
        };

        bs.onError = function(msg) {
            // Trigger the error event.
            triggerEvent("error", [msg]);

            // Reconnect the socket.
            reconnect();
        };

        bs.onMessage = function(data) {
            // Reset the ping timeout.
            resetPingTimeout();

            // Log if the received data is too short.
            if (data.length < Commands.Len) {
                console.log("glue: received invalid data from server: data is too short.");
                return;
            }

            // Extract the command from the received data string.
            var cmd = data.substr(0, Commands.Len);
            data = data.substr(Commands.Len);

            if (cmd === Commands.Ping) {
                // Response with a pong message.
                send(Commands.Pong);
            }
            else if (cmd === Commands.Pong) {
                // Don't do anything.
                // The ping timeout was already reset.
            }
            else if (cmd === Commands.Invalid) {
                // Log.
                console.log("glue: server replied with an invalid request notification!");
            }
            else if (cmd === Commands.DontAutoReconnect) {
                // Disable auto reconnections.
                autoReconnectDisabled = true;

                // Log.
                console.log("glue: server replied with an don't automatically reconnect request. This might be due to an incompatible protocol version.");
            }
            else if (cmd === Commands.Init) {
                initSocket(data);
            }
            else if (cmd === Commands.ChannelData) {
                // Obtain the two values from the data string.
                var v = utils.unmarshalValues(data);
                if (!v) {
                    console.log("glue: server requested an invalid channel data request: " + data);
                    return;
                }

                // Trigger the event.
                channel.emitOnMessage(v.first, v.second);
            }
            else {
                console.log("glue: received invalid data from server with command '" + cmd + "' and data '" + data + "'!");
            }
        };

        // Connect during the next tick.
        // The user should be able to connect the event functions first.
        setTimeout(function() {
            // Set the state and trigger the event.
            if (reconnectCount > 0) {
                currentState = States.Reconnecting;
                triggerEvent("reconnecting");
            }
            else {
                currentState = States.Connecting;
                triggerEvent("connecting");
            }

            // Reset or start the connect timeout.
            resetConnectTimeout();

            // Connect to the server
            bs.open();
        }, 0);
    };

    var resetSocket = function() {
        // Stop the timeouts.
        stopConnectTimeout();
        stopPingTimeout();

        // Reset flags and variables.
        isReady = false;
        socketID = "";

        // Clear the buffer.
        // This buffer is attached to each single socket.
        beforeReadySendBuffer = [];

        // Reset previous backend sockets if defined.
        if (bs) {
            // Set dummy functions.
            // This will ensure, that previous old sockets don't
            // call our valid methods. This would mix things up.
            bs.onOpen = bs.onClose = bs.onMessage = bs.onError = function() {};

            // Reset everything and close the socket.
            bs.reset();
            bs = false;
        }
    };

    reconnect = function() {
        // Reset the socket.
        resetSocket();

        // If no reconnections should be made or more than max
        // reconnect attempts where made, trigger the disconnected event.
        if ((options.reconnectAttempts > 0 && reconnectCount > options.reconnectAttempts) ||
            options.reconnect === false || autoReconnectDisabled)
        {
            // Set the state and trigger the event.
            currentState = States.Disconnected;
            triggerEvent("disconnected");

            return;
        }

        // Increment the count.
        reconnectCount += 1;

        // Calculate the reconnect delay.
        var reconnectDelay = options.reconnectDelay * reconnectCount;
        if (reconnectDelay > options.reconnectDelayMax) {
            reconnectDelay = options.reconnectDelayMax;
        }

        // Try to reconnect.
        setTimeout(function() {
            connectSocket();
        }, reconnectDelay);
    };

    closeSocket = function() {
        // Check if the socket exists.
        if (!bs) {
            return;
        }

        // Notify the server.
        send(Commands.Close);

        // Reset the socket.
        resetSocket();

        // Set the state and trigger the event.
        currentState = States.Disconnected;
        triggerEvent("disconnected");
    };



    /*
     * Initialize section
     */

    // Create the main channel.
    mainChannel = channel.get(MainChannelName);

    // Prepare the host string.
    // Use the current location if the host string is not set.
    if (!host) {
        host = window.location.protocol + "//" + window.location.host;
    }
    // The host string has to start with http:// or https://
    if (!host.match("^http://") && !host.match("^https://")) {
        console.log("glue: invalid host: missing 'http://' or 'https://'!");
        return;
    }

    // Merge the options with the default options.
    options = utils.extend({}, DefaultOptions, options);

    // The max value can't be smaller than the delay.
    if (options.reconnectDelayMax < options.reconnectDelay) {
        options.reconnectDelayMax = options.reconnectDelay;
    }

    // Prepare the base URL.
    // The base URL has to start and end with a slash.
    if (options.baseURL.indexOf("/") !== 0) {
        options.baseURL = "/" + options.baseURL;
    }
    if (options.baseURL.slice(-1) !== "/") {
        options.baseURL = options.baseURL + "/";
    }

    // Create the initial backend socket and establish a connection to the server.
    connectSocket();



    /*
     * Socket object
     */

    var socket = {
        // version returns the glue socket protocol version.
        version: function() {
            return Version;
        },

        // type returns the current used socket type as string.
        // Either "WebSocket" or "AjaxSocket".
        type: function() {
            return currentSocketType;
        },

        // state returns the current socket state as string.
        // Following states are available:
        //  - "disconnected"
        //  - "connecting"
        //  - "reconnecting"
        //  - "connected"
        state: function() {
            return currentState;
        },

        // socketID returns the socket's ID.
        // This is a cryptographically secure pseudorandom number.
        socketID: function() {
            return socketID;
        },

        // send a data string to the server.
        // One optional discard callback can be passed.
        // It is called if the data could not be send to the server.
        // The data is passed as first argument to the discard callback.
        // returns:
        //  1 if immediately send,
        //  0 if added to the send queue and
        //  -1 if discarded.
        send: function(data, discardCallback) {
            mainChannel.send(data, discardCallback);
        },

        // onMessage sets the function which is triggered as soon as a message is received.
        onMessage: function(f) {
            mainChannel.onMessage(f);
        },

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
        on: function() {
            emitter.on.apply(emitter, arguments);
        },

        // Reconnect to the server.
        // This is ignored if the socket is not disconnected.
        // It will reconnect automatically if required.
        reconnect: function() {
            if (currentState !== States.Disconnected) {
                return;
            }

            // Reset the reconnect count and the auto reconnect disabled flag.
            reconnectCount = 0;
            autoReconnectDisabled = false;

            // Reconnect the socket.
            reconnect();
        },

        // close the socket connection.
        close: function() {
            closeSocket();
        },

        // channel returns the given channel object specified by name
        // to communicate in a separate channel than the default one.
        channel: function(name) {
            return channel.get(name);
        }
    };

    // Define the function body of the triggerEvent function.
    triggerEvent = function() {
        emitter.emit.apply(emitter, arguments);
    };

    // Return the newly created socket.
    return socket;
};
