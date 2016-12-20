package main

import (
	"flag"
	"fmt"
	"github.com/mtso/addsvcdemo"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	stdopentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-kit/kit/tracing/opentracing"
)

const (
	local_port = "3000"
)

func main() {
	var (
		debugAddr = flag.String("debug.addr", ":8080", "Debug and metrics listen address")
		httpAddr  = flag.String("http.addr", ":8081", "HTTP listen address")
	)
	flag.Parse()

	// logging domain.
	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stdout)
		logger = log.NewContext(logger).With("ts", log.DefaultTimestampUTC)
		logger = log.NewContext(logger).With("caller", log.DefaultCaller)
	}
	logger.Log("msg", "hello")
	defer logger.Log("msg", "goodbye")

	// Metrics domain.
	var ints metrics.Counter
	{
		// Business metrics.
		ints = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "addsvcemo",
			Name:      "integers_summed",
			Help:      "Total count of integers summed via the Sum method.",
		}, []string{})
	}
	var duration metrics.Histogram
	{
		// Transport level metrics
		duration = prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "addsvcemo",
			Name:      "request_duration_ns",
			Help:      "Request duration in nanoseconds.",
		}, []string{"method", "success"})
	}

	// Tracing domain.
	var tracer stdopentracing.Tracer
	{
		logger := log.NewContext(logger).With("tracer", "none")
		logger.Log()
		tracer = stdopentracing.GlobalTracer()
	}

	// Business domain.
	var service addsvcdemo.Service
	{
		service = addsvcdemo.NewStatelessService()
		service = addsvcdemo.ServiceLoggingMiddleware(logger)(service)
		service = addsvcdemo.ServiceInstrumentingMiddleware(ints)(service)
	}

	// Endpoint domain.
	var sumEndpoint endpoint.Endpoint
	{
		sumDuration := duration.With("method", "Sum")
		sumLogger := log.NewContext(logger).With("method", "Sum")

		sumEndpoint = addsvcdemo.MakeSumEndpoint(service)
		sumEndpoint = opentracing.TraceServer(tracer, "Sum")(sumEndpoint)
		sumEndpoint = addsvcdemo.EndpointInstrumentingMiddleware(sumDuration)(sumEndpoint)
		sumEndpoint = addsvcdemo.EndpointLoggingMiddleware(sumLogger)(sumEndpoint)
	}
	endpoints := addsvcdemo.Endpoints{
		SumEndpoint: sumEndpoint,
	}

	// Mechanical domain.
	errc := make(chan error)
	ctx := context.Background()

	// Interrupt handler.
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	// Debug listener
	go func() {
		logger := log.NewContext(logger).With("transport", "debug")
		// TODO
		m := http.NewServeMux()
		m.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
		m.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		m.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		m.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		m.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
		m.Handle("/metrics", stdprometheus.Handler())

		logger.Log("addr", *debugAddr)
		errc <- http.ListenAndServe(*debugAddr, m)
	}()

	// HTTP transport.
	go func() {
		logger := log.NewContext(logger).With("transport", "HTTP")
		h := addsvcdemo.MakeHTTPHandler(ctx, endpoints, tracer, logger)
		logger.Log("addr", *httpAddr) // TODO FIX THIS
		errc <- http.ListenAndServe(*debugAddr, h)
	}()

	// Run
	logger.Log("exit", <-errc)
}
