# Change Log
All notable changes to this project will be documented in this file. This project follows the [Semantic Versioning](http://semver.org/).

## 1.8.0 - 2015-11-26
- Added support for the Cross-Origin Resource Sharing (CORS) mechanism. Added new option EnableCORS.

## 1.7.0 - 2015-11-15
- Added CheckOrigin option.

## 1.6.0 - 2015-11-15
- Added method to obtain the socket ID of the client-side.
- Websocket: don't log messages if the 1005 close code is received (CloseNoStatusReceived).

## 1.5.0 - 2015-11-04
- Added GetSocket method to server.
- Some minor improvements.

## 1.4.1 - 2015-10-19
- websocket: removed dirty hack to fix unnecessary log messages and fixed this with the websocket close code.

## 1.4.0 - 2015-10-18
- Implemented socket ClosedChan method.
- Reformatted README.

## 1.3.1 - 2015-09-09
- Javascript client: code cleanup and small fixes (JSHint).
- Updated Version
- Added semantic version check with backward compatibility check.
- Implemented ajax poll timeout and close handling.
- Suppress unnecessary websocket close error message.

## 1.3.0 - 2015-09-02
- Restructured backend sockets.
- Moved glue methods into a server struct.
- New socket ID generation.
- Added support to set custom HTTP base URLs.
- Added server options.
- HTTP server is now started by the glue server.
- Added support for custom HTTP multiplexers.

## 1.2.0 - 2015-08-11
- Several small fixes and improvements.

### Added
- Added channel support to communicate in different channels.
- Sockets are added to a map of active sockets.
- A list of active sockets can be retrieved with glue.Sockets().
- Added unique ID for each socket.
- Added Release function to block new incoming connections and to close all current connected sockets.
- Added socket.Value interface to store custom data.
- Added glue socket protocol versions check during socket connection initialization.

### Removed
- Removed discard_send_buffers from the frontend library. Use the discard callback in the send function instead.

## 1.1.1 - 2015-07-21
### Added
- Added socket OnRead event. Either use Read or OnRead.
- Added internal read handler with locking...
- Updated README.

## 1.1.0 - 2015-07-15
### Added
- Updated TODO.
- Added Changelog.
- Added blocking socket Read method.

### Removed
- Removed socket OnRead event.
