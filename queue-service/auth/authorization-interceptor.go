package auth

import (
	"context"
	"github.com/harryrose/godm/log"
	"github.com/harryrose/godm/log/keys"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	AuthorizationKey = "authorization"
)

func AuthorizationInterceptor(key string) grpc.UnaryServerInterceptor {
	authenticate := func(ctx context.Context) error {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Warnw("authorization fail", keys.Error, "no metadata in context")
			return status.Error(codes.Unauthenticated, "Unauthenticated")
		}
		values := md.Get(AuthorizationKey)
		if l := len(values); l != 1 {
			log.Warnw("authorization fail", keys.Error, "unexpected number of values", keys.Expected, 1, keys.Got, l)
			return status.Error(codes.Unauthenticated, "Unauthenticated")
		}
		if values[0] != key {
			log.Warnw("authorization fail", keys.Error, "incorrect key")
			return status.Error(codes.Unauthenticated, "Unauthenticated")
		}
		return nil
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if err := authenticate(ctx); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}
