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

var channel = (function() {
    /*
     * Variables
     */

     var instance = {}, // Our public instance object returned by this function.
         channels = {}; // Object as key value map.



     /*
      * Private Methods
      */

     var newChannel = function(name) {
         // Create the channel object.
         var channel = {
             // Set to a dummy function.
             onMessageFunc: function() {}
         };

         // Set the channel public instance object.
         // This is the value which is returned by the public glue.channel(...) function.
         channel.instance = {
             // onMessage sets the function which is triggered as soon as a message is received.
             onMessage: function(f) {
                 channel.onMessageFunc = f;
             },

             // send a data string to the channel.
             // One optional discard callback can be passed.
             // It is called if the data could not be send to the server.
             // The data is passed as first argument to the discard callback.
             // returns:
             //  1 if immediately send,
             //  0 if added to the send queue and
             //  -1 if discarded.
             send: function(data, discardCallback) {
                 // Discard empty data.
                 if (!data) {
                     return -1;
                 }

                 // Call the helper method and send the data to the channel.
                 return sendBuffered(Commands.ChannelData, utils.marshalValues(name, data), discardCallback);
             }
         };

         // Return the channel object.
         return channel;
     };



     /*
      * Public Methods
      */

     // Get or create a channel if it does not exists.
     instance.get = function(name) {
         if (!name) {
             return false;
         }

         // Get the channel.
         var c = channels[name];
         if (c) {
             return c.instance;
         }

         // Create a new one, if it does not exists and add it to the map.
         c = newChannel(name);
         channels[name] = c;

         return c.instance;
     };

     instance.emitOnMessage = function(name, data) {
         if (!name || !data) {
             return;
         }

         // Get the channel.
         var c = channels[name];
         if (!c) {
             console.log("glue: channel '" + name + "': emit onMessage event: channel does not exists");
             return;
         }

         // Call the channel's on message event.
         try {
             c.onMessageFunc(data);
         }
         catch(err) {
             console.log("glue: channel '" + name + "': onMessage event call failed: " + err.message);
             return;
         }
     };

     return instance;
})();
