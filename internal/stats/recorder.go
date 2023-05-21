package stats

import (
	"bufio"
	"errors"
	"net"
	"net/http"
)

// ResponseWriter defines an interface that custom response writers can implement
type ResponseWriter interface {
	http.ResponseWriter          // Inherits http.ResponseWriter interface
	http.Flusher                 // Inherits http.Flusher interface for flushing writes
	http.Pusher                  // Inherits http.Pusher interface for HTTP/2 server push
	http.Hijacker                // Inherits http.Hijacker interface for hijacking the connection
	Status() int                 // Gets the status of the response
	Written() bool               // Checks whether the response has been written to
	Size() int                   // Gets the size of the response body
	Before(func(ResponseWriter)) // Sets a function that will be executed before the response is written
}

// beforeFunc defines a function that is called before the ResponseWriter has been written to
type beforeFunc func(ResponseWriter)

// responseRecorder is an implementation of http.ResponseWriter that keeps track of its HTTP status
// code and body size
type responseRecorder struct {
	http.ResponseWriter              // Inherits http.ResponseWriter interface
	status              int          // The HTTP status code
	size                int          // The size of the response body
	beforeFuncs         []beforeFunc // A slice of functions to be called before the response is written
	written             bool         // A boolean indicating whether the response has been written to
}

// NewResponseRecorder creates a new responseRecorder that wraps the provided http.ResponseWriter
func NewResponseRecorder(w http.ResponseWriter, statusCode int) ResponseWriter {
	return &responseRecorder{ResponseWriter: w, status: statusCode}
}

// WriteHeader writes an HTTP response header with the provided status code
func (r *responseRecorder) WriteHeader(code int) {
	r.written = true
	r.ResponseWriter.WriteHeader(code)
	r.status = code
}

// Flush flushes the ResponseWriter if it is a http.Flusher
func (r *responseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Status returns the HTTP status of the response
func (r *responseRecorder) Status() int {
	return r.status
}

// Write writes the provided data to the ResponseWriter and updates the size
func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.Written() {
		r.WriteHeader(http.StatusOK)
	}

	size, err := r.ResponseWriter.Write(b)
	r.size += size
	return size, err
}

// Written checks if the ResponseWriter has been written
func (r *responseRecorder) Written() bool {
	return r.status != 0
}

// Push initiates an HTTP/2 server push, sends a synthetic request using the given target and options
func (r *responseRecorder) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := r.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return errors.New("the ResponseWriter doesn't support the Pusher interface")
}

// Hijack supports websockets and other upgrade protocols
func (r *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if !r.written {
		r.status = 0
	}
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("the ResponseWriter doesn't support the Hijacker interface")
	}
	return hijacker.Hijack()
}

// Size returns the size of the response body
func (r *responseRecorder) Size() int {
	return r.size
}

// Before adds a function to be called before the ResponseWriter is written to
func (r *responseRecorder) Before(before func(ResponseWriter)) {
	r.beforeFuncs = append(r.beforeFuncs, before)
}
