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


var newAjaxSocket = function () {
    /*
     * Constants
     */

    var ajaxHost = host + options.baseURL + "ajax",
        sendTimeout = 8000,
        pollTimeout = 45000;

    var PollCommands = {
        Timeout:    "t",
        Closed:     "c"
    };

    var Commands = {
        Delimiter:  "&",
        Init:       "i",
        Push:       "u",
        Poll:       "o"
    };



    /*
     * Variables
     */

    var s = {},
        uid, pollToken,
        pollXhr = false,
        sendXhr = false,
        poll;



    /*
     * Methods
     */

    var stopRequests = function() {
        // Set the poll function to a dummy function.
        // This will prevent further poll calls.
        poll = function() {};

        // Kill the ajax requests.
        if (pollXhr) {
            pollXhr.abort();
        }
        if (sendXhr) {
            sendXhr.abort();
        }
    };

    var triggerClosed = function() {
        // Stop the ajax requests.
        stopRequests();

        // Trigger the event.
        s.onClose();
    };

    var triggerError = function(msg) {
        // Stop the ajax requests.
        stopRequests();

        // Create the error message.
        if (msg) {
            msg = "the ajax socket closed the connection with the error: " + msg;
        }
        else {
            msg = "the ajax socket closed the connection with an error.";
        }

        // Trigger the event.
        s.onError(msg);
    };

    var send = function (data, callback) {
        sendXhr = $.ajax({
            url: ajaxHost,
            success: function (data) {
                sendXhr = false;

                if (callback) {
                    callback(data);
                }
            },
            error: function (r, msg) {
                sendXhr = false;
                triggerError(msg);
            },
            type: "POST",
            data: data,
            dataType: "text",
            timeout: sendTimeout
        });
    };

    poll = function () {
        pollXhr = $.ajax({
            url: ajaxHost,
            success: function (data) {
                pollXhr = false;

                // Check if this jax request has reached the server's timeout.
                if (data == PollCommands.Timeout) {
                    // Just start the next poll request.
                    poll();
                    return;
                }

                // Check if this ajax connection was closed.
                if (data == PollCommands.Closed) {
                    // Trigger the closed event.
                    triggerClosed();
                    return;
                }

                // Split the new token from the rest of the data.
                var i = data.indexOf(Commands.Delimiter);
                if (i < 0) {
                    triggerError("ajax socket: failed to split poll token from data!");
                    return;
                }

                // Set the new token and the data variable.
                pollToken = data.substring(0, i);
                data = data.substr(i + 1);

                // Start the next poll request.
                poll();

                // Call the event.
                s.onMessage(data);
            },
            error: function (r, msg) {
                pollXhr = false;
                triggerError(msg);
            },
            type: "POST",
            data: Commands.Poll + uid + Commands.Delimiter + pollToken,
            dataType: "text",
            timeout: pollTimeout
        });
    };



    /*
     * Socket layer implementation.
     */

    s.open = function () {
        // Initialize the ajax socket session
        send(Commands.Init, function (data) {
            // Get the uid and token string
            var i = data.indexOf(Commands.Delimiter);
            if (i < 0) {
                triggerError("ajax socket: failed to split uid and poll token from data!");
                return;
            }

            // Set the uid and token.
            uid = data.substring(0, i);
            pollToken = data.substr(i + 1);

            // Start the long polling process.
            poll();

            // Trigger the event.
            s.onOpen();
        });
    };

    s.send = function (data) {
        // Always prepend the command with the uid to the data.
        send(Commands.Push + uid + Commands.Delimiter + data);
    };

	s.reset = function() {
        // Stop the ajax requests.
        stopRequests();
    };

	return s;
};
