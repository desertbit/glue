Change Log
==========

All notable changes to this project will be documented in this file.

1.2.0 - 2015-08-11
------------------

-	Several small fixes and improvements.

### Added

-	Added channel support to communicate in different channels.
-	Sockets are added to a map of active sockets.
-	A list of active sockets can be retrieved with glue.Sockets().
-	Added unique ID for each socket.
-	Added Release function to block new incoming connections and to close all current connected sockets.
-	Added socket.Value interface to store custom data.
-	Added glue socket protocol versions check during socket connection initialization.

### Removed

-	Removed discard_send_buffers from the frontend library. Use the discard callback in the send function instead.

1.1.1 - 2015-07-21
------------------

### Added

-	Added socket OnRead event. Either use Read or OnRead.
-	Added internal read handler with locking...
-	Updated README.

1.1.0 - 2015-07-15
------------------

### Added

-	Updated TODO.
-	Added Changelog.
-	Added blocking socket Read method.

### Removed

-	Removed socket OnRead event.
