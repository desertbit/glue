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


var newAjaxSocket = function () {
    /*
     * Constants
     */

    var ajaxHost = host + "/glue/ajax",
        sendTimeout = 8000,
        pollTimeout = 45000;

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

                // Split the new token from the rest of the data.
                var i = data.indexOf(Commands.Delimiter);
                if (i < 0) {
                    triggerError("ajax socket: failed to split poll token from data!");
                    return;
                }

                // Set the new token and the data variable.
                pollToken = data.substring(0, i);
                data = data.substr(i + 1);

                // Start the next poll request
                poll();

                // Call the event
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