package addsvcdemo

import (
	"fmt"
	"time"
	"golang.org/x/net/context"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/log"
)

type Endpoints struct {
	SumEndpoint endpoint.Endpoint
}

// Sum implements Service
func (e Endpoints) Sum(ctx context.Context, x, y int) (int, error) {
	request := sumRequest{ X: x, Y: y }
	response, err := e.SumEndpoint(ctx, request)
	if err != nil {
		return 0, err
	}
	return response.(sumResponse).V, response.(sumResponse).Err
}

// Returns an endpoint that invokes Sum on the service
func MakeSumEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		sumRequest:= request.(sumRequest)
		v, err := s.Sum(ctx, sumRequest.X, sumRequest.Y)
		if err == ErrIntOverflow {
			return nil, error
		}
		return sumResponse{
			V: v,
			Err: err
		}
	}
}

// Returns an endpoint middleware that logs the duration 
// of each invocation, and resulting error, if any
func EndpointLoggingMiddleware(logger log.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			
			defer func(begin time.Time) {
				logger.Log("error", err, "took", time.Since(begin))
			}(time.Now())
			return next(ctx, request)
		}
	}
}

// Returns the endpoint middleware that records
// the duration of each invocation to the passed histogram.
// The middlware adds a field: "success", for true if no error is returned
func EndpointInstrumentingMiddleware(duration metrics.Histogram) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {

			defer func(begin time.Time) {
				duration.With("success", fmt.Sprint(err == nil)).Observe(time.Since(begin).Seconds())
			}(time.Now())
			return next(ctx, request)
		}
	}
}

type sumRequest struct { X, Y int }

type sumResponse struct {
	V int
	Err error
}

