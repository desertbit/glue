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

/*
 *  This code lives inside the glue function.
 */


var newWebSocket = function () {
    /*
     * Variables
     */

    var s = {},
        ws;



    /*
     * Socket layer implementation.
     */

    s.open = function () {
        try {
            // Generate the websocket url.
            var url;
            if (host.match("^https://")) {
                url = "wss" + host.substr(5);
            } else {
                url = "ws" + host.substr(4);
            }
            url += "/glue/ws";

            // Open the websocket connection
            ws = new WebSocket(url);

            // Set the callback handlers
            ws.onmessage = function(event) {
                s.onMessage(event.data.toString());
            };

            ws.onerror = function(event) {
                var msg = "the websocket closed the connection with ";
                if (event.code) {
                    msg += "the error code: " + event.code;
                }
                else {
                    msg += "an error.";
                }

                s.onError(msg);
            };

            ws.onclose = function() {
                s.onClose();
            };

            ws.onopen = function() {
                s.onOpen();
            };
        } catch (e) {
            s.onError();
        }
    };

    s.send = function (data) {
        // Send the data to the server
        ws.send(data);     
    };

	s.reset = function() {
        // Close the websocket if defined.
        if (ws) {
            ws.close();
        }

        ws = undefined;
    };

	return s;
};