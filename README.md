# HTTP Log

[![Go Reference](https://pkg.go.dev/badge/github.com/hekmon/httplog.svg)](https://pkg.go.dev/github.com/hekmon/httplog)

HTTP Log provides a HTTP middleware that use the provided [structured logger](https://go.dev/blog/slog) to log HTTP requests and response within a Go HTTP server.
While many web frameworks already provide this, this lightweight package is useful for those who want to stick to a KISS codebase and the standard library as muxer/server.

The middleware will:

* Pre
  * Generate a uniq request ID used by all log calls within the middleware
  * Logs basic info about the incoming request (host, method, URI, client IP & headers).
  * If logger level is set to `Debug` it will also dump and log the body up to a certain size while still making the body available thru the original request.
  * Prepare a response catcher (available as a separate package if you want to use only this part)
    * This custom catcher also automatically flush data if the content type is a streaming type, see the [catcherflusher](https://pkg.go.dev/github.com/hekmon/httplog/catcherflusher) sub package for more informations.
  * Pass the uniq request ID within the request context
* Call the next middleware
* Post
  * Log the status code (and status) and duration of the request.
  * If logger level is set to `Debug` it will also dump and log the response body up to a certain size.

## Example

```go
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/hekmon/httplog"
)

var (
	// Global logger
	logger *slog.Logger
)

func main() {
	// Initiate the structured main logger.
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create the HTTP structured logger inheriting the main slogger
	httplogger := httplog.New(logger)

	// Setup mux and server
	http.HandleFunc("/", httplogger.Log(ActualHandlerFunc))

	// Start the server
	if err := http.ListenAndServe(":80", nil); err != nil {
		panic(err)
	}
}

func ActualHandlerFunc(w http.ResponseWriter, r *http.Request) {
	// Retreive the request ID from context
	// Thanks to the httplog.ReqIDKey custom key,
	// we are certain that we are getting the httplog reqid, which is a uint64
	reqID := r.Context().Value(httplog.ReqIDKey).(uint64)

	// Setup a local logger that will always print out the request ID
	logger := logger.With(slog.Uint64(httplog.ReqIDKeyName, reqID))

	/*
		do stuff
	*/

	// Let's use our local logger
	logger.Debug("this message will have the request id automatically attached to it")

	fmt.Fprintf(w, "Hello request %d!\n", reqID)
}
```

### Output

#### Client

```
$ curl http://127.0.0.1
Hello request 1!
$ curl http://127.0.0.1
Hello request 2!
$
```

#### Server

```
time=2024-06-04T11:14:56.524+02:00 level=INFO msg="HTTP request received" request_id=1 host=127.0.0.1 method=GET URI=/ client=127.0.0.1:64983 headers="map[Accept:[*/*] User-Agent:[curl/8.6.0]]"
time=2024-06-04T11:14:56.525+02:00 level=DEBUG msg="this message will have the request id automatically attached to it" request_id=1
time=2024-06-04T11:14:56.525+02:00 level=INFO msg="HTTP request handled" request_id=1 response_code=200 response_status=OK response_time=452.417µs
time=2024-06-04T11:14:56.525+02:00 level=DEBUG msg="HTTP response" request_id=1 response_body="Hello request 1!\n" response_size=17
time=2024-06-04T11:14:58.761+02:00 level=INFO msg="HTTP request received" request_id=2 host=127.0.0.1 method=GET URI=/ client=127.0.0.1:64984 headers="map[Accept:[*/*] User-Agent:[curl/8.6.0]]"
time=2024-06-04T11:14:58.761+02:00 level=DEBUG msg="this message will have the request id automatically attached to it" request_id=2
time=2024-06-04T11:14:58.761+02:00 level=INFO msg="HTTP request handled" request_id=2 response_code=200 response_status=OK response_time=103.541µs
time=2024-06-04T11:14:58.761+02:00 level=DEBUG msg="HTTP response" request_id=2 response_body="Hello request 2!\n" response_size=17
```
