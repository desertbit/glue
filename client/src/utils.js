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

var utils = (function() {
    /*
     * Constants
     */

    var Delimiter = "&";



    /*
     * Variables
     */

     var instance = {}; // Our public instance object returned by this function.



    /*
     * Public Methods
     */

    // unmarshalValues splits two values from a single string.
    // This function is chainable to extract multiple values.
    // An object with two strings (first, second) is returned.
    instance.unmarshalValues = function(data) {
        if (!data) {
            return false;
        }

        // Find the delimiter position.
        var pos = data.indexOf(Delimiter);

        // Extract the value length integer of the first value.
        var len = parseInt(data.substring(0, pos), 10);
        data = data.substring(pos + 1);

        // Validate the length.
        if (len < 0 || len > data.length) {
            return false;
        }

        // Now split the first value from the second.
        var firstV = data.substr(0, len);
        var secondV = data.substr(len);

        // Return an object with both values.
        return {
            first:  firstV,
            second: secondV
        };
    };

    // marshalValues joins two values into a single string.
    // They can be decoded by the unmarshalValues function.
    instance.marshalValues = function(first, second) {
        return String(first.length) + Delimiter + first + second;
    };


    return instance;
})();
