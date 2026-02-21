package interceptors

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryLoggingInterceptor creates a gRPC unary interceptor that logs
// method name, duration, and error status for each request.
func UnaryLoggingInterceptor(log *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		log.Info("gRPC request started",
			zap.String("method", info.FullMethod),
		)

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		st, _ := status.FromError(err)

		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.String("status_code", st.Code().String()),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			log.Error("gRPC request failed", fields...)
		} else {
			log.Info("gRPC request completed", fields...)
		}

		return resp, err
	}
}
