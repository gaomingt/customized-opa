package log

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GrpcLogger(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	start := time.Now()
	res, err := handler(ctx, req)
	duration := time.Since(start)
	statusCode := codes.Unknown
	if st, ok := status.FromError(err); ok {
		statusCode = st.Code()
	}
	event := log.Info()
	if err != nil {
		event = log.Error().Err(err)
	}
	loggerHelper(event, grpcProtocol, "", info.FullMethod, statusCode.String(), int(statusCode), duration)
	return res, err
}
