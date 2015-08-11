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

var utils = new function() {
    /*
     * Public Methods
     */

    // unmarshalValues splits two values from a single string.
    // This function is chainable to extract multiple values.
    // An object with two strings (first, second) is returned.
    this.unmarshalValues = function(data) {
        if (!data) {
            return false
        }

        // Find the delimiter position.
        var pos = data.indexOf(Delimiter);

        // Extract the value length integer of the first value.
        var len = parseInt(data.substring(0, pos), 10);
        data = data.substring(pos + 1);

        // Validate the length.
        if (len < 0 || len > data.length) {
            return false
        }

        // Now split the first value from the second.
        var firstV = data.substr(0, len);
        var secondV = data.substr(len);

        // Return an object with both values.
        return {
            first:  firstV,
            second: secondV
        }
    };

    // marshalValues joins two values into a single string.
    // They can be decoded by the unmarshalValues function.
    this.marshalValues = function(first, second) {
        return String(first.length) + Delimiter + first + second;
    };
};
