package httpinspect

import (
	"bytes"
	"net/http"
)

// StreamingContentTypes is a list of content types that are considered
// to be streaming. If the content type of a ResponseWriter is in this list,
// WriteHeader and any subsequent writes will be flushed directly to the client.
var StreamingContentTypes = []string{
	"text/event-stream",
	"application/x-ndjson",
}

// NewResponseWriter returns a new ResponseWriter that wraps the provided
// http.ResponseWriter. It will capture the HTTP response code and also capture
// all writes if captureBody is set to true.
func NewResponseWriter(w http.ResponseWriter, captureBody bool) (rw *ResponseWriter) {
	if w == nil {
		return
	}
	rw = &ResponseWriter{
		wrapped: w,
	}
	if f, ok := w.(http.Flusher); ok {
		rw.flusher = f
	}
	if captureBody {
		rw.body = new(bytes.Buffer)
	}
	return
}

// ResponseWriter is a wrapper around http.ResponseWriter that captures the
// response code and body. It also provides auto-flushing of the underlying
// writer if the content type is in the StreamingContentTypes list.
type ResponseWriter struct {
	wrapped http.ResponseWriter
	flusher http.Flusher
	code    int
	body    *bytes.Buffer
}

// GetResponseCode returns the HTTP response code that was written to the
// underlying http.ResponseWriter.
func (rw *ResponseWriter) GetResponseCode() int {
	return rw.code
}

// GetBody returns the body that was written to the underlying
// http.ResponseWriter. Note that captureBody must be set to true when
// creating the ResponseWriter for the body to be captured.
func (rw *ResponseWriter) GetBody() []byte {
	if rw.body == nil {
		return nil
	}
	return rw.body.Bytes()
}

/*
	Implements http.ResponseWriter
*/

// Header returns the header map that will be sent by
// [ResponseWriter.WriteHeader]. The [Header] map also is the mechanism with which
// [Handler] implementations can set HTTP trailers.
//
// Changing the header map after a call to [ResponseWriter.WriteHeader] (or
// [ResponseWriter.Write]) has no effect unless the HTTP status code was of the
// 1xx class or the modified headers are trailers.
//
// There are two ways to set Trailers. The preferred way is to
// predeclare in the headers which trailers you will later
// send by setting the "Trailer" header to the names of the
// trailer keys which will come later. In this case, those
// keys of the Header map are treated as if they were
// trailers. See the example. The second way, for trailer
// keys not known to the [Handler] until after the first [ResponseWriter.Write],
// is to prefix the [Header] map keys with the [TrailerPrefix]
// constant value.
//
// To suppress automatic response headers (such as "Date"), set
// their value to nil.
func (rw *ResponseWriter) Header() http.Header {
	return rw.wrapped.Header()
}

// Write writes the data to the connection as part of an HTTP reply.
//
// If [ResponseWriter.WriteHeader] has not yet been called, Write calls
// WriteHeader(http.StatusOK) before writing the data. If the Header
// does not contain a Content-Type line, Write adds a Content-Type set
// to the result of passing the initial 512 bytes of written data to
// [DetectContentType]. Additionally, if the total size of all written
// data is under a few KB and there are no Flush calls, the
// Content-Length header is added automatically.
//
// Depending on the HTTP protocol version and the client, calling
// Write or WriteHeader may prevent future reads on the
// Request.Body. For HTTP/1.x requests, handlers should read any
// needed request body data before writing the response. Once the
// headers have been flushed (due to either an explicit Flusher.Flush
// call or writing enough data to trigger a flush), the request body
// may be unavailable. For HTTP/2 requests, the Go HTTP server permits
// handlers to continue to read the request body while concurrently
// writing the response. However, such behavior may not be supported
// by all HTTP/2 clients. Handlers should read before writing if
// possible to maximize compatibility.
func (rw *ResponseWriter) Write(data []byte) (int, error) {
	if rw.code == 0 {
		// wrapped write will do it itself on the wrapped response if we don't
		// but we won't know about it: let's do it ourself.
		rw.WriteHeader(http.StatusOK)
	}
	if rw.body != nil {
		rw.body.Write(data)
	}
	if rw.flusher != nil {
		defer rw.flusher.Flush()
	}
	return rw.wrapped.Write(data)
}

// WriteHeader sends an HTTP response header with the provided
// status code.
//
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to
// send error codes or 1xx informational responses.
//
// The provided code must be a valid HTTP 1xx-5xx status code.
// Any number of 1xx headers may be written, followed by at most
// one 2xx-5xx header. 1xx headers are sent immediately, but 2xx-5xx
// headers may be buffered. Use the Flusher interface to send
// buffered data. The header map is cleared when 2xx-5xx headers are
// sent, but not with 1xx headers.
//
// The server will automatically send a 100 (Continue) header
// on the first read from the request body if the request has
// an "Expect: 100-continue" header.
//
// If original ResponseWriter was an http.Flusher and current
// content type is one of StreamingContentTypes header and
// any subsequent Write will be flushed to the client and not buffered.
func (rw *ResponseWriter) WriteHeader(statusCode int) {
	// Forward and save the status code
	rw.wrapped.WriteHeader(statusCode)
	rw.code = statusCode
	// Detect content types that are streaming and require flushing
	if rw.flusher == nil {
		// underlying response writer does not support flushing, abort
		return
	}
	var shouldFlush bool
	responseContentType := rw.Header().Get("Content-Type")
	for _, ct := range StreamingContentTypes {
		if ct == responseContentType {
			shouldFlush = true
			break
		}
	}
	if shouldFlush {
		defer rw.flusher.Flush()
	} else {
		// nullyfy flusher to prevent further flushes
		rw.flusher = nil
	}
}

/*
	Implements http.Flusher
*/

// Flush sends any buffered data to the client if the underlying response
// writer supports flushing. If not supported, this method does nothing.
//
// Note that flush is automatically called by Write and WriteHeader methods,
// if the underlying response writer supports it (i.e., http.ResponseWriter)
// and the content type of response is one of StreamingContentTypes.
func (rw *ResponseWriter) Flush() {
	if rw.flusher != nil {
		rw.flusher.Flush()
	}
}
