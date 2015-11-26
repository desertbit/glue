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

package glue

import (
	"net/http"
	"net/url"
	"strings"
)

//#################//
//### Constants ###//
//#################//

// A HTTPSocketType defines which socket type to use for the HTTP glue server.
type HTTPSocketType int

const (
	// HTTPSocketTypeNone defines to not configure and run a HTTP server.
	HTTPSocketTypeNone HTTPSocketType = 1 << iota

	// HTTPSocketTypeTCP defines to use a TCP server.
	HTTPSocketTypeTCP HTTPSocketType = 1 << iota

	// HTTPSocketTypeUnix defines to use a Unix socket server.
	HTTPSocketTypeUnix HTTPSocketType = 1 << iota
)

//####################//
//### Options type ###//
//####################//

// Options holds the glue server options.
type Options struct {
	// HTTPSocketType defines which socket type to use for the HTTP glue server.
	// Default: HTTPSocketTypeTCP
	HTTPSocketType HTTPSocketType

	// The HTTP address to listen on.
	// Default: ":80"
	HTTPListenAddress string

	// HTTPHandleURL defines the base url to handle glue HTTP socket requests.
	// This has to be set, even if the none socket type is used.
	// Default: "/glue/"
	HTTPHandleURL string

	// CheckOrigin returns true if the request Origin header is acceptable. If
	// CheckOrigin is nil, the host in the Origin header must not be set or
	// must match the host of the request.
	// This method is used by the backend sockets before establishing connections.
	CheckOrigin func(r *http.Request) bool

	// Enables the Cross-Origin Resource Sharing (CORS) mechanism.
	// This will set the Access-Control-Allow-Origin HTTP headers.
	// A resource makes a cross-origin HTTP request when it requests a resource
	// from a different domain than the one which served itself.
	EnableCORS bool
}

// SetDefaults sets unset option values to its default value.
func (o *Options) SetDefaults() {
	// Set the socket type.
	if o.HTTPSocketType != HTTPSocketTypeNone &&
		o.HTTPSocketType != HTTPSocketTypeTCP &&
		o.HTTPSocketType != HTTPSocketTypeUnix {
		o.HTTPSocketType = HTTPSocketTypeTCP
	}

	// Set the listen address.
	if len(o.HTTPListenAddress) == 0 {
		o.HTTPListenAddress = ":80"
	}

	// Set the handle URL.
	if len(o.HTTPHandleURL) == 0 {
		o.HTTPHandleURL = "/glue/"
	}

	// Be sure that the handle URL ends with a slash.
	if !strings.HasSuffix(o.HTTPHandleURL, "/") {
		o.HTTPHandleURL += "/"
	}

	// Set the default check origin function if not set.
	if o.CheckOrigin == nil {
		o.CheckOrigin = checkSameOrigin
	}
}

//###############//
//### Private ###//
//###############//

// checkSameOrigin returns true if the origin is not set or is equal to the request host.
// Source from gorilla websockets.
func checkSameOrigin(r *http.Request) bool {
	origin := r.Header["Origin"]
	if len(origin) == 0 {
		return true
	}
	u, err := url.Parse(origin[0])
	if err != nil {
		return false
	}
	return u.Host == r.Host
}
