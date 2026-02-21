package interceptors

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	grpcRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_requests_total",
			Help: "Total number of gRPC requests processed, partitioned by method and status code.",
		},
		[]string{"method", "status"},
	)

	grpcRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_request_duration_seconds",
			Help:    "Histogram of gRPC request durations in seconds, partitioned by method.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)
)

func init() {
	prometheus.MustRegister(grpcRequestsTotal)
	prometheus.MustRegister(grpcRequestDuration)
}

// UnaryMetricsInterceptor creates a gRPC unary interceptor that records
// Prometheus metrics for each request: a counter for total requests and
// a histogram for request duration.
func UnaryMetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		st, _ := status.FromError(err)

		grpcRequestsTotal.WithLabelValues(info.FullMethod, st.Code().String()).Inc()
		grpcRequestDuration.WithLabelValues(info.FullMethod).Observe(duration.Seconds())

		return resp, err
	}
}
