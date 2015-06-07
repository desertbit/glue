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

'use strict';

var glue = function(host, options) {

    // Include the sockets layer implementations.
    @@include('./websocket.js')
    @@include('./ajaxsocket.js')



    /*
     * Constants
     */

    var SocketTypes = {
        WebSocket:  "WebSocket",
        AjaxSocket: "AjaxSocket"
    }

    var Commands = {
        Len: 	  1,
        Ping:    'i',
        Pong:    'o',
        Data: 	 'd',
        Invalid: 'z'
    };

    var States = {
        Disconnected:   "disconnected",
        Connecting:     "connecting",
        Reconnecting:   "reconnecting",
        Connected:      "connected"
    };

    var DefaultOptions = {
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
        reconnected:        true,
        reconnectDelay:     1000,
        reconnectDelayMax:  5000,
        // To disable set to 0 (endless).
        reconnectAttempts:  10,

        // Reset the send buffer after the timeout.
        resetSendBufferTimeout: 7000 
    };



    /*
     * Variables
     */

    var bs                      = false,
        initialConnectedOnce    = false,    // If atleast one successful connection was made.
        bsNewFunc,                          // Function to create a new backend socket.
        currentSocketType,
        currentState            = States.Disconnected,
        reconnectCount          = 0,
        connectTimeout          = false,
        pingTimeout             = false,
        pingReconnectTimeout    = false,
        sendBuffer              = [],
        resetSendBufferTimeout  = false,
        resetSendBufferTimedOut = false,
        onMessageFunc           = function() {}; // Set to a dummy function.



    /*
     * Methods
     */

    var reconnect, triggerEvent;

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

            // Trigger the event if any buffered send data is discarded.
            if (sendBuffer.length > 0) {
                triggerEvent("discard_send_buffer", [sendBuffer]);
            }

            // Reset the buffer.
            sendBuffer = [];
        }, options.resetSendBufferTimeout); 
    };

    var sendDataFromSendBuffer = function() {
        // Stop the reset send buffer tiemout.
        stopResetSendBufferTimeout();

        // Skip if empty.
        if (sendBuffer.length == 0) {
            return;
        }

        // Send data, which could not be send...
        for (var i = 0; i < sendBuffer.length; i++) {
            bs.send(Commands.Data + sendBuffer[i]);
        }

        // Clear the buffer again.
        sendBuffer = [];
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
            bs.send(Commands.Ping);

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
        if ((!options.forceSocketType && window["WebSocket"])
                || options.forceSocketType === SocketTypes.WebSocket)
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

    var connectSocket = function() {
        // Connect during the next tick.
        // The user should be able to connect event functions first.
        setTimeout(function() {
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

                // Set the state and trigger the event.
                currentState = States.Connected;
                triggerEvent("connected");

                // Send the queued data from the send buffer if present.
                // Do this after the next tick to be sure, that
                // the connected event gets fired first.
                setTimeout(sendDataFromSendBuffer, 0);
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
                var cmd = data.substr(0,Commands.Len);
                data = data.substr(Commands.Len);

                if (cmd === Commands.Ping) {
                    // Response with a pong message.
                    bs.send(Commands.Pong);
                }
                else if (cmd === Commands.Pong) {
                    // Don't do anything.
                    // The ping timeout was already reset.
                }
                else if (cmd === Commands.Invalid) {
                    // Log.
                    console.log("glue: server replied with an invalid request notification!");
                }
                else if (cmd === Commands.Data) {
                    // Call the on message event.
                    if (data) {
                        try {
                            onMessageFunc(data);
                        }
                        catch(err) {
                            console.log("glue: onMessage event call failed: " + err.message);
                        }  
                    }
                }
                else {
                    console.log("glue: received invalid data from server with command '" + cmd + "' and data '" + data + "'!");
                }
            };

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
        // Reset previous backend sockets if defined.
        if (!bs) {
            return;
        }

        // Stop the timeouts.
        stopConnectTimeout();
        stopPingTimeout();

        // Set dummy functions.
        // This will ensure, that previous old sockets don't
        // call our valid methods. This would mix things up.
        bs.onOpen = bs.onClose = bs.onMessage = bs.onError = function() {};

        // Reset everything and close the socket.
        bs.reset();
        bs = false;
    };



    reconnect = function() {
        // Reset the socket.
        resetSocket();

        // If no reconnections should be made or more than max
        // reconnect attempts where made, trigger the error event.
        if ((options.reconnectAttempts > 0 && reconnectCount > options.reconnectAttempts)
                || options.reconnect === false)
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



    /*
     * Initialize section
     */

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
    options = $.extend(DefaultOptions, options);

    // The max value can't be smaller than the delay.
    if (options.reconnectDelayMax < options.reconnectDelay) {
        options.reconnectDelayMax = options.reconnectDelay;
    }

    // Create the initial backend socket and establish a connection to the server.
    connectSocket();



    /*
     * Socket object
     */

    var socket = {
        // type returns the current used socket type as string. 
        type: function() {
            return currentSocketType;
        },

        // state returns the current socket state as string. 
        state: function() {
            return currentState;
        },

        // send a data string to the server.
        // returns
        //  1 if immediately send,
        //  0 if added to the send queue and
        //  -1 if discarded.
        send: function(data) {
            // Add the data to the send buffer if disconnected.
            // They will be buffered for a short timeout to bridge short connection errors.
            if (!bs || currentState !== States.Connected) {
                // If already timed out, just discard this.
                if (resetSendBufferTimedOut) {
                    return -1;
                }

                // Reset the send buffer after a specific timeout.
                startResetSendBufferTimeout();

                // Append to the buffer.
                sendBuffer.push(data);

                return 0;
            }

            // Send the data with the data command to the server.
            bs.send(Commands.Data + data);

            return 1;
        },

        // onMessage sets the function which is triggered as soon as a message is received.
        onMessage: function(f) {
            onMessageFunc = f;
        },

        // on binds event functions to events.
        // This function is equivalent to jQuery's on method.
        on: function() {
            var s = $(socket);
            s.on.apply(s, arguments);
        },

        // Reconnect to the server.
        // This is ignored if the socket is not disconnected.
        // It will reconnect automatically if required.
        reconnect: function() {
            if (currentState !== States.Disconnected) {
                return;
            }

            // Reset the reconnect count.
            reconnectCount = 0;

            // Reconnect the socket.
            reconnect();
        },

        // close the socket connection.
        close: function() {
            // Check if the socket exists.
            if (!bs) {
                return;
            }

            // Reset the socket.
            resetSocket();

            // Set the state and trigger the event.
            currentState = States.Disconnected;
            triggerEvent("disconnected");
        }
    };

    // Define the function body of the triggerEvent function.
    triggerEvent = function() {
        var s = $(socket);
        s.triggerHandler.apply(s, arguments);
    };

    // Return the newly created socket.
    return socket;
};