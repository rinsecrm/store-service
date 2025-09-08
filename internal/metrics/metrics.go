package metrics

import (
	"context"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// gRPC server metrics
var (
	grpcServerCallsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_server_calls_total",
			Help: "Total number of gRPC server calls",
		},
		[]string{"service", "method", "status_code"},
	)

	grpcServerCallDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_server_call_duration_seconds",
			Help:    "gRPC server call duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method"},
	)

	grpcServerCallsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "grpc_server_calls_in_flight",
			Help: "Current number of gRPC server calls being processed",
		},
	)

	// Business metrics
	storeOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "store_operations_total",
			Help: "Total number of store operations",
		},
		[]string{"operation"},
	)

	storeOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "store_operation_duration_seconds",
			Help:    "Store operation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	storeOperationErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "store_operation_errors_total",
			Help: "Total number of store operation errors",
		},
		[]string{"operation"},
	)
)

// init registers all metrics
func init() {
	prometheus.MustRegister(grpcServerCallsTotal)
	prometheus.MustRegister(grpcServerCallDuration)
	prometheus.MustRegister(grpcServerCallsInFlight)
	prometheus.MustRegister(storeOperationsTotal)
	prometheus.MustRegister(storeOperationDuration)
	prometheus.MustRegister(storeOperationErrors)
}

// UnaryServerInterceptor provides Prometheus metrics for gRPC unary calls
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Extract service and method names from FullMethod (format: /package.Service/Method)
		parts := strings.Split(info.FullMethod, "/")
		var service, method string
		if len(parts) >= 3 {
			service = parts[1] // package.Service
			method = parts[2]  // Method
		} else {
			service = "unknown"
			method = info.FullMethod
		}

		// Increment in-flight calls counter
		grpcServerCallsInFlight.Inc()
		defer grpcServerCallsInFlight.Dec()

		// Call the handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)
		durationSeconds := float64(duration) / float64(time.Second)

		// Get gRPC status code
		var code string
		if err != nil {
			st, _ := status.FromError(err)
			code = st.Code().String()
		} else {
			code = "OK"
		}

		// Record metrics
		grpcServerCallsTotal.WithLabelValues(service, method, code).Inc()
		grpcServerCallDuration.WithLabelValues(service, method).Observe(durationSeconds)

		return resp, err
	}
}

// Business metrics functions
func RecordStoreOperation(operation string) {
	storeOperationsTotal.WithLabelValues(operation).Inc()
}

func RecordStoreOperationDuration(operation string, duration float64) {
	storeOperationDuration.WithLabelValues(operation).Observe(duration)
}

func RecordStoreOperationError(operation string) {
	storeOperationErrors.WithLabelValues(operation).Inc()
}

// MetricsHandler returns the Prometheus metrics handler
func MetricsHandler() *promhttp.Handler {
	return &promhttp.Handler{}
}
