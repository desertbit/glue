/*
 *  Glue - Robust Go and Javascript Socket Library
 *  Copyright (C) 2015  Roland Singer <roland.singer[at]desertbit.com>
 * 
 *  SPDX-License-Identifier: MIT
 */

// Package utils provides utilities for the glue socket implementation.
package utils

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

//#################//
//### Constants ###//
//#################//

const (
	delimiter = "&"
)

//########################//
//### Public Functions ###//
//########################//

// RandomString generates a random string.
func RandomString(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

// UnmarshalValues splits two values from a single string.
// This function is chainable to extract multiple values.
func UnmarshalValues(data string) (first, second string, err error) {
	// Find the delimiter.
	pos := strings.Index(data, delimiter)
	if pos < 0 {
		err = fmt.Errorf("unmarshal values: no delimiter found: '%s'", data)
		return
	}

	// Extract the value length integer of the first value.
	l, err := strconv.Atoi(data[:pos])
	if err != nil {
		err = fmt.Errorf("invalid value length: '%s'", data[:pos])
		return
	}

	// Remove the value length from the data string.
	data = data[pos+1:]

	// Validate the value length.
	if l < 0 || l > len(data) {
		err = fmt.Errorf("invalid value length: out of bounds: '%v'", l)
		return
	}

	// Split the first value from the second.
	first = data[:l]
	second = data[l:]

	return
}

// MarshalValues joins two values into a single string.
// They can be decoded by the UnmarshalValues function.
func MarshalValues(first, second string) string {
	return strconv.Itoa(len(first)) + delimiter + first + second
}

// RemoteAddress returns the IP address of the request.
// If the X-Forwarded-For or X-Real-Ip http headers are set, then
// they are used to obtain the remote address.
// The boolean is true, if the remote address is obtained using the
// request RemoteAddr() method.
func RemoteAddress(r *http.Request) (string, bool) {
	hdr := r.Header

	// Try to obtain the ip from the X-Forwarded-For header
	ip := hdr.Get("X-Forwarded-For")
	if ip != "" {
		// X-Forwarded-For is potentially a list of addresses separated with ","
		parts := strings.Split(ip, ",")
		if len(parts) > 0 {
			ip = strings.TrimSpace(parts[0])

			if ip != "" {
				return ip, false
			}
		}
	}

	// Try to obtain the ip from the X-Real-Ip header
	ip = strings.TrimSpace(hdr.Get("X-Real-Ip"))
	if ip != "" {
		return ip, false
	}

	// Fallback to the request remote address
	return RemovePortFromRemoteAddr(r.RemoteAddr), true
}

// RemovePortFromRemoteAddr removes the port if present from the remote address.
func RemovePortFromRemoteAddr(remoteAddr string) string {
	pos := strings.LastIndex(remoteAddr, ":")
	if pos < 0 {
		return remoteAddr
	}

	return remoteAddr[:pos]
}
