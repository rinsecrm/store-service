package canaryctx

import (
	"context"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	CanaryHeader = "X-Canary"
)

type contextKey string

const canaryKey contextKey = "canary"

// FromContext extracts the canary PR number from context
func FromContext(ctx context.Context) (string, bool) {
	canary, ok := ctx.Value(canaryKey).(string)
	return canary, ok
}

// WithCanary adds canary PR number to context
func WithCanary(ctx context.Context, canary string) context.Context {
	return context.WithValue(ctx, canaryKey, canary)
}

// IsValidCanary checks if the canary value is a valid PR number (digits only)
func IsValidCanary(canary string) bool {
	if canary == "" {
		return false
	}
	_, err := strconv.Atoi(canary)
	return err == nil
}

// UnaryServerInterceptor extracts X-Canary from incoming gRPC metadata
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get(CanaryHeader); len(values) > 0 {
				canary := strings.TrimSpace(values[0])
				if IsValidCanary(canary) {
					ctx = WithCanary(ctx, canary)
				}
			}
		}
		return handler(ctx, req)
	}
}

// UnaryClientInterceptor adds X-Canary to outgoing gRPC metadata
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if canary, ok := FromContext(ctx); ok {
			md := metadata.Pairs(CanaryHeader, canary)
			ctx = metadata.NewOutgoingContext(ctx, md)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
