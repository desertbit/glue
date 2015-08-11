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

var channel = new function() {
    /*
     * Variables
     */

     var channels = {}; // Object as key value map.



     /*
      * Private Methods
      */

     var newChannel = function(name) {
         // Create and return the channel object.
         return new function() {
             // Private
             // ######

             var parent = this;

             // Public
             // ######

             // Set to a dummy function.
             this.onMessageFunc = function() {};

             // The public instance object.
             // This is the value which is returned by glue.channel(...) function.
             this.instance = {
                 // onMessage sets the function which is triggered as soon as a message is received.
                 onMessage: function(f) {
                     parent.onMessageFunc = f;
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
         };
     };



     /*
      * Public Methods
      */

     // Get or create a channel if it does not exists.
     this.get = function(name) {
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

     this.emitOnMessage = function(name, data) {
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
};
