package addsvcdemo

// Provides server-side bindings for the HTTP transport
// Utilizes the transport/http.Server

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"encoding/json"

	"golang.org/x/net/context"
	stdopentracing "github.com/opentracing/opentracing-go"
	transport "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/tracing/opentracing"
)

// Returns a handler that makes a set of
// endpoints available on predefined paths
func MakeHTTPHandler(
	ctx context.Context,
	endpoints Endpoints, // Endpoints is defined in endpoints.go
	tracer stdopentracing.Tracer,
	logger log.Logger,
) http.Handler {
	options := []transport.ServerOption{
		transport.ServerErrorEncoder(errorEncoder),
		transport.ServerErrorLogger(logger),
	}
	m := http.NewServeMux()
	m.Handle("/sum", transport.NewServer(
		ctx,
		endpoints.SumEndpoint,
		DecodeHTTPSumRequest,
		EncodeHTTPGenericResponse,
		append(options, transport.ServerBefore(opentracing.FromHTTPRequest(tracer, "Sum", logger)))...,
	))
	return m
}

// Used for decoding an error
type errorWrapper struct {
	Error string `json:"error"`
}

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	code := http.StatusInternalServerError
	msg := err.Error()

	switch err {
	case ErrIntOverflow: // ErrIntOverflow is defined in service.go
		code = http.StatusBadRequest
	}

	w.WriteHeader(code)
	json.NewEncoder(w).Encode(errorWrapper{Error: msg})
}

func errorDecoder(r *http.Response) error {
	var w errorWrapper
	if err := json.NewDecoder(r.Body).Decode(&w); err != nil {
		return err
	}
	return errors.New(w.Error)
}

// This is a transport/http.DecodeRequestFunc that decodes
// a JSON-encoded sum request from the HTTP request body
func DecodeHTTPSumRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req sumRequest // sumRequest is defined in endpoints.go
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

// Is a transport/http.DecodeResponseFunc that decodes
// a JSON-encoded sum response from the HTTP response body
// Decode error message from the response body for non-200 status codes
func DecodeHTTPSumResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errorDecoder(r)
	}
	var resp sumResponse // sumResponse is defined in endpoints.go
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

// transport/http.EncodeRequestFunc that JSON-encodes any request to the request body
func EncodeHTTPGenericRequest(_ context.Context, r *http.Request, request interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(&buf)
	return nil
}

// transport/http.EncodeResponseFunc that encodes the response as JSON to the response writer
func EncodeHTTPGenericResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}
