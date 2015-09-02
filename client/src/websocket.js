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
            url += options.baseURL + "ws";

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
